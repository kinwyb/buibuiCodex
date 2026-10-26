// 本文件实现 session.turn.replay 命令：回放单轮的完整执行过程。

package command

import (
	"context"
	"strings"

	"github.com/kinwyb/buibuiCodex/core/db"
	"github.com/kinwyb/buibuiCodex/core/ws/conn"
	"github.com/kinwyb/buibuiCodex/core/ws/protocol"
	"github.com/kinwyb/buibuiCodex/core/ws/replay"
)

// turnReplay 回放单轮的完整执行过程。
// 与 session.history 的区别：不限帧数、不截断内容，供用户点开某一轮后细看。
func (r *Registry) turnReplay(ctx context.Context, c *conn.Client, env *protocol.Envelope) {
	var data protocol.TurnReplayData
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
	if r.deps.Turns == nil {
		c.ReplyError(env, &protocol.ErrorBody{Code: protocol.CodeInternal, Message: "服务未启用历史消息"})
		return
	}

	turnID := strings.TrimSpace(data.TurnID)
	if turnID == "" {
		c.ReplyError(env, &protocol.ErrorBody{Code: protocol.CodeBadRequest, Message: "缺少 turnID"})
		return
	}
	turn := r.deps.Turns.TurnQueryByID(ctx, turnID)
	if turn == nil {
		c.ReplyError(env, &protocol.ErrorBody{Code: protocol.CodeBadRequest, Message: "轮次不存在", Detail: turnID})
		return
	}
	// 老数据可能没有回填 sessionID，此时只能依赖会话键已通过归属校验
	if turn.SessionID != "" && turn.SessionID != sessionID {
		c.ReplyError(env, &protocol.ErrorBody{Code: protocol.CodeForbidden, Message: "该轮次不属于此会话"})
		return
	}

	query := db.TurnEventQuery{
		TurnIDs:  []string{turnID},
		Methods:  replay.KeyMethods(),
		AfterSeq: data.AfterSeq,
		Limit:    data.Limit,
	}
	events, err := r.deps.Turns.TurnEventList(ctx, query)
	if err != nil {
		c.ReplyError(env, &protocol.ErrorBody{Code: protocol.CodeInternal, Message: "查询执行过程失败", Detail: err.Error()})
		return
	}

	// 详情接口不设帧数上限、不截断内容，完整还原这一轮
	frames, lossy := replay.Build(events, replay.Options{
		SessionID: sessionID,
		ReqID:     turn.ReqID,
		Agent:     turn.Agent,
		MaxFrames: -1,
		BodyLimit: -1,
	})

	result := &protocol.TurnReplayResult{
		SessionID: sessionID,
		TurnID:    turnID,
		Frames:    frames,
		HasMore:   len(events) >= query.NormalizeLimit(),
		Truncated: lossy,
	}
	if result.HasMore && len(events) > 0 {
		result.NextAfterSeq = events[len(events)-1].Seq
	}

	c.SendEnvelope(protocol.NewEnvelope(protocol.KindSystem, protocol.TypeTurnReplay).
		WithReplyTo(env.ID).
		WithSession(sessionID, "").
		WithData(result))
}
