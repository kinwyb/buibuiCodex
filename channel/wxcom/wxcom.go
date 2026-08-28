// Package wxcom provides enterprise WeChat (企业微信) intelligent robot channel implementation.
// Based on WebSocket long connection for message sending/receiving, streaming replies, template cards, etc.
package wxcom

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/kinwyb/buibuiCodex/channel"
	"github.com/kinwyb/buibuiCodex/core/bus"
	"github.com/kinwyb/buibuiCodex/core/types"
)

// TempFilePath 临时文件目录
var TempFilePath, _ = filepath.Abs("./tmp")

// Channel 企业微信智能机器人Channel实现
type Channel struct {
	*channel.BaseChannel
	// WebSocket管理器
	wsManager *WsManager
	// 消息处理器
	handler *MessageHandler
	// 配置
	config *WxComConfig
	// 流式消息缓冲
	streamBuf *channel.StreamBuffer
	// logger
	logger *slog.Logger
	// 运行状态
	running   bool
	runningMu sync.Mutex
	// session 取消器
	canceler channel.Canceller
	// streamid count
	streamIDMapLock sync.RWMutex
	streamIDMap     map[string]int
}

// NewWxComChannel 创建企业微信Channel
func NewWxComChannel(msgBus *bus.MessageBus, cfg *WxComConfig, canceler channel.Canceller) (*Channel, error) {
	return NewWxComChannelWithAccount(msgBus, cfg, "", "", canceler)
}

// NewWxComChannelWithAccount 创建企业微信Channel（带账号标识）
// accountID 用于多账号场景，生成唯一 channel 名称：wxCom:accountID
// defaultAgent 用于指定默认处理的 agent 名称
func NewWxComChannelWithAccount(msgBus *bus.MessageBus, cfg *WxComConfig, accountID string, defaultAgent string, canceler channel.Canceller) (*Channel, error) {
	// 设置默认值
	cfg.SetDefaults()

	// 验证配置
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	// 生成唯一的 channel 名称
	channelName := channel.WxComChannel
	if accountID != "" {
		channelName = channel.WxComChannel + ":" + accountID
	}

	logger := slog.Default().With("channel", channelName, "bot_id", cfg.BotID)

	// 创建基础配置
	baseConfig := channel.BaseChannelConfig{
		Enabled:    cfg.Enabled,
		AccountID:  cfg.BotID,
		Name:       channelName,
		AllowedIDs: nil, // 企业微信通过后台配置权限
	}

	// 创建基础实现
	base := channel.NewChannelBase(channelName, cfg.BotID, baseConfig, msgBus, defaultAgent)

	// 创建WebSocket管理器
	wsManager := NewWsManager(cfg, logger)

	// 创建消息处理器
	handler := NewMessageHandler(logger)

	c := &Channel{
		BaseChannel: base,
		wsManager:   wsManager,
		handler:     handler,
		config:      cfg,
		logger:      logger,
		canceler:    canceler,
		streamIDMap: make(map[string]int),
	}
	c.streamBuf = channel.NewStreamBufferWithDone(bus.StreamingModeAccumulate, c.sendStream, c.cleanupStreamID)
	return c, nil
}

// Start 启动Channel
func (c *Channel) Start(ctx context.Context) error {
	if !c.config.Enabled {
		c.logger.Info("WxCom channel is disabled")
		return nil
	}

	c.runningMu.Lock()
	if c.running {
		c.runningMu.Unlock()
		return nil
	}
	c.running = true
	c.runningMu.Unlock()

	c.logger.Info("Starting WxCom channel")

	// 设置WebSocket回调
	c.setupCallbacks()

	// 建立WebSocket连接
	if err := c.wsManager.Connect(ctx); err != nil {
		return fmt.Errorf("failed to connect WebSocket: %w", err)
	}

	// 调用基础实现的Start
	c.BaseChannel.Start(ctx)

	c.logger.Info("WxCom channel started")

	return nil
}

// setupCallbacks 设置WebSocket回调
func (c *Channel) setupCallbacks() {
	c.wsManager.SetOnConnected(func() {
		c.logger.Info("WebSocket connected")
	})

	c.wsManager.SetOnAuthenticated(func() {
		c.logger.Info("WebSocket authenticated")
	})

	c.wsManager.SetOnDisconnected(func(reason string) {
		c.logger.Warn("WebSocket disconnected", "reason", reason)
	})

	c.wsManager.SetOnReconnecting(func(attempt int) {
		c.logger.Info("WebSocket reconnecting", "attempt", attempt)
	})

	c.wsManager.SetOnError(func(err error) {
		c.logger.Error("WebSocket error", "error", err)
	})

	// 消息回调
	c.wsManager.SetOnMessage(func(frame *WsFrame) {
		c.handleMessage(frame)
	})

	// 事件回调
	c.wsManager.SetOnEvent(func(frame *WsFrame) {
		c.handleEvent(frame)
	})
}

