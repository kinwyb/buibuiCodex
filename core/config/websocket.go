package config

import "strings"

// WebSocketConfig WebSocket 服务配置
type WebSocketConfig struct {
	Enabled      bool   `json:"enabled"`       // 是否启用
	Port         int    `json:"port"`          // 监听端口，默认 8765
	Host         string `json:"host"`          // 监听地址，默认 0.0.0.0
	Path         string `json:"path"`          // 升级路径，默认 /ws
	AuthToken    string `json:"auth_token"`    // 认证 token（可选，为空不校验）
	ReadTimeout  int    `json:"read_timeout"`  // 读超时（秒），默认 60
	WriteTimeout int    `json:"write_timeout"` // 写超时（秒），默认 10

	AllowOrigins       []string `json:"allow_origins"`        // 允许的 Origin 白名单，为空表示不限制
	MaxClients         int      `json:"max_clients"`          // 最大连接数，默认 100
	SendBuffer         int      `json:"send_buffer"`          // 单连接发送缓冲，默认 256
	HeartbeatInterval  int      `json:"heartbeat_interval"`   // 心跳间隔（秒），默认 30
	ForwardAllChannels bool     `json:"forward_all_channels"` // 是否转发所有渠道的事件，默认仅转发 ws 渠道
	DefaultAgent       string   `json:"default_agent"`        // 前端未指定时使用的 agent
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
}
