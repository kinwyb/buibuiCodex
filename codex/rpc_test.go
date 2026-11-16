package codex

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// startDrainServer 起一个只收不回的 ws 服务：Call 会一直挂起，正好用来验证 Close 的清理。
func startDrainServer(t *testing.T) *httptest.Server {
	t.Helper()
	upgrader := websocket.Upgrader{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}))
	t.Cleanup(server.Close)
	return server
}

func wsURL(server *httptest.Server) string {
	return strings.Replace(server.URL, "http://", "ws://", 1)
}

// 连接断开后 Close 必须立即失败挂起的请求，而不是让调用方干等到 ctx 超时。
func TestRpcCloseUnblocksPendingCall(t *testing.T) {
	server := startDrainServer(t)
	j := newRpc(context.Background()).(*jsonRpcV2)
	if err := j.Connect(wsURL(server), ""); err != nil {
		t.Fatalf("connect failed: %v", err)
	}

	callErr := make(chan error, 1)
	go func() {
		callErr <- j.Call(context.Background(), Initialize, struct{}{}, nil)
	}()

	// 等请求登记进 pending 再关，确保走的是「挂起中被 Close」这条路径
	deadline := time.Now().Add(2 * time.Second)
	for {
		j.sendMu.Lock()
		n := len(j.pendingRequests)
		j.sendMu.Unlock()
		if n > 0 || time.Now().After(deadline) {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}

	j.Close()

	select {
	case err := <-callErr:
		if err == nil || !strings.Contains(err.Error(), "connection closed") {
			t.Fatalf("挂起的 Call 应拿到连接关闭错误，实际 %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Close 后挂起的 Call 没有及时返回")
	}

	select {
	case <-j.Done():
	default:
		t.Fatal("Close 后 Done 应已触发")
	}
}

// 关闭后再 Call 应返回明确错误，不能对着空连接 panic 或写死 socket。
func TestRpcCallAfterCloseReturnsError(t *testing.T) {
	server := startDrainServer(t)
	j := newRpc(context.Background())
	if err := j.Connect(wsURL(server), ""); err != nil {
		t.Fatalf("connect failed: %v", err)
	}
	j.Close()

	err := j.Call(context.Background(), Initialize, struct{}{}, nil)
	if err == nil || !strings.Contains(err.Error(), "connection is closed") {
		t.Fatalf("关闭后的 Call 应返回连接已关闭错误，实际 %v", err)
	}
	if err := j.Response("any", struct{}{}); err == nil {
		t.Fatal("关闭后的 Response 应返回错误")
	}
}

// Connect 必须登记连接状态，否则 Close 永远不清理旧 socket（broken pipe 的根源）。
func TestRpcConnectTracksState(t *testing.T) {
	server := startDrainServer(t)
	j := newRpc(context.Background()).(*jsonRpcV2)
	if err := j.Connect(wsURL(server), ""); err != nil {
		t.Fatalf("connect failed: %v", err)
	}
	j.mu.Lock()
	connected, conn := j.isConnected, j.conn
	j.mu.Unlock()
	if !connected || conn == nil {
		t.Fatal("Connect 后应登记 isConnected 与 conn")
	}

	j.Close()
	j.mu.Lock()
	connected, conn = j.isConnected, j.conn
	j.mu.Unlock()
	if connected || conn != nil {
		t.Fatal("Close 后应清空连接状态")
	}
}
