package config

import (
	"encoding/json"
	"fmt"
	"os"
)

type Config struct {
	ManagerConfig
	Channels  *ChannelsConfig  `json:"channels"`
	Log       *LogConfig       `json:"log"`       // 日志配置
	WebSocket *WebSocketConfig `json:"websocket"` // WebSocket 服务配置
	McpServer *MCPServerConfig `json:"mcp_server"`
}

// ProviderConfig 供应商配置
type ProviderConfig struct {
	Provider     string `json:"provider"` // 可选：openai,anthropic
	APIKey       string `json:"api_key"`
	APIBaseURL   string `json:"api_base_url"`
	DefaultModel string `json:"default_model"`
}

type MCPConfig struct {
	Name string `json:"name"`
	// Stdio 模式字段
	Enabled bool              `json:"enabled,omitempty"`
	Command string            `json:"command,omitempty"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
	Auth    string            `json:"auth,omitempty"`
	// HTTP/SSE 模式字段
	URL     string            `json:"url,omitempty"`
	Headers map[string]string `json:"http_headers,omitempty"`
}

// LogConfig 日志配置
type LogConfig struct {
	Level      string `json:"level"`       // 日志级别: debug, info, warn, error
	File       string `json:"file"`        // 日志文件路径（可选），为空则输出到 stdout
	MaxSize    int    `json:"max_size"`    // 单个日志文件最大大小（MB），默认 100
	MaxBackups int    `json:"max_backups"` // 保留的旧日志文件数量，默认 3
	MaxAge     int    `json:"max_age"`     // 保留旧日志文件的最大天数，默认 7
	Compress   bool   `json:"compress"`    // 是否压缩旧日志文件，默认 false
}

// Load 从指定路径加载配置文件
func Load(path string) (*Config, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(content, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &cfg, nil
}

// GetAgentName 获取默认agent名称
func (cfg *Config) GetAgentName() string {
	if cfg.Agent != "" {
		return cfg.Agent
	}
	if len(cfg.Agents) == 0 {
		return ""
	}
	return cfg.Agents[0].Name
}
