package config

import "strings"

// WebSocketConfig WebSocket 服务配置
type WebSocketConfig struct {
	Enabled      bool   `json:"enabled"`       // 是否启用
	Port         int    `json:"port"`          // 监听端口，默认 8765
	Host         string `json:"host"`          // 监听地址，默认 0.0.0.0
	Path         string `json:"path"`          // 升级路径，默认 /ws
	ReadTimeout  int    `json:"read_timeout"`  // 读超时（秒），默认 60
	WriteTimeout int    `json:"write_timeout"` // 写超时（秒），默认 10

	AllowOrigins []string `json:"allow_origins"` // 允许的 Origin 白名单，为空表示不限制
	MaxClients   int      `json:"max_clients"`   // 最大连接数，默认 100
	SendBuffer   int      `json:"send_buffer"`   // 单连接发送缓冲，默认 256

	// 增量帧合并：把同一轮、同一片段的 message_delta / reasoning_delta 在一个窗口内
	// 拼成更大的增量帧再发。只降帧数、不升带宽（合并的是增量，不是全文），
	// 目的是别让 conn 的发送队列因浏览器卡顿而填满 —— 队列一满就丢最旧的 delta，
	// 界面退化成等 message_completed 一次性刷出来，现象就是「没有流式」。
	StreamCoalesce         *bool  `json:"stream_coalesce"`           // 默认开启
	StreamCoalesceMs       int    `json:"stream_coalesce_ms"`        // 合并窗口（毫秒），默认 50
	StreamCoalesceMaxBytes int    `json:"stream_coalesce_max_bytes"` // 单片段攒到多少字节立刻落地，默认 4096
	HeartbeatInterval      int    `json:"heartbeat_interval"`        // 心跳间隔（秒），默认 30
	ForwardAllChannels     bool   `json:"forward_all_channels"`      // 是否转发所有渠道的事件，默认仅转发 ws 渠道
	DefaultAgent           string `json:"default_agent"`             // 前端未指定时使用的 agent
	// PageRouter 前置界面路由：消息进 agent 之前，先用决策模型判断要不要替用户打开页面 / 点开入口。
	// 为空或未启用时完全不影响现有流程。
	PageRouter *PageRouterConfig `json:"page_router"`
}

// SetDefaults 填充默认值，不覆盖已显式配置的字段
func (c *WebSocketConfig) SetDefaults() {
	if c.Host == "" {
		c.Host = "0.0.0.0"
	}
	if c.Port == 0 {
		c.Port = 8765
	}
	if c.Path == "" {
		c.Path = "/ws"
	}
	if !strings.HasPrefix(c.Path, "/") {
		c.Path = "/" + c.Path
	}
	if c.ReadTimeout <= 0 {
		c.ReadTimeout = 60
	}
	if c.WriteTimeout <= 0 {
		c.WriteTimeout = 10
	}
	if c.MaxClients <= 0 {
		c.MaxClients = 100
	}
	if c.SendBuffer <= 0 {
		c.SendBuffer = 256
	}
	if c.HeartbeatInterval <= 0 {
		c.HeartbeatInterval = 30
	}
	if c.StreamCoalesceMs <= 0 {
		c.StreamCoalesceMs = 50
	}
	if c.StreamCoalesceMaxBytes <= 0 {
		c.StreamCoalesceMaxBytes = 4096
	}
	if c.PageRouter != nil {
		c.PageRouter.SetDefaults()
	}
}

// StreamCoalesceEnabled 是否合并增量帧（默认开）。关掉后退化成逐帧直发，
// 弱网/慢页面下会重新出现「丢最旧 delta → 整段一次性刷出」的现象。
func (c *WebSocketConfig) StreamCoalesceEnabled() bool {
	return boolOr(c.StreamCoalesce, true)
}

// PageRouterConfig 前置界面路由配置（决策模型 + 界面动作预算）。
//
// 决策模型的凭证留空时回落到环境变量（DASHSCOPE_API_KEY / DASHSCOPE_WORKSPACE_ID 等），
// 详见 decision.New。
type PageRouterConfig struct {
	Enabled bool `json:"enabled"`

	// 决策模型接入
	APIKey  string `json:"api_key"`  // 留空读 DASHSCOPE_API_KEY
	BaseURL string `json:"base_url"` // 决策模型完整接口地址，留空读 DASHSCOPE_BASE_URL
	Model   string `json:"model"`    // 默认 decision-model-preview

	// 行为开关
	DeepAction *bool `json:"deep_action"` // 打开页面后是否继续判断子入口，默认 true
	Annotate   *bool `json:"annotate"`    // 是否把已完成的界面动作附到用户消息里交给 agent，默认 true

	// 预算与门槛
	Timeout    int `json:"timeout"`     // 整个前置流程的总预算（秒），默认 8
	MaxPages   int `json:"max_pages"`   // 送给模型的候选页面数上限，默认 60
	MaxActions int `json:"max_actions"` // 送给模型的界面入口数上限，默认 40
}

// SetDefaults 填充默认值，不覆盖已显式配置的字段
func (c *PageRouterConfig) SetDefaults() {
	if c.Model == "" {
		c.Model = "decision-model-preview"
	}
	if c.Timeout <= 0 {
		c.Timeout = 8
	}
	if c.MaxPages <= 0 {
		c.MaxPages = 60
	}
	if c.MaxActions <= 0 {
		c.MaxActions = 40
	}
}

// DeepActionEnabled 打开页面后是否继续判断子入口（默认开）
func (c *PageRouterConfig) DeepActionEnabled() bool {
	return boolOr(c.DeepAction, true)
}

// AnnotateEnabled 是否把结果附到用户消息里（默认开）
func (c *PageRouterConfig) AnnotateEnabled() bool {
	return boolOr(c.Annotate, true)
}

// boolOr 取 *bool 配置，nil 时用默认值
func boolOr(v *bool, def bool) bool {
	if v == nil {
		return def
	}
	return *v
}
