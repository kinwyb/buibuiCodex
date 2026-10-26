// 本文件测试 chat.cancel。

package command

import (
	"context"
	"testing"

	"github.com/kinwyb/buibuiCodex/core/ws/protocol"
)

func TestChatCancelRejectsForeignSession(t *testing.T) {
	reg, hub := newTestRegistry(nil, nil)
	c := newTestClientAs(hub, "alice")

	env := protocol.NewEnvelope(protocol.KindClient, protocol.TypeChatCancel).
		WithData(&protocol.ChatCancelData{SessionID: "ws_bob_c1"})
	reg.Dispatch(context.Background(), c, env)

	raw := <-c.Outbound()
	resp, _ := protocol.DecodeEnvelope(raw)
	var body protocol.ErrorBody
	_ = resp.DecodeData(&body)
	if body.Code != protocol.CodeForbidden {
		t.Fatalf("跨用户取消应被拒绝，错误码 %s", body.Code)
	}
}

func TestChatCancelUsesClientChatID(t *testing.T) {
	reg, hub := newTestRegistry(nil, nil)
	c := newTestClientAs(hub, "alice")
	c.SetChatID("chat-a")

	env := protocol.NewEnvelope(protocol.KindClient, protocol.TypeChatCancel)
	reg.Dispatch(context.Background(), c, env)

	raw := <-c.Outbound()
	resp, _ := protocol.DecodeEnvelope(raw)
	// 未提供取消器时应走到 internal，说明 session 已从连接上下文推导出来
	var body protocol.ErrorBody
	_ = resp.DecodeData(&body)
	if body.Code != protocol.CodeInternal {
		t.Fatalf("应从连接上下文推导会话，错误码 %s", body.Code)
	}
}
