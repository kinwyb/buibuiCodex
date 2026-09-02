package codex

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/kinwyb/buibuiCodex/codex/jsonRpc"
)

// EventHandler 异步事件监听函数类型
type EventHandler func(event *Event)

type EventType string

const (
	EventTypeNotification EventType = "notification"
	EventTypeRequest      EventType = "request"
)

type Event struct {
	ID       any             `json:"id"`        // request 类型时有效
	ThreadID string          `json:"thread_id"` //threadID
	TurnID   string          `json:"turn_id"`   //turnID
	Type     EventType       `json:"type"`
	Method   string          `json:"method"` //方法
	RawData  json.RawMessage `json:"raw_data"`
	Raw      any             `json:"-"` //原始数据解析出来的对象
}

func (e *Event) To[T any]() (T, error) {
	var target T

	// 1. 尝试从 Raw 转换
	if e.Raw != nil {
		// 1.1 精确匹配：Raw 的类型与 T 完全一致 (如 T 是 User 存 User，或 T 是 *User 存 *User)
		if val, ok := e.Raw.(T); ok {
			return val, nil
		}

		// 1.2 情况 A：T 是结构体值 User，但 Raw 中存的是指针 *User
		if ptrVal, ok := e.Raw.(*T); ok && ptrVal != nil {
			return *ptrVal, nil
		}

		// 1.3 情况 B：T 是指针 *User，但 Raw 中存的是结构体值 User (使用反射转换)
		targetVal := reflect.ValueOf(&target).Elem() // 获取 target (即 T 类型) 的 reflect.Value
		if targetVal.Kind() == reflect.Pointer {
			// 获取 T 指向的底层基类型 (例如 *User 对应的 User)
			elemType := targetVal.Type().Elem()
			rawVal := reflect.ValueOf(e.Raw)

			// 检查 Raw 的类型是否正好等于指针指向的底层基类型
			if rawVal.IsValid() && rawVal.Type() == elemType {
				// 动态分配一块内存空间，并将 Raw 的值复制进去
				ptr := reflect.New(elemType)
				ptr.Elem().Set(rawVal)
				// 将新建的指针转换为泛型 T 返回
				return ptr.Interface().(T), nil
			}
		}
	}

	// 2. Raw 转换失败或为 nil，降级通过 RawData 进行反序列化
	if len(e.RawData) > 0 {
		err := json.Unmarshal(e.RawData, &target)
		if err != nil {
			return target, fmt.Errorf("failed to unmarshal RawData to %T: %w", target, err)
		}
		return target, nil
	}

	return target, fmt.Errorf("cannot convert event to %T: Raw is incompatible and RawData is empty", target)
}

type Client struct {
	serverURL string
	authToken string
	conn      rpc
	ctx       context.Context
	cancel    context.CancelFunc
	threadMu  sync.Mutex
	threadSub map[string]EventHandler
	eHandler  EventHandler
}

// NewClient 创建并初始化一个 Codex Client
func NewClient(serverURL string, token string) (*Client, error) {
	ctx, cancel := context.WithCancel(context.Background())
	conn := newRpc(ctx)
	err := conn.Connect(serverURL, token)
	if err != nil {
		return nil, err
	}
	c := &Client{
		serverURL: serverURL,
		authToken: token,
		conn:      conn,
		threadSub: make(map[string]EventHandler),
		ctx:       ctx,
		cancel:    cancel,
	}
	// 收到的请求处理
	go c.requestLoop()

	//【关键步骤】连接建立后，立刻发送 initialize 请求
	initCtx, initCancel := context.WithTimeout(ctx, 5*time.Second)
	defer initCancel()

	initParams := jsonRpc.InitializeParams{}
	initParams.ClientInfo.Name = "KanFlux-Go-Client"
	initParams.ClientInfo.Version = "1.0.0"
	initParams.Capabilitie = &jsonRpc.Capabilities{
		ExperimentalApi: true,
	}

	var initResult json.RawMessage
	if err = conn.Call(initCtx, Initialize, initParams, &initResult); err != nil {
		c.Close()
		return nil, fmt.Errorf("codex app-server initialize failed: %w", err)
	}

	return c, nil
}