// handleMessage 处理企业微信收到的消息内容
func (c *Channel) handleMessage(frame *WsFrame) {
	msg, err := c.handler.ParseInboundMessage(frame)
	if err != nil {
		c.logger.Error("Failed to parse inbound message", "error", err)
		return
	}

	// 转换为bus消息，使用 channel 自己的名称（支持多账号场景）
	inbound := c.handler.ConvertToInbound(msg, c.Name(), c.config.BotID)

	if inbound == nil { //忽略空消息
		return
	}
	inbound.Content = strings.TrimSpace(inbound.Content)
	// 清理@消息
	if strings.HasPrefix(inbound.Content, "@") {
		re := regexp.MustCompile(`^@\S+\s?`)
		inbound.Content = strings.TrimSpace(re.ReplaceAllString(inbound.Content, ""))
	}

	// 下载并解密媒体文件
	if len(inbound.Media) > 0 {
		c.processMedia(context.Background(), inbound)
	}

	// 检测停止命令
	if isStopCommand(inbound.Content) {
		sessionKey := inbound.SessionID() + inbound.SenderID
		if c.canceler != nil {
			c.canceler.Cancel(sessionKey)
		}
		c.SendMessage(context.Background(), inbound.ChatID, "已停止当前处理")
		return
	}

	// 发布入站消息
	if err = c.PublishInbound(context.Background(), inbound); err != nil {
		c.logger.Error("Failed to publish inbound message", "error", err)
	}
}

// isStopCommand 判断是否为停止命令
func isStopCommand(content string) bool {
	c := strings.TrimSpace(strings.ToLower(content))
	return c == "停止" || c == "stop" || c == "/stop"
}

// handleEvent 处理企业微信收到的事件
func (c *Channel) handleEvent(frame *WsFrame) {
	event, err := c.handler.ParseEvent(frame)
	if err != nil {
		c.logger.Error("Failed to parse event", "error", err)
		return
	}

	c.logger.Debug("Received event", "event_type", event.EventType, "user_id", event.UserID)

	// 处理进入会话事件
	if event.EventType == EventTypeEnterChat {
		// 发送欢迎语 (可选)
		return
		// 需要在5秒内回复，这里不自动回复，由Agent处理
		//inbound := &bus.InboundMessage{
		//	ID:            frame.Headers["req_id"],
		//	Channel:       c.Name(),
		//	AccountID:     c.config.BotID,
		//	SenderID:      event.UserID,
		//	ChatID:        event.ChatID,
		//	Content:       "你好", // 空内容表示进入会话事件
		//	StreamingMode: bus.StreamingModeAccumulate,
		//	Timestamp:     event.EventTime,
		//	Metadata: map[string]interface{}{
		//		"event_type": EventTypeEnterChat,
		//		"req_id":     frame.Headers["req_id"],
		//	},
		//}
		//c.PublishInbound(context.Background(), inbound)
	}

	// 处理模板卡片事件
	if event.EventType == EventTypeTemplateCardEvent {
		inbound := &types.InputMessage{
			ID:        frame.Headers["req_id"],
			Channel:   c.Name(),
			AccountID: c.config.BotID,
			SenderID:  event.UserID,
			ChatID:    event.ChatID,
			Content:   fmt.Sprintf("模板卡片事件: %s", event.EventKey),
			Timestamp: event.EventTime,
			Metadata: map[string]any{
				"event_type": EventTypeTemplateCardEvent,
				"event_key":  event.EventKey,
				"task_id":    event.TaskID,
				"req_id":     frame.Headers["req_id"],
			},
		}
		c.PublishInbound(context.Background(), inbound)
	}

	// 处理反馈事件
	if event.EventType == EventTypeFeedbackEvent {
		inbound := &types.InputMessage{
			ID:        frame.Headers["req_id"],
			Channel:   c.Name(),
			AccountID: c.config.BotID,
			SenderID:  event.UserID,
			ChatID:    event.ChatID,
			Content:   "用户反馈事件",
			Timestamp: event.EventTime,
			Metadata: map[string]any{
				"event_type": EventTypeFeedbackEvent,
				"req_id":     frame.Headers["req_id"],
			},
		}
		c.PublishInbound(context.Background(), inbound)
	}
}

