// 本文件测试 session.delete。

package command

import (
	"context"
	"testing"

	"github.com/kinwyb/buibuiCodex/core/db"
	"github.com/kinwyb/buibuiCodex/core/ws/conn"
	"github.com/kinwyb/buibuiCodex/core/ws/protocol"
)

func deleteRequest(t *testing.T, reg *Registry, c *conn.Client, data protocol.SessionDeleteData) *protocol.Envelope {
	t.Helper()
	env := protocol.NewEnvelope(protocol.KindClient, protocol.TypeSessionDelete).WithData(data)
	reg.Dispatch(context.Background(), c, env)

	select {
	case raw := <-c.Outbound():
		resp, body := protocol.DecodeEnvelope(raw)
		if body != nil {
			t.Fatalf("解析响应失败: %+v", body)
		}
		return resp
	default:
		t.Fatal("未返回响应")
		return nil
	}
}

func TestSessionDeleteMarksOwnedThread(t *testing.T) {
	reader := &fakeTurnReader{threads: []db.SessionThread{{
		ThreadID: "t1", SessionID: "ws_alice_s1", Agent: "codex",
	}}}
	reg, hub := newTestRegistryWithTurns(nil, nil, reader)
	c := newTestClientAs(hub, "alice")

	resp := deleteRequest(t, reg, c, protocol.SessionDeleteData{ThreadID: "t1"})
	if resp.Kind != protocol.KindSystem || resp.Type != protocol.TypeSessionDelete {
		t.Fatalf("响应类型不符合预期: %s/%s", resp.Kind, resp.Type)
	}
	var result protocol.SessionDeleteResult
	if err := resp.DecodeData(&result); err != nil {
		t.Fatalf("解析载荷失败: %v", err)
	}
	if !result.Deleted || result.ThreadID != "t1" {
		t.Fatalf("删除结果不符合预期: %+v", result)
	}
	if len(reader.threadDeleteCalls) != 1 || reader.threadDeleteCalls[0] != (threadDeleteCall{SessionID: "ws_alice_s1", ThreadID: "t1"}) {
		t.Fatalf("应按线程所属会话执行软删除: %+v", reader.threadDeleteCalls)
	}
}

func TestSessionDeleteRejectsForeignThread(t *testing.T) {
	reader := &fakeTurnReader{threads: []db.SessionThread{{
		ThreadID: "t1", SessionID: "ws_bob_s1", Agent: "codex",
	}}}
	reg, hub := newTestRegistryWithTurns(nil, nil, reader)
	c := newTestClientAs(hub, "alice")

	resp := deleteRequest(t, reg, c, protocol.SessionDeleteData{ThreadID: "t1"})
	var body protocol.ErrorBody
	_ = resp.DecodeData(&body)
	if body.Code != protocol.CodeForbidden {
		t.Fatalf("跨用户删除应被拒绝，错误码 %s", body.Code)
	}
	if len(reader.threadDeleteCalls) != 0 {
		t.Fatalf("越权请求不应触发删除: %+v", reader.threadDeleteCalls)
	}
}

func TestSessionDeleteRejectsMissingThreadID(t *testing.T) {
	reg, hub := newTestRegistryWithTurns(nil, nil, &fakeTurnReader{})
	c := newTestClient(hub)

	resp := deleteRequest(t, reg, c, protocol.SessionDeleteData{})
	var body protocol.ErrorBody
	_ = resp.DecodeData(&body)
	if body.Code != protocol.CodeBadRequest {
		t.Fatalf("缺少 threadID 应返回 bad_request，实际 %s", body.Code)
	}
}

func TestSessionDeleteWithoutReader(t *testing.T) {
	reg, hub := newTestRegistryWithTurns(nil, nil, nil)
	c := newTestClient(hub)

	resp := deleteRequest(t, reg, c, protocol.SessionDeleteData{ThreadID: "t1"})
	var body protocol.ErrorBody
	_ = resp.DecodeData(&body)
	if body.Code != protocol.CodeInternal {
		t.Fatalf("未启用存储时应返回 internal，实际 %s", body.Code)
	}
}