func (c *Client) requestLoop() {
	defer func() {
		if r := recover(); r != nil {
			go c.requestLoop()
		}
	}()
	ch := c.conn.Requests()
	for {
		select {
		case <-c.ctx.Done():
			return
		case req, ok := <-ch:
			if ok && req != nil {
				event := &Event{
					ID:      req.ID,
					Method:  req.Method,
					RawData: req.Params,
				}
				switch req.Method {
				case string(ItemToolCall):
					event.Type = EventTypeRequest
					toolCall := jsonRpc.ToolCall{}
					err := json.Unmarshal(req.Params, &toolCall)
					if err != nil {
						slog.Error(fmt.Sprintf("codex app-server tool call unmarshal failed: %v", err))
						break
					}
					slog.Debug(fmt.Sprintf("toolCall: %v", toolCall))
					event.Raw = toolCall
					event.ThreadID = toolCall.ThreadID
					event.TurnID = toolCall.TurnID
				case string(ItemCommandExecutionRequestApproval), string(ExecCommandApproval),
					string(ItemPermissionsRequestApproval), string(ItemFileChangeRequestApproval):
					event.Type = EventTypeRequest
					command := jsonRpc.BaseApprovalEvent{}
					err := json.Unmarshal(req.Params, &command)
					if err != nil {
						slog.Error(fmt.Sprintf("codex app-server command approval unmarshal failed: %v", err))
						break
					}
					slog.Debug(fmt.Sprintf("command approval: %v", command))
					event.Raw = command
					event.ThreadID = command.ThreadID
					event.TurnID = command.TurnID
				case string(ItemCompleted), string(ItemStarted):
					event.Type = EventTypeNotification
					var itemEvt jsonRpc.ItemLifecycleEvent
					err := json.Unmarshal(req.Params, &itemEvt)
					if err != nil {
						slog.Error(fmt.Sprintf("codex app-server item lifecycle event unmarshal failed: %v", err))
						break
					}
					event.Raw = itemEvt
					event.ThreadID = itemEvt.ThreadID
					event.TurnID = itemEvt.TurnID
				case string(ItemReasoningSummaryTextDelta), string(ItemReasoningTextDelta), string(ItemAgentMessageDelta):
					event.Type = EventTypeNotification
					var itemEvt jsonRpc.TextDeltaEvent
					err := json.Unmarshal(req.Params, &itemEvt)
					if err != nil {
						slog.Error(fmt.Sprintf("codex app-server text delta event unmarshal failed: %v", err))
						break
					}
					if strings.HasPrefix(req.Method, "item/reasoning") {
						itemEvt.DeltaType = jsonRpc.Reasoning
					} else if strings.HasPrefix(req.Method, "item/agentMessage") {
						itemEvt.DeltaType = jsonRpc.AgentMessage
					}
					event.Raw = itemEvt
					event.ThreadID = itemEvt.ThreadID
					event.TurnID = itemEvt.TurnID
				case string(ThreadTokenUsageUpdated):
					event.Type = EventTypeNotification
					var itemEvt jsonRpc.TokenUsageNotification
					err := json.Unmarshal(req.Params, &itemEvt)
					if err != nil {
						slog.Error(fmt.Sprintf("codex app-server token usage event unmarshal failed: %v", err))
						break
					}
					event.Raw = itemEvt
					event.ThreadID = itemEvt.ThreadID
					event.TurnID = itemEvt.TurnID
				case string(TurnStart), string(TurnCompleted):
					event.Type = EventTypeNotification
					var itemEvt jsonRpc.TurnStartEvent
					err := json.Unmarshal(req.Params, &itemEvt)
					if err != nil {
						slog.Error(fmt.Sprintf("codex app-server turn start event unmarshal failed: %v", err))
						break
					}
					event.Raw = itemEvt
					event.ThreadID = itemEvt.ThreadID
					event.TurnID = itemEvt.Turn.TurnID
				case string(NotificationMethodError):
					var itemEvt jsonRpc.CodexErrorEvent
					err := json.Unmarshal(req.Params, &itemEvt)
					if err != nil {
						slog.Error(fmt.Sprintf("codex app-server code event unmarshal failed: %v", err))
						break
					}
					slog.Error("request loop handler error:" + itemEvt.Error.Message)
				default:
					slog.Debug(fmt.Sprintf("codex app-server unsupported request type: %s => %s", req.Method, req.Params.String()))
				}
				if c.eHandler != nil {
					c.eHandler(event)
				}
				// 如果存在线程ID,并且是有效的事件，进行事件回调
				if event.ThreadID != "" && event.Type != "" {
					c.threadMu.Lock()
					sub, has := c.threadSub[event.ThreadID]
					c.threadMu.Unlock()
					if has {
						sub(event)
					}
				}
			}
		}
	}
}

func (c *Client) RegisterDefaultEventHandler(handler EventHandler) {
	c.eHandler = handler
}

// SubScribeEvent 订阅事件
func (c *Client) SubScribeEvent(threadID string, handler EventHandler) error {
	if threadID == "" {
		return fmt.Errorf("thread id is empty")
	}
	c.threadMu.Lock()
	defer c.threadMu.Unlock()
	c.threadSub[threadID] = handler
	return nil
}

// UnSubScribeEvent 取消事件订阅
func (c *Client) UnSubScribeEvent(threadID string) {
	if threadID == "" {
		return
	}
	c.threadMu.Lock()
	defer c.threadMu.Unlock()
	delete(c.threadSub, threadID)
}

// Close 关闭客户端连接
func (c *Client) Close() error {
	c.cancel()
	if c.conn != nil {
		c.conn.Close()
	}
	return nil
}

// Call 发送
func (c *Client) Call(ctx context.Context, method ClientRequestMethod, params any, result any) error {
	return c.conn.Call(ctx, method, params, result)
}

// Response 底层 JSON-RPC 同步请求封装（支持 Context 超时控制）
func (c *Client) Response(id any, result any) error {
	return c.conn.Response(id, result)
}

// --- 业务层 API 便捷方法 ---

func (c *Client) Thread() *Thread {
	return NewThread(c)
}
