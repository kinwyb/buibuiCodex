// 本文件实现 session.list 命令：按 thread 列出当前用户的对话。

package command

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/kinwyb/buibuiCodex/core/db"
	"github.com/kinwyb/buibuiCodex/core/ws/conn"
	"github.com/kinwyb/buibuiCodex/core/ws/protocol"
)

// 对话列表的展示长度上限（按字符数截断，避免中文被切半个）
const (
	listTitleRunes   = 40
	listPreviewRunes = 80
)

// sessionList 返回当前用户的对话列表，一条对应一个 thread。
//
// 一个 thread 就是一次完整对话，其下挂多轮 turn；同一会话换过 agent 会生成
// 新的 thread，因此在列表里表现为多个对话——这与「按 thread 分」的约定一致。
// 只能看到自己的对话：归属过滤在 db 层经 sessions 表关联完成。
func (r *Registry) sessionList(ctx context.Context, c *conn.Client, env *protocol.Envelope) {
	var data protocol.SessionListData
	if err := env.DecodeData(&data); err != nil {
		c.ReplyError(env, &protocol.ErrorBody{Code: protocol.CodeBadRequest, Message: "参数解析失败", Detail: err.Error()})
		return
	}
	if r.deps.Turns == nil {
		c.ReplyError(env, &protocol.ErrorBody{Code: protocol.CodeInternal, Message: "服务未启用会话列表"})
		return
	}

	query := db.ThreadQuery{
		Channel:   r.deps.Channel,
		AccountID: c.UserID(),
		Limit:     data.Limit,
	}
	if data.BeforeLastUpdate > 0 {
		query.BeforeLastUpdate = time.UnixMilli(data.BeforeLastUpdate)
		query.BeforeThreadID = strings.TrimSpace(data.BeforeThreadID)
	}

	threads, err := r.deps.Turns.ThreadPage(ctx, query)
	if err != nil {
		c.ReplyError(env, &protocol.ErrorBody{Code: protocol.CodeInternal, Message: "查询对话列表失败", Detail: err.Error()})
		return
	}

	ids := make([]string, 0, len(threads))
	for _, thread := range threads {
		ids = append(ids, thread.ThreadID)
	}
	summaries, err := r.deps.Turns.ThreadSummaries(ctx, ids)
	if err != nil {
		// 摘要只影响标题与预览，缺了也不该让整个列表失败
		slog.Warn("查询对话摘要失败", "sessionKey", r.SessionKey(c.UserID(), c.ChatID()), "error", err)
		summaries = nil
	}

	items := make([]protocol.SessionListItem, 0, len(threads))
	for _, thread := range threads {
		summary := summaries[thread.ThreadID]
		items = append(items, protocol.SessionListItem{
			ThreadID:   thread.ThreadID,
			SessionID:  thread.SessionID,
			Agent:      thread.Agent,
			Title:      listTitle(summary.FirstQuestion),
			Preview:    truncateRunes(strings.TrimSpace(summary.LastAnswer), listPreviewRunes),
			Turns:      summary.Turns,
			Completed:  summary.LastCompleted,
			CreatedAt:  millis(thread.CreateTime),
			LastUpdate: millis(thread.LastUpdate),
			Usage:      threadUsageOf(thread),
		})
	}

	result := &protocol.SessionListResult{Items: items}
	// 返回条数触顶即保守认为还有下一页，以空列表为准
	if limit := query.NormalizeLimit(); len(threads) >= limit {
		result.HasMore = true
		last := threads[len(threads)-1]
		result.NextBeforeLastUpdate = millis(last.LastUpdate)
		result.NextBeforeThreadID = last.ThreadID
	}

	c.SendEnvelope(protocol.NewEnvelope(protocol.KindSystem, protocol.TypeSessionList).
		WithReplyTo(env.ID).
		WithData(result))
}

// listTitle 由首轮提问生成对话标题。
// 纯媒体轮次的 question 存的是 media JSON，没有展示价值，直接留空由前端兜底。
func listTitle(question string) string {
	q := strings.TrimSpace(question)
	if q == "" || strings.HasPrefix(q, "[") || strings.HasPrefix(q, "{") {
		return ""
	}
	return truncateRunes(q, listTitleRunes)
}
