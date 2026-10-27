// 本文件负责装配「界面前置路由」：按 websocket.page_router 配置创建决策模型客户端，
// 把 preflow.Router 接到 chat.send 上。界面动作仍走既有的 ui.request 反向 RPC，
// 所以浏览器收到的还是那三种操作（observe / navigate / act），协议与前端都不用改。

package server

import (
	"time"

	"github.com/kinwyb/buibuiCodex/core/config"
	"github.com/kinwyb/buibuiCodex/core/ws/command"
	"github.com/kinwyb/buibuiCodex/core/ws/preflow"
	"github.com/kinwyb/buibuiCodex/decision"
)

// newPreflow 创建前置路由器。未启用或决策模型配置不全时返回 nil，
// 此时 chat.send 的行为与改造前完全一致（fail-open，不因为决策模型不可用而挡消息）。
func (s *Server) newPreflow() command.Preflow {
	pcfg := s.cfg.PageRouter
	if pcfg == nil || !pcfg.Enabled {
		return nil
	}

	dec := s.opts.PreflowDecider
	model := pcfg.Model
	if dec == nil {
		client, err := newDecisionClient(pcfg)
		if err != nil {
			s.log.Warn("界面前置路由未启用", "error", err)
			return nil
		}
		dec = client
		model = client.Model
	}

	cfg := preflow.Config{
		Enabled:    true,
		Timeout:    time.Duration(pcfg.Timeout) * time.Second,
		DeepAction: pcfg.DeepActionEnabled(),
		Annotate:   pcfg.AnnotateEnabled(),
		MaxPages:   pcfg.MaxPages,
		MaxActions: pcfg.MaxActions,
	}
	s.log.Info("界面前置路由已启用",
		"model", model, "timeout", cfg.Timeout.String(),
		"deep_action", cfg.DeepAction, "annotate", cfg.Annotate)
	// transport 传 s 自己：界面请求走的就是 agent 工具在用的那条反向 RPC
	return preflow.New(cfg, dec, s, s.log)
}

// newDecisionClient 只把显式配置项传进去，留空的由 decision 回落到环境变量
// DASHSCOPE_API_KEY / DASHSCOPE_BASE_URL。
func newDecisionClient(pcfg *config.PageRouterConfig) (*decision.Client, error) {
	var opts []decision.Option
	if pcfg.APIKey != "" {
		opts = append(opts, decision.WithAPIKey(pcfg.APIKey))
	}
	if pcfg.BaseURL != "" {
		opts = append(opts, decision.WithBaseURL(pcfg.BaseURL))
	}
	if pcfg.Model != "" {
		opts = append(opts, decision.WithModel(pcfg.Model))
	}
	return decision.New(opts...)
}
