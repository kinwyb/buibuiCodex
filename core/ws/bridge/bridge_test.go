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
	if env.EventID != "e1" {
		t.Fatalf("信封应透传事件实体标识，实际 %q", env.EventID)
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

// 同一 item 的多次更新必须共享 eventID：
// 前端按它归并，就不必依赖「先 start 后 completed」的到达顺序。
func TestConvertKeepsEventIDAcrossLifecycle(t *testing.T) {
	b := New(Options{Channel: "ws"})
	for _, typ := range []types.EventType{
		types.EventMessageStart,
		types.EventMessageDelta,
		types.EventMessageCompleted,
	} {
		env := b.convert(&types.Event{
			EventID:   "item-1",
			SessionID: "s1",
			ReqID:     "r1",
			Type:      typ,
			Channel:   "ws",
		})
		if env.EventID != "item-1" {
			t.Fatalf("%s 帧的 eventID 丢失: %q", typ, env.EventID)
		}
	}

	// 系统类事件没有实体标识时应留空，而不是填个假值
	env := b.convert(&types.Event{SessionID: "s1", ReqID: "r1", Type: types.EventFinished, Channel: "ws"})
	if env.EventID != "" {
		t.Fatalf("无实体的事件不应带 eventID，实际 %q", env.EventID)
	}
}

// token 用完统计必须作为独立载荷透出：它是状态而不是文本，
// Delta 恒为 false，前端只更新用量面板、不碰正文。
func TestConvertTokenUsage(t *testing.T) {
	b := New(Options{Channel: "ws"})
	env := b.convert(&types.Event{
		EventID:   "tu_1",
		SessionID: "s1",
		ReqID:     "r1",
		Type:      types.EventTokenUsage,
		Channel:   "ws",
		Message:   &types.Message{Role: types.RoleAssistant},
		Usage: &types.TokenUsage{
			Total:              types.TokenDetail{TotalTokens: 900, InputTokens: 700, OutputTokens: 200},
			Last:               types.TokenDetail{TotalTokens: 120, InputTokens: 100, OutputTokens: 20},
			ModelContextWindow: 200000,
		},
	})

	var data protocol.AgentData
	if err := env.DecodeData(&data); err != nil {
		t.Fatalf("解析载荷失败: %v", err)
	}
	if data.Event != string(types.EventTokenUsage) {
		t.Fatalf("载荷事件类型不符: %q", data.Event)
	}
	if data.Delta {
		t.Fatal("token_usage 不属于增量，Delta 应为 false")
	}
	if data.Content != "" {
		t.Fatalf("token_usage 不应携带正文: %q", data.Content)
	}
	if data.Usage == nil {
		t.Fatal("缺少 usage 载荷")
	}
	if data.Usage.Total.TotalTokens != 900 || data.Usage.Last.TotalTokens != 120 {
		t.Fatalf("用量数据不符: %+v", data.Usage)
	}
	if data.Usage.ModelContextWindow != 200000 {
		t.Fatalf("上下文窗口不符: %d", data.Usage.ModelContextWindow)
	}
	if protocol.IsDelta(protocol.KindAgent, string(types.EventTokenUsage)) {
		t.Fatal("token_usage 不应被当作可丢弃的增量帧")
	}
}
