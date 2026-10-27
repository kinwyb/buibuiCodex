package conn

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// newTestPair 起一个真实的 ws 服务端，返回「服务端侧的 Client」和「客户端收到的文本帧」
func newTestPair(t *testing.T, cfg Config) (*Client, <-chan string) {
	t.Helper()

	received := make(chan string, 32)
	upgrader := websocket.Upgrader{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		peer, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer peer.Close()
		for {
			_, raw, err := peer.ReadMessage()
			if err != nil {
				return
			}
			received <- string(raw)
		}
	}))
	t.Cleanup(srv.Close)

	url := "ws" + strings.TrimPrefix(srv.URL, "http")
	ws, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("连接测试服务端失败: %v", err)
	}

	if cfg.PingPeriod <= 0 {
		cfg.PingPeriod = time.Minute
	}
	if cfg.WriteWait <= 0 {
		cfg.WriteWait = 2 * time.Second
	}
	c := NewClient(cfg, ws, NewHub(), nil, Identity{UserID: "web"})
	t.Cleanup(c.Close)
	return c, received
}

// writePump 必须按入队顺序把帧全部发出去：
// 队列改造后消费者靠 notify 醒来，漏掉一次通知就会「永远不发」，这条守的就是它。
func TestWritePumpDrainsQueueInOrder(t *testing.T) {
	c, received := newTestPair(t, Config{SendBuffer: 8})
	go c.writePump()

	for _, label := range []string{"a", "b", "c"} {
		if !c.enqueue([]byte(label), false) {
			t.Fatalf("入队失败：%s", label)
		}
	}

	for _, want := range []string{"a", "b", "c"} {
		select {
		case got := <-received:
			if got != want {
				t.Fatalf("发送顺序不符：期望 %s，实际 %s", want, got)
			}
		case <-time.After(3 * time.Second):
			t.Fatalf("超时未收到 %s", want)
		}
	}
}

// 队列积压时新帧仍然要能发出去（丢的是最旧的，不是最新的）
func TestWritePumpKeepsSendingUnderBacklog(t *testing.T) {
	c, received := newTestPair(t, Config{SendBuffer: 2})
	go c.writePump()

	// 一次塞入远超队列容量的增量帧：只有最近几条能留下（旧的在 push 时就被挤掉）
	for i := 0; i < 20; i++ {
		c.enqueue([]byte{byte('a' + i)}, false)
	}
	// 关键帧必须送达
	if !c.enqueue([]byte("END"), true) {
		t.Fatal("关键帧应能挤掉增量帧入队")
	}

	deadline := time.After(3 * time.Second)
	for {
		select {
		case got := <-received:
			if got == "END" {
				// 发完之后队列必须被清空，否则说明消费者漏了通知
				if queued, _ := c.out.stats(); queued != 0 {
					t.Fatalf("结束后队列应清空，实际 %d", queued)
				}
				return
			}
		case <-deadline:
			t.Fatal("超时未收到关键帧")
		}
	}
}
