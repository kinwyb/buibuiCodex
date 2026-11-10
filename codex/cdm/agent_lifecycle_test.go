package cdm

import (
	"errors"
	"testing"
	"time"

	"github.com/kinwyb/buibuiCodex/core/config"
)

func TestAgentStopWaitsForActiveTurn(t *testing.T) {
	a := NewAgent(&config.AgentConfig{Name: "codex"}, 0)
	if err := a.beginTurn(); err != nil {
		t.Fatalf("登记活跃 turn 失败: %v", err)
	}

	done := make(chan error, 1)
	go func() {
		done <- a.Stop()
	}()

	select {
	case err := <-done:
		t.Fatalf("活跃 turn 未结束时 Stop 不应返回: %v", err)
	case <-time.After(30 * time.Millisecond):
	}

	a.endTurn()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("正常 drain 后 Stop 不应报错: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("活跃 turn 结束后 Stop 应及时返回")
	}

	if err := a.beginTurn(); !errors.Is(err, errAgentStopping) {
		t.Fatalf("Stop 后不应再接收新 turn，实际 %v", err)
	}
}

func TestAgentStopForcesAfterDrainTimeout(t *testing.T) {
	a := NewAgent(&config.AgentConfig{Name: "codex"}, 0)
	a.stopDrain = 20 * time.Millisecond
	a.stopForceWait = 20 * time.Millisecond
	if err := a.beginTurn(); err != nil {
		t.Fatalf("登记活跃 turn 失败: %v", err)
	}
	defer a.endTurn()

	start := time.Now()
	err := a.Stop()
	if !errors.Is(err, errAgentStopping) {
		t.Fatalf("超时停止应返回中断错误，实际 %v", err)
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("超时停止不应长时间阻塞，实际耗时 %s", elapsed)
	}
	select {
	case <-a.stopSignal.Done():
	default:
		t.Fatal("超时停止后应解除 Prompt 的停止等待")
	}
}