// Stop 停止Channel
func (c *Channel) Stop() error {
	c.runningMu.Lock()
	if !c.running {
		c.runningMu.Unlock()
		return nil
	}
	c.running = false
	c.runningMu.Unlock()

	c.logger.Info("Stopping WxCom channel")

	// 停止流式缓冲
	if c.streamBuf != nil {
		c.streamBuf.Stop()
	}

	// 断开WebSocket连接
	c.wsManager.Disconnect()

	// 调用基础实现的Stop
	c.BaseChannel.Stop()

	c.logger.Info("WxCom channel stopped")

	return nil
}

// IsRunning 是否运行中
func (c *Channel) IsRunning() bool {
	c.runningMu.Lock()
	defer c.runningMu.Unlock()
	return c.running && c.wsManager.IsConnected()
}

// Send 发送完整消息
func (c *Channel) Send(ctx context.Context, event *types.Event) error {
	if !c.IsRunning() {
		return nil
	}
	//slog.Info("receve msg", "event", fmt.Sprintf("%v", event), "message", fmt.Sprintf("%v", event.Message))
	if c.streamBuf != nil {
		c.streamBuf.Push(event)
		return nil
	}
	return c.sendStream(ctx, event)
}

// sendStream 发送流式消息的实际实现（作为 StreamBuffer 回调）
func (c *Channel) sendStream(ctx context.Context, event *types.Event) error {
	cmd := WsCmdResponse

	// 从metadata获取req_id
	reqID := event.ReqID
	msg := event.Message
	if msg.Content == "" && msg.ReasoningContent == "" { //没有内容的消息不发送
		return nil
	}

	// 如果是事件消息，使用专用回复方法（流式最终帧）
	if eventType := c.getEventType(event); eventType != "" && !msg.IsDetla {
		return c.sendEventReply(ctx, reqID, eventType, msg)
	}

	if reqID == "" {
		reqID = generateReqID(WsCmdSendMsg)
		cmd = WsCmdSendMsg
	}

	// req_id 作为 streamID
	streamID := reqID

	// 发送流式消息
	finish := !msg.IsDetla

	if cmd == WsCmdResponse {
		c.streamIDMapLock.RLock()
		streamCount, _ := c.streamIDMap[reqID]
		c.streamIDMapLock.RUnlock()
		// req_id 作为 streamID
		if streamCount > 0 {
			streamID = fmt.Sprintf("%s_%d", reqID, streamCount)
		}
		if finish { //如果消息完结，流式ID+1，可以区分后面的新数据
			c.streamIDMapLock.Lock()
			c.streamIDMap[reqID] = streamCount + 1
			c.streamIDMapLock.Unlock()
			//c.logger.Error("Stream finish", "count", streamCount+1)
		}
	}

	if msg.ReasoningContent == "" && msg.Content == "" { //空消息不用发送
		return nil
	}

	body := c.handler.BuildStreamReply(streamID, msg.ReasoningContent, msg.Content, finish, nil, nil)

	if cmd == WsCmdSendMsg {
		if !finish {
			return nil
		}
		body = c.handler.BuildMarkdownReply(msg.Content)
		body["chatid"] = event.ChatID
	}

	_, err := c.wsManager.SendReply(ctx, reqID, body, cmd)
	if err != nil {
		c.logger.Error("Failed to send stream message", "error", err)
		if strings.Contains(err.Error(), "stream message update expired") { //如果流式输出超时不允许更新了，修改流式数据ID尝试重新发送
			isRetryMsg := false
			if event.Metadata != nil {
				_, isRetryMsg = event.Metadata["is_retry"]
			}
			if isRetryMsg { //如果is_retry已经存在，还是提示失败就不再重复处理
				c.logger.Info("retry send by update streaming id fail")
			} else {
				c.logger.Info("retry send by update streaming id")
				c.streamIDMapLock.Lock()
				c.streamIDMap[reqID] = c.streamIDMap[reqID] + 1
				c.streamIDMapLock.Unlock()
				if event.Metadata == nil {
					event.Metadata = make(map[string]any)
				}
				event.Metadata["is_retry"] = true
				c.sendStream(ctx, event) //尝试重新发送消息
			}
		}
		return err
	}

	return nil
}

// getEventType 从消息元数据中获取事件类型
func (c *Channel) getEventType(msg *types.Event) string {
	if msg.Metadata == nil {
		return ""
	}
	if et, ok := msg.Metadata["event_type"].(string); ok {
		return et
	}
	return ""
}

