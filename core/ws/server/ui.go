// 本文件实现界面操作的反向 RPC：服务端向浏览器要快照、下发命令，并同步等待回执。
//
// 与其它上行命令的区别：普通命令是「浏览器问、服务端答」，这里是反过来的
// 「服务端问、浏览器答」，因此需要一张等待表把回执按 replyTo 认领回来。

package server

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/kinwyb/buibuiCodex/core/ws/conn"
	"github.com/kinwyb/buibuiCodex/core/ws/protocol"
)

// defaultUIRequestTimeout 界面操作请求的默认超时。
//
// 这是人机同步的等待：页面正常在几百毫秒内响应，超过 10 秒基本可以判定
// 页面已关闭、卡死或不在前台，此时应当让工具失败，而不是让模型一直等。
const defaultUIRequestTimeout = 10 * time.Second

// ErrNoUIHost 该会话当前没有可操作的页面连接。
//
// 常见原因：页面没打开、连接已断开，或页面尚未订阅该会话
// （订阅发生在 chat.send / session.history / session.list 之后）。
var ErrNoUIHost = errors.New("no ui host connected for session")

// uiPending 一次等待回执的界面操作请求
type uiPending struct {
	sessionID string
	ch        chan *protocol.UIResponseData
}

// Request 向订阅了该会话的浏览器发起一次界面操作请求，并同步等待它的回执。
//
// 这是 agent 界面工具（ui_observe / ui_act）的底座：请求经 Hub 按会话下发，
// 回执由 handleUIResponse 按 replyTo 认领。下列情况都返回 error，由工具转成
// 模型可见的结果，而不是让模型猜测发生了什么：
//
//	ErrNoUIHost  该会话没有页面连接（投递数为 0）
//	超时          页面在超时时间内没有回执
//	ctx 取消      本轮对话被中断
//	服务停止      进程退出中
//
// 已知限制：同一会话有多个页面连接时，Hub 会把请求投给全部连接，先回执者胜出。
// 需要精确到某个标签页时应引入 targetID 并在通道里消歧（当前未实现）。
func (s *Server) Request(ctx context.Context, sessionID string, req *protocol.UIRequestData) (*protocol.UIResponseData, error) {
	if sessionID == "" {
		return nil, errors.New("ui request: sessionID is required")
	}
	if req == nil || req.Op == "" {
		return nil, errors.New("ui request: op is required")
	}

	env := protocol.NewEnvelope(protocol.KindUI, protocol.TypeUIRequest).WithData(req)
	env.SessionID = sessionID

	pending := &uiPending{sessionID: sessionID, ch: make(chan *protocol.UIResponseData, 1)}
	s.uiMu.Lock()
	s.uiPending[env.ID] = pending
	s.uiMu.Unlock()

	if s.hub.Dispatch(sessionID, env) == 0 {
		s.dropUIPending(env.ID)
		return nil, ErrNoUIHost
	}

	timeout := s.uiTimeout
	if timeout <= 0 {
		timeout = defaultUIRequestTimeout
	}
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case resp := <-pending.ch:
		return resp, nil
	case <-timer.C:
		s.dropUIPending(env.ID)
		return nil, fmt.Errorf("ui request timeout after %s", timeout)
	case <-ctx.Done():
		s.dropUIPending(env.ID)
		return nil, ctx.Err()
	case <-s.stopChan():
		s.dropUIPending(env.ID)
		return nil, errors.New("ui request: server stopped")
	}
}

// handleUIResponse 实现上行命令协议.TypeUIResponse：认领等待中的请求。
//
// 校验顺序：replyTo 存在 -> 载荷可解析 -> 请求仍在等待 -> 会话一致 -> 连接归属一致。
// 迟到的回执（已超时）只记日志不回错误：回执方通常也已经放弃，回错误没有意义。
func (s *Server) handleUIResponse(_ context.Context, c *conn.Client, env *protocol.Envelope) {
	if env.ReplyTo == "" {
		c.ReplyError(env, &protocol.ErrorBody{
			Code:    protocol.CodeBadRequest,
			Message: "ui.response 缺少 replyTo",
		})
		return
	}
	var data protocol.UIResponseData
	if err := env.DecodeData(&data); err != nil {
		c.ReplyError(env, &protocol.ErrorBody{
			Code:    protocol.CodeBadRequest,
			Message: "ui.response 载荷解析失败",
			Detail:  err.Error(),
		})
		return
	}

	s.uiMu.Lock()
	pending := s.uiPending[env.ReplyTo]
	s.uiMu.Unlock()
	if pending == nil {
		s.log.Debug("ui.response 无对应请求（可能已超时）", "replyTo", env.ReplyTo, "clientID", c.ID())
		return
	}

	// 请求只投给该会话的订阅者，回执必须把会话原样带回，避免多标签页之间串号。
	if env.SessionID != pending.sessionID {
		c.ReplyError(env, &protocol.ErrorBody{
			Code:    protocol.CodeForbidden,
			Message: "ui.response 的会话与请求不一致",
		})
		return
	}
	if !s.commands.OwnsSession(c.UserID(), env.SessionID) {
		c.ReplyError(env, &protocol.ErrorBody{
			Code:    protocol.CodeForbidden,
			Message: "ui.response 不属于当前连接",
		})
		return
	}

	s.dropUIPending(env.ReplyTo)
	select {
	case pending.ch <- &data:
	default:
		// 缓冲为 1，正常不会走到这里；并发重复回执时丢弃后者即可
	}
}

// dropUIPending 移除等待项，可重复调用
func (s *Server) dropUIPending(id string) {
	s.uiMu.Lock()
	delete(s.uiPending, id)
	s.uiMu.Unlock()
}

// stopChan 返回服务停止信号；未启动时返回 nil，在 select 中永久阻塞
func (s *Server) stopChan() <-chan struct{} {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ctx == nil {
		return nil
	}
	return s.ctx.Done()
}
