package types

import "testing"

func TestInputMessageSessionIDIncludesSender(t *testing.T) {
	base := InputMessage{
		Channel:   "wxcom",
		AccountID: "bot-1",
		ChatID:    "group-1",
	}

	alice := base
	alice.SenderID = "alice"
	if got := alice.SessionID(); got != "wxcom_bot-1_group-1_alice" {
		t.Fatalf("sessionID 应包含发送者: %s", got)
	}

	bob := base
	bob.SenderID = "bob"
	if alice.SessionID() == bob.SessionID() {
		t.Fatal("同一群聊的不同发送者必须生成不同 sessionID")
	}
}

func TestInputMessageSessionIDWithoutSenderKeepsOldFormat(t *testing.T) {
	msg := &InputMessage{
		Channel:   "wxcom",
		AccountID: "bot-1",
		ChatID:    "group-1",
	}
	if got := msg.SessionID(); got != "wxcom_bot-1_group-1" {
		t.Fatalf("SenderID 为空时应保持原格式: %s", got)
	}
}

func TestInputMessageSessionIDOmitsDuplicatePrivateSender(t *testing.T) {
	msg := &InputMessage{
		Channel:   "wxcom",
		AccountID: "bot-1",
		ChatID:    "alice",
		SenderID:  "alice",
	}
	if got := msg.SessionID(); got != "wxcom_bot-1_alice" {
		t.Fatalf("私聊不应重复拼接 SenderID: %s", got)
	}
}

func TestInputMessageSessionIDKeepsExplicitSession(t *testing.T) {
	msg := &InputMessage{
		Session:   "ws_user_chat",
		Channel:   "ws",
		AccountID: "user",
		ChatID:    "chat",
		SenderID:  "user",
	}
	if got := msg.SessionID(); got != "ws_user_chat" {
		t.Fatalf("显式 Session 不应被重写: %s", got)
	}
}

func TestInputMessageThreadIDFromMetadata(t *testing.T) {
	msg := &InputMessage{
		Metadata: map[string]any{MetadataThreadID: " t-1 "},
	}
	if got := msg.ThreadID(); got != "t-1" {
		t.Fatalf("应读取并清理 metadata 中的 threadID，实际 %q", got)
	}

	msg.Metadata = map[string]any{"threadID": "t-2"}
	if got := msg.ThreadID(); got != "t-2" {
		t.Fatalf("应兼容协议风格键名，实际 %q", got)
	}

	var nilMsg *InputMessage
	if got := nilMsg.ThreadID(); got != "" {
		t.Fatalf("nil 消息应返回空 threadID，实际 %q", got)
	}
}
