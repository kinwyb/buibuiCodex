// 本文件测试注册表本身：命令分发、会话键与归属判断。

package command

import (
	"context"
	"testing"

	"github.com/kinwyb/buibuiCodex/core/bus"
	"github.com/kinwyb/buibuiCodex/core/types"
	"github.com/kinwyb/buibuiCodex/core/ws/protocol"
)

func TestDispatchRejectsNonClientKind(t *testing.T) {
	msgBus := bus.NewMessageBus(2)
	defer msgBus.Close()

	reg, hub := newTestRegistry(msgBus, nil)
	c := newTestClient(hub)

	reg.Dispatch(context.Background(), c, protocol.NewEnvelope(protocol.KindAgent, "message_delta"))

	raw := <-c.Outbound()
	e, _ := protocol.DecodeEnvelope(raw)
	var data protocol.ErrorBody
	_ = e.DecodeData(&data)
	if data.Code != protocol.CodeUnsupported {
		t.Fatalf("错误码应为 unsupported，实际 %s", data.Code)
	}
}

func TestDispatchRejectsUnknownType(t *testing.T) {
	msgBus := bus.NewMessageBus(2)
	defer msgBus.Close()

	reg, hub := newTestRegistry(msgBus, nil)
	c := newTestClient(hub)

	reg.Dispatch(context.Background(), c, protocol.NewEnvelope(protocol.KindClient, "chat.unknown"))

	raw := <-c.Outbound()
	e, _ := protocol.DecodeEnvelope(raw)
	if e.Type != protocol.TypeError {
		t.Fatalf("未知命令应返回 error，实际 %s", e.Type)
	}
}

func TestSessionKeyMatchesInputMessage(t *testing.T) {
	reg, _ := newTestRegistry(nil, nil)
	in := &types.InputMessage{Channel: "ws", AccountID: "web", ChatID: "c1"}
	if got := reg.SessionKey("web", "c1"); got != in.SessionID() {
		t.Fatalf("会话键不一致: %s != %s", got, in.SessionID())
	}
}

func TestOwnsSessionPrefix(t *testing.T) {
	reg, _ := newTestRegistry(nil, nil)
	cases := []struct {
		userID    string
		sessionID string
		want      bool
	}{
		{"alice", "ws_alice_c1", true},
		{"alice", "ws_alice_", true},
		{"alice", "ws_bob_c1", false},
		// 关键：alice 不能匹配到 alice2 的会话
		{"alice", "ws_alice2_c1", false},
		{"alice", "alice_c1", false},
		{"", "ws_alice_c1", false},
	}
	for _, c := range cases {
		if got := reg.ownsSession(c.userID, c.sessionID); got != c.want {
			t.Fatalf("ownsSession(%q, %q) = %v，期望 %v", c.userID, c.sessionID, got, c.want)
		}
	}
}
