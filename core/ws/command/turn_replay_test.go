// 本文件测试 session.turn.replay。

package command

import (
	"context"
	"testing"

	"github.com/kinwyb/buibuiCodex/core/ws/protocol"
)

func TestTurnReplayReturnsFullFrames(t *testing.T) {
	reader := replayReader()
	reg, hub := newTestRegistryWithTurns(nil, nil, reader)
	c := newTestClient(hub)

	env := protocol.NewEnvelope(protocol.KindClient, protocol.TypeTurnReplay).
		WithData(protocol.TurnReplayData{SessionID: "ws_web_s1", TurnID: "u1"})
	reg.Dispatch(context.Background(), c, env)

	raw := <-c.Outbound()
	resp, _ := protocol.DecodeEnvelope(raw)
	if resp.Type != protocol.TypeTurnReplay || resp.ReplyTo != env.ID {
		t.Fatalf("响应类型不符合预期: %s/%s", resp.Type, resp.ReplyTo)
	}
	var result protocol.TurnReplayResult
	if err := resp.DecodeData(&result); err != nil {
		t.Fatalf("解析载荷失败: %v", err)
	}
	if result.TurnID != "u1" || len(result.Frames) != 3 {
		t.Fatalf("单轮回放结果不符合预期: %+v", result)
	}
	if result.Frames[0].Seq != 11 {
		t.Fatalf("帧应回填事件 seq，实际 %d", result.Frames[0].Seq)
	}
	if result.Frames[2].Kind != protocol.KindAgent {
		t.Fatalf("回放帧应与实时同域: %s", result.Frames[2].Kind)
	}
}

func TestTurnReplayRejectsForeignTurn(t *testing.T) {
	reader := replayReader()
	// 轮次属于别人的会话
	reader.turns["t1"][0].SessionID = "ws_bob_s1"
	reg, hub := newTestRegistryWithTurns(nil, nil, reader)
	c := newTestClientAs(hub, "alice")

	env := protocol.NewEnvelope(protocol.KindClient, protocol.TypeTurnReplay).
		WithData(protocol.TurnReplayData{SessionID: "ws_alice_s1", TurnID: "u1"})
	reg.Dispatch(context.Background(), c, env)

	raw := <-c.Outbound()
	resp, _ := protocol.DecodeEnvelope(raw)
	var body protocol.ErrorBody
	_ = resp.DecodeData(&body)
	if body.Code != protocol.CodeForbidden {
		t.Fatalf("跨会话读过程应被拒绝，错误码 %s", body.Code)
	}
}

func TestTurnReplayRejectsMissingTurnID(t *testing.T) {
	reg, hub := newTestRegistryWithTurns(nil, nil, replayReader())
	c := newTestClient(hub)

	env := protocol.NewEnvelope(protocol.KindClient, protocol.TypeTurnReplay).
		WithData(protocol.TurnReplayData{SessionID: "ws_web_s1"})
	reg.Dispatch(context.Background(), c, env)

	raw := <-c.Outbound()
	resp, _ := protocol.DecodeEnvelope(raw)
	var body protocol.ErrorBody
	_ = resp.DecodeData(&body)
	if body.Code != protocol.CodeBadRequest {
		t.Fatalf("缺少 turnID 应返回 bad_request，实际 %s", body.Code)
	}
}

// ---- 对话列表 ----
