package config

// WebSocketConfig WebSocket 服务配置
type WebSocketConfig struct {
	Enabled      bool   `json:"enabled"`       // 是否启用，默认 true
	Port         int    `json:"port"`          // WebSocket 端口，默认 8765
	Host         string `json:"host"`          // 主机地址，默认 localhost
	Path         string `json:"path"`          // WebSocket 路径，默认 /ws
	AuthToken    string `json:"auth_token"`    // 认证 token（可选）
	ReadTimeout  int    `json:"read_timeout"`  // 读超时（秒），默认 60
	WriteTimeout int    `json:"write_timeout"` // 写超时（秒），默认 60
}
