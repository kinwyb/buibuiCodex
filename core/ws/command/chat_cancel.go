// 本文件实现 chat.cancel 命令：中断当前会话正在跑的 agent。

package command

import (
	"context"
	"strings"

	"github.com/kinwyb/buibuiCodex/core/ws/conn"
	"github.com/kinwyb/buibuiCodex/core/ws/protocol"
)

// chatCancel 中断指定会话当前运行中的 agent
func (r *Registry) chatCancel(_ context.Context, c *conn.Client, env *protocol.Envelope) {
	var data protocol.ChatCancelData
	if err := env.DecodeData(&data); err != nil {
		c.ReplyError(env, &protocol.ErrorBody{Code: protocol.CodeBadRequest, Message: "参数解析失败", Detail: err.Error()})
		return
	}
	userID := c.UserID()
	sessionID := strings.TrimSpace(data.SessionID)
	chatID := strings.TrimSpace(data.ChatID)
	if chatID == "" {
		chatID = c.ChatID()
	}
	if sessionID == "" && chatID != "" {
		sessionID = r.SessionKey(userID, chatID)
	}
	if sessionID == "" {
		c.ReplyError(env, &protocol.ErrorBody{Code: protocol.CodeBadRequest, Message: "缺少 sessionID 或 chatID"})
		return
	}
	if !r.ownsSession(userID, sessionID) {
		c.ReplyError(env, &protocol.ErrorBody{Code: protocol.CodeForbidden, Message: "无权访问该会话"})
		return
	}
	if r.deps.Canceller == nil {
		c.ReplyError(env, &protocol.ErrorBody{Code: protocol.CodeInternal, Message: "服务未提供会话取消能力"})
		return
	}
	r.deps.Canceller.Cancel(sessionID)
	c.ReplyAck(env, &protocol.AckData{OK: true, SessionID: sessionID})
}