// sendEventReply 使用专用回复方法回复事件
func (c *Channel) sendEventReply(ctx context.Context, reqID, eventType string, msg *types.Message) error {
	content := msg.Content
	//if msg.Error != "" {
	//	content = msg.Error
	//}

	switch eventType {
	case EventTypeEnterChat:
		if content == "" {
			content = "你好"
		}
		return c.ReplyWelcome(ctx, reqID, content)

	case EventTypeTemplateCardEvent:
		return c.UpdateTemplateCard(ctx, reqID, NewTextNoticeCard("任务完成", content), nil)

	case EventTypeFeedbackEvent:
		return c.ReplyWelcome(ctx, reqID, content)

	default:
		_, err := c.wsManager.SendReply(ctx, reqID, c.handler.BuildTextReply(content), WsCmdResponse)
		return err
	}
}

// HandleEvent 处理聊天事件（仅状态通知）
func (c *Channel) HandleEvent(ctx context.Context, event *types.Event) error {
	if !c.IsRunning() {
		return nil
	}

	// ChatEvent 现在只做状态通知，不携带内容
	// 内容通过 OutboundMessage 发送，由 Send/SendStream 处理
	switch event.Type {
	case types.EventMessageStart:
		// 开始处理（可选：显示 loading 状态）
		c.logger.Info("Chat message started", "chat_id", event.ChatID)

	case types.EventReasoningStart:
		// 开始处理（可选：显示 loading 状态）
		c.logger.Info("Chat reasoning started", "chat_id", event.ChatID)

	case types.EventToolStart:
		for _, tool := range event.Message.ToolCalls {
			c.logger.Info("Tool started", "tool_id", tool.ID, "tool_name", tool.Name, "chat_id", event.ChatID, "args", tool.Params)
		}

	case types.EventToolCompleted:
		for _, tool := range event.Message.ToolCalls {
			c.logger.Info("Tool completed", "tool_id", tool.ID, "tool_name", tool.Name, "chat_id", event.ChatID)
		}

	case types.EventError: // 错误
		c.logger.Error("Chat error", "error", event.Message.Content, "chat_id", event.ChatID)
	}
	return nil
}

func (c *Channel) cleanupStreamID(reqID string) {
	if reqID == "" {
		return
	}
	c.streamIDMapLock.Lock()
	delete(c.streamIDMap, reqID)
	c.streamIDMapLock.Unlock()
}

// IsAllowed 检查发送者是否允许
func (c *Channel) IsAllowed(senderID string) bool {
	// 企业微信通过后台配置权限，这里始终返回true
	return c.config.Enabled
}

// ReplyWelcome 回复欢迎语 (需在收到enter_chat事件5秒内调用)
func (c *Channel) ReplyWelcome(ctx context.Context, reqID string, content string) error {
	if !c.IsRunning() {
		return ErrNotConnected
	}

	body := c.handler.BuildTextReply(content)
	_, err := c.wsManager.SendReply(ctx, reqID, body, WsCmdResponseWelcome)
	return err
}

// ReplyMarkdown 回复Markdown消息
func (c *Channel) ReplyMarkdown(ctx context.Context, reqID string, content string) error {
	if !c.IsRunning() {
		return ErrNotConnected
	}

	body := c.handler.BuildMarkdownReply(content)
	_, err := c.wsManager.SendReply(ctx, reqID, body, WsCmdResponse)
	return err
}

// SendMessage 主动发送消息
func (c *Channel) SendMessage(ctx context.Context, chatID string, content string) error {
	if !c.IsRunning() {
		return ErrNotConnected
	}

	reqID := generateReqID(WsCmdSendMsg)
	body := c.handler.BuildSendMessage(chatID, c.handler.BuildMarkdownReply(content))
	_, err := c.wsManager.SendReply(ctx, reqID, body, WsCmdSendMsg)
	return err
}

// ReplyTemplateCard 回复模板卡片消息
func (c *Channel) ReplyTemplateCard(ctx context.Context, reqID string, card *TemplateCard, feedback *CardFeedback) error {
	if !c.IsRunning() {
		return ErrNotConnected
	}

	body := c.handler.BuildTemplateCardReply(card, feedback)
	_, err := c.wsManager.SendReply(ctx, reqID, body, WsCmdResponse)
	return err
}

// ReplyStreamWithCard 发送流式消息+模板卡片组合回复
func (c *Channel) ReplyStreamWithCard(ctx context.Context, reqID, streamID, content string, finish bool,
	card *TemplateCard, cardFeedback *CardFeedback) error {
	if !c.IsRunning() {
		return ErrNotConnected
	}

	body := c.handler.BuildStreamWithCardReply(streamID, content, finish, nil, nil, card, cardFeedback)
	_, err := c.wsManager.SendReply(ctx, reqID, body, WsCmdResponse)
	return err
}

