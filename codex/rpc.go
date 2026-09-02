package codex

import (
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"sync"
	"uuid"

	"github.com/gorilla/websocket"
	"github.com/kinwyb/buibuiCodex/codex/jsonRpc"
)

// rpc 通信接口
type rpc interface {

	// Connect 连接
	Connect(url string, authToken string) error

	// Call 发送
	Call(ctx context.Context, method ClientRequestMethod, params any, result any) error

	// Response 回复消息
	Response(id any, result any) error

	// Close 关闭
	Close()

	// Requests 收到的数据
	Requests() <-chan *jsonRpc.RPCRequest
}

type jsonRpcV2 struct {
	isConnected     bool
	mu              sync.Mutex
	sendMu          sync.Mutex
	conn            *websocket.Conn
	pendingRequests map[string]chan *jsonRpc.RPCResponse
	ctx             context.Context
	cancel          context.CancelFunc
	requests        chan *jsonRpc.RPCRequest
}

func (j *jsonRpcV2) Connect(url string, authToken string) error {
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.isConnected {
		return nil
	}
	if url == "" {
		return fmt.Errorf("url is empty")
	}
	dialer := websocket.DefaultDialer
	if strings.HasPrefix(url, "unix") {
		newDialer := *dialer
		newDialer.NetDialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
			var d net.Dialer
			// 强行使用 "unix" 网络协议，忽略传入的 host:port，改连 socketPath
			return d.DialContext(ctx, "unix", url)
		}
		dialer = &newDialer
		url = "ws://localhost/ws"
	}
	headers := http.Header{}
	if authToken != "" {
		headers.Add("Authorization", "Bearer "+authToken)
	}
	conn, _, err := dialer.DialContext(j.ctx, url, headers)
	if err != nil {
		return fmt.Errorf("dial codex app-server failed: %w", err)
	}
	j.conn = conn
	go j.readLoop()
	return nil
}

func (j *jsonRpcV2) Call(ctx context.Context, method ClientRequestMethod, params any, result any) error {
	reqID := uuid.NewV4().String()
	paramRaw, err := json.Marshal(params)
	if err != nil {
		return fmt.Errorf("marshal json rpc request failed: %w", err)
	}
	req := jsonRpc.RPCRequest{
		JSONRPC: "2.0",
		ID:      reqID,
		Method:  string(method),
		Params:  paramRaw,
	}
	resChan := make(chan *jsonRpc.RPCResponse, 1)

	j.sendMu.Lock()
	j.pendingRequests[reqID] = resChan
	j.sendMu.Unlock()

	defer func() {
		j.sendMu.Lock()
		delete(j.pendingRequests, reqID)
		j.sendMu.Unlock()
	}()

	// 发送消息
	j.sendMu.Lock()
	err = j.conn.WriteJSON(req)
	j.sendMu.Unlock()
	if err != nil {
		return fmt.Errorf("write websocket json failed: %w", err)
	}

	// 等待响应或 Context 超时
	select {
	case <-ctx.Done():
		return ctx.Err()
	case res := <-resChan:
		if res.Error != nil {
			return res.Error
		}
		if result != nil && len(res.Result) > 0 {
			if err := json.Unmarshal(res.Result, result); err != nil {
				return fmt.Errorf("unmarshal RPC result failed: %w", err)
			}
		}
		return nil
	}
}

func (j *jsonRpcV2) Response(id any, result any) error {
	resultRaw, merr := json.Marshal(result)
	if merr != nil {
		return fmt.Errorf("marshal RPC result failed: %w", merr)
	}
	req := jsonRpc.RPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  resultRaw,
	}
	// 发送消息
	j.sendMu.Lock()
	err := j.conn.WriteJSON(req)
	j.sendMu.Unlock()
	if err != nil {
		return fmt.Errorf("write websocket json failed: %w", err)
	}
	return nil
}

func (j *jsonRpcV2) Requests() <-chan *jsonRpc.RPCRequest {
	return j.requests
}

func (j *jsonRpcV2) readLoop() {
	defer func() {
		if err := recover(); err != nil {
			slog.Error("jsonRpcV2 readLoop recover err:", err)
			go j.readLoop()
		}
	}()

	for {
		select {
		case <-j.ctx.Done():
			return
		default:
			var msg jsonRpc.RPCData
			err := j.conn.ReadJSON(&msg)
			if err != nil {
				if !errors.As(err, io.ErrUnexpectedEOF) {
					slog.Error("jsonRpcV2 readLoop read json error", "error", err)
				}
				j.Close()
				return
			}
			if msg.IsResponse() {
				resp := msg.Response()
				j.sendMu.Lock()
				ch, exists := j.pendingRequests[resp.IDString()]
				j.sendMu.Unlock()
				if exists {
					ch <- resp
				}
				continue
			}
			// 服务端主动推送的信息
			req := msg.Request()
			j.requests <- req
		}
	}
}

func (j *jsonRpcV2) Close() {
	j.cancel()
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.isConnected {
		j.conn.Close()
		j.conn = nil
		j.isConnected = false
	}
}

// 创建一个rpc连接
func newRpc(c context.Context) rpc {
	ctx, cancel := context.WithCancel(c)
	return &jsonRpcV2{
		conn:            nil,
		ctx:             ctx,
		cancel:          cancel,
		pendingRequests: make(map[string]chan *jsonRpc.RPCResponse),
		requests:        make(chan *jsonRpc.RPCRequest, 100),
	}
}
