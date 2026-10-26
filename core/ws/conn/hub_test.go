package conn

import (
	"testing"

	"github.com/kinwyb/buibuiCodex/core/types"
	"github.com/kinwyb/buibuiCodex/core/ws/protocol"
)

func TestHubDispatchBySession(t *testing.T) {
	h := NewHub()
	a := NewClient(Config{SendBuffer: 8}, nil, h, nil)
	b := NewClient(Config{SendBuffer: 8}, nil, h, nil)
	h.Add(a)
	h.Add(b)
	h.Subscribe(a, "s1")

	if n := h.Dispatch("s1", protocol.NewEnvelope(protocol.KindAgent, string(types.EventMessageCompleted))); n != 1 {
		t.Fatalf("投递数量应为 1，实际 %d", n)
	}
	select {
	case raw := <-a.Outbound():
		env, body := protocol.DecodeEnvelope(raw)
		if body != nil {
			t.Fatalf("解析失败: %+v", body)
		}
		if env.Seq == 0 {
			t.Fatal("应设置全局序号")
		}
	default:
		t.Fatal("订阅者未收到消息")
	}
	select {
	case <-b.Outbound():
		t.Fatal("未订阅的连接不应收到消息")
	default:
	}

	if n := h.Dispatch("s2", protocol.NewEnvelope(protocol.KindAgent, string(types.EventMessageCompleted))); n != 0 {
		t.Fatalf("无订阅者时应投递 0 条，实际 %d", n)
	}
}

func TestHubRemoveCleansSubscriptions(t *testing.T) {
	h := NewHub()
	c := NewClient(Config{SendBuffer: 8}, nil, h, nil)
	h.Add(c)
	h.Subscribe(c, "s1")
	if h.SessionCount() != 1 {
		t.Fatalf("会话数应为 1，实际 %d", h.SessionCount())
	}

	h.Remove(c)
	if h.Count() != 0 || h.SessionCount() != 0 {
		t.Fatalf("注销后应清空连接与会话: count=%d sessions=%d", h.Count(), h.SessionCount())
	}
}

func TestHubBroadcast(t *testing.T) {
	h := NewHub()
	a := NewClient(Config{SendBuffer: 8}, nil, h, nil)
	b := NewClient(Config{SendBuffer: 8}, nil, h, nil)
	h.Add(a)
	h.Add(b)

	if n := h.Broadcast(protocol.NewEnvelope(protocol.KindNotice, "alert")); n != 2 {
		t.Fatalf("广播数量应为 2，实际 %d", n)
	}
	for _, c := range []*Client{a, b} {
		select {
		case <-c.Outbound():
		default:
			t.Fatal("广播未送达所有连接")
		}
	}
}

func TestHubBackpressureDropsDelta(t *testing.T) {
	h := NewHub()
	c := NewClient(Config{SendBuffer: 1}, nil, h, nil)
	h.Add(c)
	h.Subscribe(c, "s1")

	// 塞满队列后，增量消息应被丢弃而不是阻塞或断开
	h.Dispatch("s1", protocol.NewEnvelope(protocol.KindAgent, string(types.EventMessageDelta)))
	h.Dispatch("s1", protocol.NewEnvelope(protocol.KindAgent, string(types.EventMessageDelta)))
	if h.Count() != 1 {
		t.Fatal("增量消息丢弃时不应断开连接")
	}

	// 关键消息在队列满时应断开连接
	h.Dispatch("s1", protocol.NewEnvelope(protocol.KindAgent, string(types.EventMessageCompleted)))
	if h.Count() != 1 {
		t.Fatal("连接应仍由 Hub 持有，直到读协程注销")
	}
	select {
	case <-c.done:
	default:
		t.Fatal("关键消息投递失败时应触发连接关闭")
	}
}