// UpdateTemplateCard 更新模板卡片 (需在收到template_card_event事件5秒内调用)
func (c *Channel) UpdateTemplateCard(ctx context.Context, reqID string, card *TemplateCard, userIDs []string) error {
	if !c.IsRunning() {
		return ErrNotConnected
	}

	body := c.handler.BuildUpdateTemplateCard(card, userIDs)
	_, err := c.wsManager.SendReply(ctx, reqID, body, WsCmdResponseUpdate)
	return err
}

// SendTemplateCard 主动发送模板卡片消息
func (c *Channel) SendTemplateCard(ctx context.Context, chatID string, card *TemplateCard) error {
	if !c.IsRunning() {
		return ErrNotConnected
	}

	reqID := generateReqID(WsCmdSendMsg)
	body := c.handler.BuildSendMessage(chatID, c.handler.BuildTemplateCardReply(card, nil))
	_, err := c.wsManager.SendReply(ctx, reqID, body, WsCmdSendMsg)
	return err
}

// HandleRequest 处理请求消息（如 send_file）
func (c *Channel) HandleRequest(ctx context.Context, tp types.SyncMessageType, event *types.Event) (*types.InputMessage, error) {
	// 根据请求类型处理
	switch tp {
	case types.SyncMessageTypeSendFile:
		return c.handleSendFileRequest(ctx, event)
	default:
		// 不支持的请求类型，返回 nil 表示不处理
		return nil, nil
	}
}

// handleSendFileRequest 处理发送文件请求
func (c *Channel) handleSendFileRequest(ctx context.Context, event *types.Event) (*types.InputMessage, error) {
	if !c.IsRunning() {
		return nil, errors.New("发送失败：通道未连接")
	}
	request := event.Message
	// 从 request.Media 获取文件路径
	if len(request.Media) == 0 {
		return nil, errors.New("发送失败：未提供文件")
	}

	filePath := request.Media[0].URL
	fileType := request.Media[0].Type
	caption := ""
	if request.Media[0].Metadata != nil {
		if cap, ok := request.Media[0].Metadata["caption"].(string); ok {
			caption = cap
		}
	}

	// 实现微信发送文件逻辑
	c.logger.Info("Send file request received", "file_path", filePath, "chat_id", event.ChatID, "caption", caption, "media_type", request.Media[0].MimeType)

	// 1. 读取文件
	var fileData []byte
	var err error
	if !strings.HasPrefix(filePath, "http") {
		fileData, err = os.ReadFile(filePath)
	} else {
		// 1. 发起网络请求获取文件
		resp, herr := http.Get(filePath)
		if herr == nil {
			defer resp.Body.Close() // 记得用完关闭连接
			// 检查 HTTP 状态码是否为 200 OK
			if resp.StatusCode == http.StatusOK {
				// 2. 将网络流（Body）读取为 []byte 字节数组
				// 这步拿到的 fileBytes，就跟你用 os.ReadFile 读本地文件拿到的是一模一样的
				fileData, err = io.ReadAll(resp.Body)
			} else {
				err = fmt.Errorf("文件下载失败，状态码: %d", resp.StatusCode)
			}
		} else {
			err = fmt.Errorf("网络请求失败: %w", herr)
		}
	}
	if err != nil {
		return nil, err
	}

	// 2. 上传文件获取 media_id
	filename := filepath.Base(filePath)
	mediaID, err := c.UploadMedia(ctx, fileType, filename, fileData)
	if err != nil {
		return nil, fmt.Errorf("文件读取失败:%w", err)
	}

	// 3. 发送文件消息,这里用主动发送消息不要回复消息，因为回复的消息可能还在流式输出这会导致企业微信判断有2个客户端在回复数据，发生异常
	//reqID := request.Metadata["req_id"].(string)
	//cmd := WsCmdResponse
	//if reqID == "" {
	reqID := generateReqID(WsCmdSendMsg)
	cmd := WsCmdSendMsg
	//}
	body := c.handler.BuildSendMessage(event.ChatID, map[string]any{
		"msgtype": fileType,
		fileType: map[string]any{
			"media_id": mediaID,
		},
	})
	_, err = c.wsManager.SendReply(ctx, reqID, body, cmd)
	if err != nil {
		return nil, fmt.Errorf("文件发送失败:%w", err)
	}
	return &types.InputMessage{
		Content: "✅ 文件已成功发送",
	}, nil
}

// getStringFromMap 从 map 中获取字符串值
func getStringFromMap(m map[string]any, key string) (string, bool) {
	if m == nil {
		return "", false
	}
	v, ok := m[key]
	if !ok {
		return "", false
	}
	if s, ok := v.(string); ok {
		return s, true
	}
	return "", false
}
