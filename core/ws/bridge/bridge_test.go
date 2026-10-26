package bridge

import (
	"testing"

	"github.com/kinwyb/buibuiCodex/core/types"
	"github.com/kinwyb/buibuiCodex/core/ws/protocol"
)

func TestAcceptChannel(t *testing.T) {
	b := New(Options{Channel: "ws"})
	if !b.accept("ws") {
		t.Fatal("应接受 ws 渠道")
	}
	if b.accept("wxcom") {
		t.Fatal("默认不应转发其他渠道")
	}

	all := New(Options{Channel: "ws", ForwardAll: true})
	if !all.accept("wxcom") {
		t.Fatal("开启 ForwardAll 后应接受任意渠道")
	}
}

func TestConvertMessageCompleted(t *testing.T) {
	b := New(Options{Channel: "ws"})
	ev := &types.Event{
		EventID:    "e1",
		SessionID:  "s1",
		ReqID:      "r1",
		Type:       types.EventMessageCompleted,
		Channel:    "ws",
		ChunkIndex: 5,
		Message: &types.Message{
			Role:    types.RoleAssistant,
			Content: "完成",
		},
	}
	env := b.convert(ev)
	if env.Kind != protocol.KindAgent || env.Type != string(types.EventMessageCompleted) {
		t.Fatalf("信封类型不符合预期: %s/%s", env.Kind, env.Type)
	}
	if env.SessionID != "s1" || env.ReqID != "r1" || env.Chunk != 5 {
		t.Fatalf("信封会话字段不符合预期: %+v", env)
	}
	var data protocol.AgentData
	if err := env.DecodeData(&data); err != nil {
		t.Fatalf("解析载荷失败: %v", err)
	}
	if data.Content != "完成" || data.Role != "assistant" || data.Event != string(types.EventMessageCompleted) {
		t.Fatalf("载荷不符合预期: %+v", data)
	}
	if data.Delta {
		t.Fatal("completed 必须是全量快照，Delta 应为 false，否则前端不会覆盖收敛")
	}
}

func TestConvertError(t *testing.T) {
	b := New(Options{Channel: "ws"})
	env := b.convert(&types.Event{
		SessionID: "s1",
		Type:      types.EventError,
		Channel:   "ws",
		Message:   &types.Message{Role: types.RoleSystem, Content: "失败：超时"},
	})
	var data protocol.AgentData
	if err := env.DecodeData(&data); err != nil {
		t.Fatalf("解析载荷失败: %v", err)
	}
	if data.Error == nil || data.Error.Message != "失败：超时" {
		t.Fatalf("错误信息不符合预期: %+v", data.Error)
	}
}

func TestConvertApproval(t *testing.T) {
	b := New(Options{Channel: "ws"})
	env := b.convert(&types.Event{
		SessionID: "s1",
		Type:      types.EventApproval,
		Channel:   "ws",
		Message: &types.Message{
			Role:       types.RoleAssistant,
			Content:    "是否允许执行",
			ApprovalID: "ap-1",
		},
	})
	var data protocol.AgentData
	if err := env.DecodeData(&data); err != nil {
		t.Fatalf("解析载荷失败: %v", err)
	}
	if data.Approval == nil || data.Approval.ID != "ap-1" {
		t.Fatalf("审批信息不符合预期: %+v", data.Approval)
	}
}

func TestConvertDeltaFlag(t *testing.T) {
	b := New(Options{Channel: "ws"})
	env := b.convert(&types.Event{
		SessionID: "s1",
		Type:      types.EventMessageDelta,
		Channel:   "ws",
		Message:   &types.Message{Role: types.RoleAssistant, Content: "片段", IsDetla: true},
	})
	var data protocol.AgentData
	if err := env.DecodeData(&data); err != nil {
		t.Fatalf("解析载荷失败: %v", err)
	}
	if !data.Delta {
		t.Fatal("增量事件应标记 Delta")
	}
}
