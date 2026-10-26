package ws

import (
	"testing"

	"github.com/kinwyb/buibuiCodex/core/config"
	"github.com/kinwyb/buibuiCodex/core/ws/protocol"
)

func TestNewServer(t *testing.T) {
	s := New(Options{Config: &config.WebSocketConfig{Enabled: true}})
	if s == nil {
		t.Fatal("New 返回 nil")
	}
	if s.ClientCount() != 0 || s.SessionCount() != 0 {
		t.Fatalf("新建服务不应有连接: %d/%d", s.ClientCount(), s.SessionCount())
	}
	if n := s.Broadcast(protocol.KindNotice, "ping-all", nil); n != 0 {
		t.Fatalf("无连接时广播应返回 0，实际 %d", n)
	}
	if n := s.Broadcast(protocol.KindClient, "invalid", nil); n != 0 {
		t.Fatalf("client 域不应由服务端广播，实际 %d", n)
	}
}

func TestStopWithoutStart(t *testing.T) {
	s := New(Options{Config: &config.WebSocketConfig{Enabled: true}})
	if err := s.Stop(t.Context()); err != nil {
		t.Fatalf("未启动时停止应返回 nil: %v", err)
	}
}

func TestStartWithoutBus(t *testing.T) {
	s := New(Options{Config: &config.WebSocketConfig{Enabled: true}})
	if err := s.Start(t.Context()); err == nil {
		t.Fatal("缺少消息总线时应返回错误")
	}
}
