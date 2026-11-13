// 本文件实现 chat.approval 命令：把用户对审批的「允许 / 拒绝」回传给正在跑的 agent。
//
// agent 执行到需要授权的操作时会下发 approval 帧并阻塞这一轮，客户端用 approval.id
// 回传决定。这里只做协议层校验与会话归属校验，真正的 JSON-RPC 应答由 agent 完成。

package command

import (
	"context"
	"strconv"
	"strings"

	"github.com/kinwyb/buibuiCodex/core/ws/conn"
	"github.com/kinwyb/buibuiCodex/core/ws/protocol"
)

// 审批决定取值，对齐 jsonRpc.ApprovalDecision
const (
	approvalDecisionAccept  = "accept"
	approvalDecisionDecline = "decline"
)

// approvalRequestID 把下行的字符串审批标识还原成 codex 期望的请求 ID 类型。
//
// 下行 approval.id 只能是字符串，而 codex app-server 的审批请求 ID 是数字，
// 回传时按数字带上才能对上号；解析不成整数就按字符串回传。
// 无状态转换：不保存任何「待回传」记录，也就不会随会话累积。
func approvalRequestID(id string) any {
	if number, err := strconv.ParseInt(id, 10, 64); err == nil {
		return number
	}
	return id
}

// chatApproval 回传指定会话里一条审批的处理决定
func (r *Registry) chatApproval(_ context.Context, c *conn.Client, env *protocol.Envelope) {
	var data protocol.ChatApprovalData
	if err := env.DecodeData(&data); err != nil {
		c.ReplyError(env, &protocol.ErrorBody{Code: protocol.CodeBadRequest, Message: "参数解析失败", Detail: err.Error()})
		return
	}
	approvalID := strings.TrimSpace(data.ApprovalID)
	if approvalID == "" {
		c.ReplyError(env, &protocol.ErrorBody{Code: protocol.CodeBadRequest, Message: "缺少 approvalID"})
		return
	}
	decision := strings.TrimSpace(data.Decision)
	if decision != approvalDecisionAccept && decision != approvalDecisionDecline {
		c.ReplyError(env, &protocol.ErrorBody{Code: protocol.CodeBadRequest, Message: "decision 只能是 accept 或 decline"})
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
		c.ReplyError(env, &protocol.ErrorBody{Code: protocol.CodeInternal, Message: "服务未提供审批回传能力"})
		return
	}

	if err := r.deps.Approver.Approve(sessionID, approvalRequestID(approvalID), decision); err != nil {
		c.ReplyError(env, &protocol.ErrorBody{Code: protocol.CodeInternal, Message: "审批回传失败", Detail: err.Error()})
		return
	}
	c.ReplyAck(env, &protocol.AckData{OK: true, SessionID: sessionID})
}
