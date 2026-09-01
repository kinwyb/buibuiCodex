package docker

import (
	"bytes"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// CodexConfig 完整的 Codex 配置
type CodexConfig struct {
	OpenAIBaseURL  string                     `toml:"openai_base_url,omitempty"`
	SandboxMode    string                     `toml:"sandbox_mode,omitempty"`
	ModelProviders map[string]ModelProvider   `toml:"model_providers,omitempty"`
	MCPServers     map[string]MCPServerConfig `toml:"mcp_servers,omitempty"`
}

// ModelProvider 模型供应商配置
type ModelProvider struct {
	Name               string `toml:"name,omitempty"`
	BaseURL            string `toml:"base_url,omitempty"`
	EnvKey             string `toml:"env_key,omitempty"`
	SupportsWebsockets *bool  `toml:"supports_websockets,omitempty"`
}

// MCPServerConfig MCP 服务器配置
type MCPServerConfig struct {
	// stdio 模式
	Command *string           `toml:"command,omitempty"`
	Args    []string          `toml:"args,omitempty"`
	Env     map[string]string `toml:"env,omitempty"`
	// HTTP/SSE 模式
	URL               *string           `toml:"url,omitempty"`
	Headers           map[string]string `toml:"http_headers,omitempty"`
	BearerTokenEnvVar *string           `toml:"bearer_token_env_var,omitempty"`
}

// ensureCodexConfig 确保配置文件存在并更新指定配置
func ensureCodexConfig(codexHome string, cfg CodexConfig) error {
	if codexHome == "" {
		return nil
	}
	if err := os.MkdirAll(codexHome, 0755); err != nil {
		return fmt.Errorf("创建 CodexHome 目录失败: %w", err)
	}

	configPath := filepath.Join(codexHome, "config.toml")

	// 读取现有配置
	existing, err := loadConfig(configPath)
	if err != nil {
		return err
	}

	// 合并配置（新配置覆盖旧配置）
	merged := mergeConfig(existing, cfg)

	// 写入配置
	if err := saveConfig(configPath, merged); err != nil {
		return err
	}

	slog.Info(fmt.Sprintf("已更新 Codex 配置: %s", configPath))
	return nil
}

// loadConfig 从文件加载配置
func loadConfig(path string) (*CodexConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &CodexConfig{}, nil
		}
		return nil, fmt.Errorf("读取 config.toml 失败: %w", err)
	}

	var cfg CodexConfig
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("解析 config.toml 失败: %w", err)
	}
	return &cfg, nil
}

// saveConfig 保存配置到文件
func saveConfig(path string, cfg *CodexConfig) error {
	var buf bytes.Buffer
	encoder := toml.NewEncoder(&buf)
	if err := encoder.Encode(cfg); err != nil {
		return fmt.Errorf("编码 config.toml 失败: %w", err)
	}

	if err := os.WriteFile(path, buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("写入 config.toml 失败: %w", err)
	}
	return nil
}

// mergeConfig 合并配置，新配置覆盖旧配置
func mergeConfig(existing *CodexConfig, update CodexConfig) *CodexConfig {
	result := *existing

	// 基础字段：非空则覆盖
	if update.OpenAIBaseURL != "" {
		result.OpenAIBaseURL = update.OpenAIBaseURL
	}
	if update.SandboxMode != "" {
		result.SandboxMode = update.SandboxMode
	}

	// ModelProviders：合并 map
	if len(update.ModelProviders) > 0 {
		if result.ModelProviders == nil {
			result.ModelProviders = make(map[string]ModelProvider)
		}
		for name, provider := range update.ModelProviders {
			result.ModelProviders[name] = provider
		}
	}

	// MCPServers：合并 map
	if len(update.MCPServers) > 0 {
		if result.MCPServers == nil {
			result.MCPServers = make(map[string]MCPServerConfig)
		}
		for name, server := range update.MCPServers {
			result.MCPServers[name] = server
		}
	}

	return &result
}
