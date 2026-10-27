// 本文件实现 chat.send 命令：把用户消息投递到消息总线。

package command

import (
	"context"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/kinwyb/buibuiCodex/core/types"
	"github.com/kinwyb/buibuiCodex/core/ws/conn"
	"github.com/kinwyb/buibuiCodex/core/ws/protocol"
)

// chatSend 把用户消息投递到消息总线，由 core.Manager 交给 agent 处理
func (r *Registry) chatSend(ctx context.Context, c *conn.Client, env *protocol.Envelope) {
	var data protocol.ChatSendData
	if err := env.DecodeData(&data); err != nil {
		c.ReplyError(env, &protocol.ErrorBody{Code: protocol.CodeBadRequest, Message: "参数解析失败", Detail: err.Error()})
		return
	}
	if strings.TrimSpace(data.Content) == "" && len(data.Media) == 0 {
		c.ReplyError(env, &protocol.ErrorBody{Code: protocol.CodeBadRequest, Message: "content 与 media 不能同时为空"})
		return
	}
	if data.Agent != "" && r.deps.Agents != nil {
		if !slices.Contains(r.deps.Agents.ListAgents(), data.Agent) {
			c.ReplyError(env, &protocol.ErrorBody{Code: protocol.CodeAgentNotFound, Message: "指定的 agent 不存在", Detail: data.Agent})
			return
		}
	}

	userID := c.UserID()
	sessionID := strings.TrimSpace(data.SessionID)
	if sessionID != "" && !r.ownsSession(userID, sessionID) {
		c.ReplyError(env, &protocol.ErrorBody{Code: protocol.CodeForbidden, Message: "无权访问该会话"})
		return
	}

	chatID := strings.TrimSpace(data.ChatID)
	if chatID == "" && sessionID != "" {
		// 从会话键反推 chatID，保证连接上下文与本次会话一致
		chatID = strings.TrimPrefix(sessionID, r.deps.Channel+"_"+userID+"_")
	}
	if chatID == "" {
		// 前端只带 token 时，沿用握手阶段确定的当前对话
		chatID = c.ChatID()
	}
	if chatID == "" {
		chatID = uuid.NewString()
	}
	if sessionID == "" {
		sessionID = r.SessionKey(userID, chatID)
	}
	c.SetChatID(chatID)

	// 先订阅再发送，避免 agent 首帧早于订阅导致丢失；界面请求也依赖这条订阅
	if r.deps.Hub != nil {
		r.deps.Hub.Subscribe(c, sessionID)
	}

	reqID := uuid.NewString()
	msg := &types.InputMessage{
		ID:        reqID,
		Session:   sessionID,
		Channel:   r.deps.Channel,
		AccountID: userID,
		SenderID:  userID,
		ChatID:    chatID,
		Content:   data.Content,
		Media:     data.Media,
		Metadata:  data.Metadata,
		AgentName: data.Agent,
		Timestamp: time.Now(),
	}

	// 界面前置路由要发 ui.request 并等浏览器回执，而回执恰好要从这条连接的读循环里读进来：
	// 在读循环里等就是自己等自己。所以这里先应答受理，把消息排进所属会话的队列，
	// 由队列串行地「判断 → 打开界面 → 投递」。ack 里的 reqID 就是最终那条消息的 ID。
	if r.routeQueue != nil && strings.TrimSpace(data.Content) != "" {
		c.ReplyAck(env, &protocol.AckData{OK: true, SessionID: sessionID, ChatID: chatID, ReqID: msg.ID})
		r.routeQueue.submit(msg)
		return
	}

	if err := r.deps.Bus.PublishInput(ctx, msg); err != nil {
		c.ReplyError(env, &protocol.ErrorBody{Code: protocol.CodeInternal, Message: "消息投递失败", Detail: err.Error()})
		return
	}
	c.ReplyAck(env, &protocol.AckData{OK: true, SessionID: sessionID, ChatID: chatID, ReqID: msg.ID})
}
