package protocol

import (
	"testing"

	"github.com/kinwyb/buibuiCodex/core/types"
)

func TestEnvelopeRoundTrip(t *testing.T) {
	env := NewEnvelope(KindAgent, string(types.EventMessageDelta)).
		WithSession("s1", "r1").
		WithChunk(3).
		WithData(&AgentData{Event: "message_delta", Delta: true, Content: "你好"})

	raw := EncodeEnvelope(env)
	if raw == nil {
		t.Fatal("EncodeEnvelope 返回 nil")
	}

	got, body := DecodeEnvelope(raw)
	if body != nil {
		t.Fatalf("解码失败: %+v", body)
	}
	if got.Kind != KindAgent || got.Type != "message_delta" {
		t.Fatalf("kind/type 不符合预期: %s/%s", got.Kind, got.Type)
	}
	if got.SessionID != "s1" || got.ReqID != "r1" || got.Chunk != 3 {
		t.Fatalf("会话字段不符合预期: %+v", got)
	}

	var data AgentData
	if err := got.DecodeData(&data); err != nil {
		t.Fatalf("解析载荷失败: %v", err)
	}
	if data.Content != "你好" || !data.Delta {
		t.Fatalf("载荷不符合预期: %+v", data)
	}
}

func TestDecodeEnvelopeRejects(t *testing.T) {
	cases := map[string]string{
		"非法 JSON": "not-json",
		"缺少 ID":   `{"v":1,"kind":"client","type":"ping"}`,
		"缺少 kind": `{"v":1,"id":"x","type":"ping"}`,
		"非法 kind": `{"v":1,"id":"x","kind":"unknown","type":"ping"}`,
		"缺少 type": `{"v":1,"id":"x","kind":"client"}`,
		"协议版本不符":  `{"v":99,"id":"x","kind":"client","type":"ping"}`,
	}
	for name, raw := range cases {
		t.Run(name, func(t *testing.T) {
			if _, body := DecodeEnvelope([]byte(raw)); body == nil {
				t.Fatalf("期望校验失败，实际通过: %s", raw)
			}
		})
	}
}

func TestIsDelta(t *testing.T) {
	if !IsDelta(KindAgent, string(types.EventMessageDelta)) {
		t.Fatal("message_delta 应视为增量消息")
	}
	if IsDelta(KindAgent, string(types.EventMessageCompleted)) {
		t.Fatal("message_completed 不应视为增量消息")
	}
	if IsDelta(KindSystem, TypePong) {
		t.Fatal("system 消息不参与增量丢弃")
	}
}

func TestKindIsValid(t *testing.T) {
	for _, k := range []Kind{KindAgent, KindSystem, KindClient, KindCron, KindNotice} {
		if !k.IsValid() {
			t.Fatalf("%s 应为合法 kind", k)
		}
	}
	if Kind("unknown").IsValid() {
		t.Fatal("未知 kind 不应通过校验")
	}
}
