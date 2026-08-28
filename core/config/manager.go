package config

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

// Manager 配置管理器，支持运行时读取和更新配置文件
type Manager struct {
	path string
	mu   sync.RWMutex
}

// NewManager 创建配置管理器
func NewManager(path string) (*Manager, error) {
	if _, err := os.Stat(path); err != nil {
		return nil, fmt.Errorf("config file not found: %w", err)
	}
	return &Manager{path: path}, nil
}

// GetConfigJSON 读取配置文件的完整 JSON
func (m *Manager) GetConfigJSON() (json.RawMessage, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	data, err := os.ReadFile(m.path)
	if err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}
	return json.RawMessage(data), nil
}

// UpdateConfig 用新的 JSON 覆盖配置文件
func (m *Manager) UpdateConfig(config json.RawMessage) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 验证 JSON 格式有效
	var parsed interface{}
	if err := json.Unmarshal(config, &parsed); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}

	// 格式化 JSON（带缩进）
	pretty, err := json.MarshalIndent(parsed, "", "  ")
	if err != nil {
		return fmt.Errorf("format JSON: %w", err)
	}

	if err := os.WriteFile(m.path, pretty, 0644); err != nil {
		return fmt.Errorf("write config file: %w", err)
	}
	return nil
}

// GetPath 返回配置文件路径
func (m *Manager) GetPath() string {
	return m.path
}
