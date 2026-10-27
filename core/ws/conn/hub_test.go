package conn

import (
	"testing"

	"github.com/kinwyb/buibuiCodex/core/types"
	"github.com/kinwyb/buibuiCodex/core/ws/protocol"
)

func TestHubDispatchBySession(t *testing.T) {
	h := NewHub()
	a := NewClient(Config{SendBuffer: 8}, nil, h, nil, Identity{UserID: "web"})
	b := NewClient(Config{SendBuffer: 8}, nil, h, nil, Identity{UserID: "web"})
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
	c := NewClient(Config{SendBuffer: 8}, nil, h, nil, Identity{UserID: "web"})
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
	a := NewClient(Config{SendBuffer: 8}, nil, h, nil, Identity{UserID: "web"})
	b := NewClient(Config{SendBuffer: 8}, nil, h, nil, Identity{UserID: "web"})
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

// 队列满时的取舍：先丢最旧的增量帧，关键帧优先拿到位置。
//
// 早期版本是「满就丢新帧」：浏览器稍一卡顿，一整轮的 delta 全被丢光，
// 界面只能等 message_completed 的全量文本才一次性刷出来 —— 看起来就是「不流式」。
func TestHubBackpressurePrefersLatestFrames(t *testing.T) {
	h := NewHub()
	c := NewClient(Config{SendBuffer: 1}, nil, h, nil, Identity{UserID: "web"})
	h.Add(c)
	h.Subscribe(c, "s1")

	// 塞满队列（1 条 delta）后再来一条 delta：挤掉最旧的，队列长度仍是 1，且不断开
	h.Dispatch("s1", protocol.NewEnvelope(protocol.KindAgent, string(types.EventMessageDelta)))
	h.Dispatch("s1", protocol.NewEnvelope(protocol.KindAgent, string(types.EventMessageDelta)))
	if n := h.Count(); n != 1 {
		t.Fatalf("丢弃增量帧时不应断开连接，当前连接数 %d", n)
	}
	if queued, dropped := c.out.stats(); queued != 1 || dropped != 1 {
		t.Fatalf("应保持 1 条排队、丢弃 1 条，实际 queued=%d dropped=%d", queued, dropped)
	}

	// 关键帧到达：同样先挤掉增量帧，不因为积压而断线
	h.Dispatch("s1", protocol.NewEnvelope(protocol.KindAgent, string(types.EventMessageCompleted)))
	select {
	case <-c.done:
		t.Fatal("关键帧应挤掉增量帧让位，而不是断开连接")
	default:
	}

	// 队列里全是关键帧、腾不出位置时，才退回断开重连
	h.Dispatch("s1", protocol.NewEnvelope(protocol.KindAgent, string(types.EventFinished)))
	select {
	case <-c.done:
	default:
		t.Fatal("队列里全是关键帧时，关键帧投递失败应触发连接关闭")
	}
}
