// 本文件实现 session.delete 命令：软删除当前用户的一个对话线程。

package command

import (
	"context"
	"strings"

	"github.com/kinwyb/buibuiCodex/core/ws/conn"
	"github.com/kinwyb/buibuiCodex/core/ws/protocol"
)

// sessionDelete 把 thread 标记为 isdel，历史轮次与事件仍保留在库里。
func (r *Registry) sessionDelete(ctx context.Context, c *conn.Client, env *protocol.Envelope) {
	var data protocol.SessionDeleteData
	if err := env.DecodeData(&data); err != nil {
		c.ReplyError(env, &protocol.ErrorBody{Code: protocol.CodeBadRequest, Message: "参数解析失败", Detail: err.Error()})
		return
	}

	threadID := strings.TrimSpace(data.ThreadID)
	if threadID == "" {
		c.ReplyError(env, &protocol.ErrorBody{Code: protocol.CodeBadRequest, Message: "缺少 threadID"})
		return
	}
	if r.deps.Turns == nil {
		c.ReplyError(env, &protocol.ErrorBody{Code: protocol.CodeInternal, Message: "服务未启用会话删除"})
		return
	}

	thread := r.deps.Turns.ThreadQueryByID(ctx, threadID)
	if thread == nil {
		c.ReplyError(env, &protocol.ErrorBody{Code: protocol.CodeBadRequest, Message: "对话不存在或已删除", Detail: threadID})
		return
	}
	if !r.ownsSession(c.UserID(), thread.SessionID) {
		c.ReplyError(env, &protocol.ErrorBody{Code: protocol.CodeForbidden, Message: "无权删除该会话"})
		return
	}
	if err := r.deps.Turns.ThreadDelete(ctx, thread.SessionID, threadID); err != nil {
		c.ReplyError(env, &protocol.ErrorBody{Code: protocol.CodeInternal, Message: "删除对话失败", Detail: err.Error()})
		return
	}

	c.SendEnvelope(protocol.NewEnvelope(protocol.KindSystem, protocol.TypeSessionDelete).
		WithReplyTo(env.ID).
		WithSession(thread.SessionID, "").
		WithData(&protocol.SessionDeleteResult{ThreadID: threadID, Deleted: true}))
}
