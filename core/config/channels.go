package config

// ChannelsConfig 通道配置
type ChannelsConfig struct {
	Telegram       *TelegramChannelConfig `json:"telegram"`
	WhatsApp       *WhatsAppChannelConfig `json:"whatsapp"`
	Feishu         *FeishuChannelConfig   `json:"feishu"`
	CLI            *CLIChannelConfig      `json:"cli"`
	WxCom          *WxComChannelConfig    `json:"wxcom"`
	ThreadBindings []ThreadBindingConfig  `json:"thread_bindings"`
}

// WxComChannelConfig 企业微信通道配置
type WxComChannelConfig struct {
	Enabled  bool                          `json:"enabled"`
	Accounts map[string]WxComAccountConfig `json:"accounts"` // 所有账号从这里配置
}

// WxComAccountConfig 企业微信账号配置
type WxComAccountConfig struct {
	Enabled           bool     `json:"enabled"`
	BotID             string   `json:"bot_id"`
	Secret            string   `json:"secret"`
	WSURL             string   `json:"ws_url,omitempty"`             // 可选，自定义 WebSocket 地址
	HeartbeatInterval int      `json:"heartbeat_interval,omitempty"` // 可选，心跳间隔(ms)
	ReconnectInterval int      `json:"reconnect_interval,omitempty"` // 可选，重连延迟(ms)
	MaxReconnect      int      `json:"max_reconnect,omitempty"`      // 可选，最大重连次数
	RequestTimeout    int      `json:"request_timeout,omitempty"`    // 可选，请求超时(ms)
	AllowedIDs        []string `json:"allowed_ids"`
	DefaultAgent      string   `json:"default_agent"` // 可选，默认处理的 agent 名称
}

// AccountConfig 通道账号配置（支持多账号）
type AccountConfig struct {
	Enabled    bool     `json:"enabled"`
	Name       string   `json:"name"` // 账号显示名称
	AllowedIDs []string `json:"allowed_ids"`
	// Telegram 专用
	Token string `json:"token"`
	// WhatsApp 专用
	BridgeURL string `json:"bridge_url"`
	// Feishu 专用
	AppID             string `json:"app_id"`
	AppSecret         string `json:"app_secret"`
	EncryptKey        string `json:"encrypt_key"`
	VerificationToken string `json:"verification_token"`
	WebhookPort       int    `json:"webhook_port"`
	// 默认处理的 agent 名称
	DefaultAgent string `json:"default_agent"`
}

// TelegramChannelConfig Telegram 通道配置
type TelegramChannelConfig struct {
	Enabled    bool                     `json:"enabled"`
	Token      string                   `json:"token"`
	AllowedIDs []string                 `json:"allowed_ids"`
	Accounts   map[string]AccountConfig `json:"accounts"` // 多账号配置
}

// WhatsAppChannelConfig WhatsApp 通道配置
type WhatsAppChannelConfig struct {
	Enabled    bool                     `json:"enabled"`
	BridgeURL  string                   `json:"bridge_url"`
	AllowedIDs []string                 `json:"allowed_ids"`
	Accounts   map[string]AccountConfig `json:"accounts"` // 多账号配置
}

// FeishuChannelConfig 飞书通道配置
type FeishuChannelConfig struct {
	Enabled           bool                     `json:"enabled"`
	AppID             string                   `json:"app_id"`
	AppSecret         string                   `json:"app_secret"`
	EncryptKey        string                   `json:"encrypt_key"`
	VerificationToken string                   `json:"verification_token"`
	WebhookPort       int                      `json:"webhook_port"`
	AllowedIDs        []string                 `json:"allowed_ids"`
	Accounts          map[string]AccountConfig `json:"accounts"` // 多账号配置
}

// CLIChannelConfig CLI 通道配置
type CLIChannelConfig struct {
	Enabled      bool     `json:"enabled"`
	AllowedIDs   []string `json:"allowed_ids"`   // CLI 通常不需要限制
	DefaultAgent string   `json:"default_agent"` // 可选，默认处理的 agent 名称
}

// ThreadBindingConfig 会话绑定配置
type ThreadBindingConfig struct {
	SessionKey    string `json:"session_key"`    // Channel:ChatID (如 "tui:chat123")
	TargetChannel string `json:"target_channel"` // 目标通道名称 (如 "telegram")
	TargetAgent   string `json:"target_agent"`   // 可选：指定 agent
	Priority      int    `json:"priority"`       // 优先级
}
