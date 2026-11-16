// 本文件实现 chat.user_input 命令：把用户对结构化提问的作答回传给正在跑的 agent。
//
// agent 调用 request_user_input 工具时会下发 user_input 帧并阻塞这一轮，
// 客户端用 userInput.id 作为 requestID 回传答案。与 chat.approval 一样无状态：
// 不保存待答记录，答案内容由 codex 侧校验，这里只做协议层与会话归属校验。

package command

import (
	"context"
	"strings"

	"github.com/kinwyb/buibuiCodex/core/ws/conn"
	"github.com/kinwyb/buibuiCodex/core/ws/protocol"
)

// chatUserInput 回传指定会话里一条结构化提问的作答
func (r *Registry) chatUserInput(_ context.Context, c *conn.Client, env *protocol.Envelope) {
	var data protocol.ChatUserInputData
	if err := env.DecodeData(&data); err != nil {
		c.ReplyError(env, &protocol.ErrorBody{Code: protocol.CodeBadRequest, Message: "参数解析失败", Detail: err.Error()})
		return
	}
	requestID := strings.TrimSpace(data.RequestID)
	if requestID == "" {
		c.ReplyError(env, &protocol.ErrorBody{Code: protocol.CodeBadRequest, Message: "缺少 requestID"})
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
	if r.deps.Approver == nil {
		c.ReplyError(env, &protocol.ErrorBody{Code: protocol.CodeInternal, Message: "服务未提供作答回传能力"})
		return
	}

	if err := r.deps.Approver.AnswerUserInput(sessionID, approvalRequestID(requestID), data.Answers); err != nil {
		c.ReplyError(env, &protocol.ErrorBody{Code: protocol.CodeInternal, Message: "作答回传失败", Detail: err.Error()})
		return
	}
	c.ReplyAck(env, &protocol.AckData{OK: true, SessionID: sessionID})
}
