// 本文件实现 session.history 命令：按 session -> thread -> turn 拉取历史，
// 并在 includeFrames 时附带逐帧执行过程。

package command

import (
	"context"
	"strings"
	"time"

	"github.com/kinwyb/buibuiCodex/core/db"
	"github.com/kinwyb/buibuiCodex/core/ws/conn"
	"github.com/kinwyb/buibuiCodex/core/ws/protocol"
	"github.com/kinwyb/buibuiCodex/core/ws/replay"
)

// eventsBatchSize 拉取执行过程时单次查询覆盖的轮次数量。
// 分批的目的是让单次查询的 limit 只约束一小批轮次，
// 否则几百轮的会话会被一条查询的条数上限截断，靠后的轮次拿不到帧。
const eventsBatchSize = 50

// sessionHistory 拉取会话历史，按 session -> thread -> turn 的层级返回。
// 三种用法：
//   - 打开会话：只传 sessionID，返回全部线程与轮次；
//   - 向上翻页：额外传 beforeSeq（取上一页第一条的 seq）；
//   - 断线补发：额外传 since（本地最后一条轮次的 createdAt）。
func (r *Registry) sessionHistory(ctx context.Context, c *conn.Client, env *protocol.Envelope) {
	var data protocol.SessionHistoryData
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

	// 打开会话即订阅，保证断线重连后能继续收到该会话的实时事件
	if r.deps.Hub != nil {
		r.deps.Hub.Subscribe(c, sessionID)
	}

	result, err := r.loadHistory(ctx, sessionID, data)
	if err != nil {
		c.ReplyError(env, &protocol.ErrorBody{Code: protocol.CodeInternal, Message: "查询历史消息失败", Detail: err.Error()})
		return
	}

	c.SendEnvelope(protocol.NewEnvelope(protocol.KindSystem, protocol.TypeSessionHistory).
		WithReplyTo(env.ID).
		WithSession(sessionID, "").
		WithData(result))
}

// loadHistory 按 thread 分组查询轮次
func (r *Registry) loadHistory(ctx context.Context, sessionID string, data protocol.SessionHistoryData) (*protocol.SessionHistoryResult, error) {
	query := db.TurnQuery{
		SessionID: sessionID,
		ThreadID:  strings.TrimSpace(data.ThreadID),
		BeforeSeq: data.BeforeSeq,
		Limit:     data.Limit,
	}
	if data.Since > 0 {
		query.Since = time.UnixMilli(data.Since)
	}
	limit := query.NormalizeLimit()
	result := &protocol.SessionHistoryResult{SessionID: sessionID}

	switch {
	case query.ThreadID != "":
		// 指定线程：只返回该线程
		turns, err := r.deps.Turns.TurnList(ctx, query)
		if err != nil {
			return nil, err
		}
		item := protocol.HistoryThread{
			ThreadID: query.ThreadID,
			Turns:    toHistoryTurns(turns),
			HasMore:  len(turns) >= limit,
		}
		// 线程级累计用量只在 thread 记录里，单独补一次查询
		if meta := r.deps.Turns.ThreadQueryByID(ctx, query.ThreadID); meta != nil {
			item.Agent = meta.Agent
			item.CreatedAt = millis(meta.CreateTime)
			item.LastUpdate = millis(meta.LastUpdate)
			item.Usage = threadUsageOf(*meta)
		}
		result.Threads = append(result.Threads, item)
	default:
		// 未指定线程：session -> thread -> turn
		threads, err := r.deps.Turns.ThreadList(ctx, sessionID)
		if err != nil {
			return nil, err
		}
		for _, thread := range threads {
			q := query
			q.ThreadID = thread.ThreadID
			turns, err := r.deps.Turns.TurnList(ctx, q)
			if err != nil {
				return nil, err
			}
			if len(turns) == 0 {
				continue
			}
			result.Threads = append(result.Threads, protocol.HistoryThread{
				ThreadID:   thread.ThreadID,
				Agent:      thread.Agent,
				CreatedAt:  millis(thread.CreateTime),
				LastUpdate: millis(thread.LastUpdate),
				Turns:      toHistoryTurns(turns),
				HasMore:    len(turns) >= limit,
				Usage:      threadUsageOf(thread),
			})
		}

		// 兼容没有 thread 记录的历史数据
		if len(result.Threads) == 0 {
			q := query
			q.ThreadID = ""
			turns, err := r.deps.Turns.TurnList(ctx, q)
			if err != nil {
				return nil, err
			}
			if len(turns) > 0 {
				result.Threads = append(result.Threads, protocol.HistoryThread{
					Turns:   toHistoryTurns(turns),
					HasMore: len(turns) >= limit,
				})
			}
		}
	}

	if data.IncludeFrames {
		if err := r.attachFrames(ctx, result); err != nil {
			return nil, err
		}
	}

	return result, nil
}

// attachFrames 为历史结果补上逐帧执行过程。
//
// 帧按「线程顺序 + 轮次顺序」铺平，与实时流的时序一致；
// 每轮的帧都复用同一个 ReqID，前端可以直接把它们与实时帧混在一条流水里处理。
func (r *Registry) attachFrames(ctx context.Context, result *protocol.SessionHistoryResult) error {
	var turns []protocol.HistoryTurn
	for _, thread := range result.Threads {
		turns = append(turns, thread.Turns...)
	}
	if len(turns) == 0 {
		return nil
	}

	ids := make([]string, 0, len(turns))
	for _, turn := range turns {
		if turn.TurnID != "" {
			ids = append(ids, turn.TurnID)
		}
	}
	if len(ids) == 0 {
		return nil
	}

	grouped := make(map[string][]db.TurnEvent, len(ids))
	// 分批查，避免一次 IN 太多轮次时被单条查询的 limit 截到后面的轮次没有帧
	for start := 0; start < len(ids); start += eventsBatchSize {
		end := min(start+eventsBatchSize, len(ids))
		batch := ids[start:end]
		events, err := r.deps.Turns.TurnEventList(ctx, db.TurnEventQuery{
			TurnIDs: batch,
			Methods: replay.KeyMethods(),
			Limit:   len(batch) * replay.DefaultMaxFrames,
		})
		if err != nil {
			return err
		}
		for _, event := range events {
			grouped[event.TurnID] = append(grouped[event.TurnID], event)
		}
	}

	frames := make([]*protocol.Envelope, 0, len(ids))
	truncated := false
	for _, turn := range turns {
		built, cut := replay.Build(grouped[turn.TurnID], replay.Options{
			SessionID: result.SessionID,
			ReqID:     turn.ReqID,
			Agent:     turn.Agent,
			MaxFrames: replay.DefaultMaxFrames,
			BodyLimit: replay.DefaultBodyLimit,
		})
		truncated = truncated || cut
		frames = append(frames, built...)
	}
	result.Frames = frames
	result.FramesTruncated = truncated
	return nil
}

// toHistoryTurns 转换数据库轮次为协议结构
func toHistoryTurns(list []db.SessionTurn) []protocol.HistoryTurn {
	out := make([]protocol.HistoryTurn, 0, len(list))
	for _, t := range list {
		item := protocol.HistoryTurn{
			Seq:       t.Seq,
			ReqID:     t.ReqID,
			ThreadID:  t.ThreadID,
			TurnID:    t.TurnID,
			Agent:     t.Agent,
			Question:  t.Question,
			Answer:    t.Answer,
			Completed: t.IsCompleted,
			CreatedAt: millis(t.CreatedAt),
		}
		if t.CompletedAt != nil {
			item.CompletedAt = millis(*t.CompletedAt)
		}
		item.Usage = turnUsageOf(t)
		out = append(out, item)
	}
	return out
}
