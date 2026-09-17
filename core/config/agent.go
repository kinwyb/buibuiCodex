package config

import (
	"fmt"
	"path/filepath"
	"slices"

	"github.com/kinwyb/buibuiCodex/core/db"
)

type ManagerConfig struct {
	Providers   map[string]*ProviderConfig `json:"providers"`
	Provider    string                     `json:"provider"`
	Agent       string                     `json:"agent"`
	Workspace   string                     `json:"workspace"` // 工作区根目录（所有 agent 共享）
	Agents      []*AgentConfig             `json:"agents"`
	MCP         []*MCPConfig               `json:"mcp"` //mcp配置
	DockerImage string                     `json:"docker_image"`
}

// AgentConfig agent 配置
type AgentConfig struct {
	Name         string                `json:"name"`
	Description  string                `json:"description"` // Agent 描述
	ProviderName string                `json:"provider"`
	Model        string                `json:"model"`
	Mcp          []string              `json:"mcp,omitempty"` //mcp配置
	MCPProvider  map[string]*MCPConfig `json:"mcp_provider,omitempty"`
	DockerImage  string                `json:"docker_image"`
	EnableMemory bool                  `json:"enable_memory"`
	WorkSpace    string                `json:"-"` //agent工作区
	Provider     *ProviderConfig       `json:"-"` //供应商配置
	SessionDB    db.ISession           `json:"-"`
	SkillDir     string                `json:"-"`
}

// GetAgent 获取指定名称的 agent 配置
func (cfg *ManagerConfig) GetAgent(name string) *AgentConfig {
	for _, agent := range cfg.Agents {
		if agent.Name == name {
			return agent
		}
	}
	return nil
}

// ResolveAgentConfig 解析 agent 的最终配置
// 处理 provider/model 的默认值继承
func (cfg *ManagerConfig) ResolveAgentConfig(name string) (*AgentConfig, error) {
	agent := cfg.GetAgent(name)
	if agent == nil {
		return nil, fmt.Errorf("agent '%s' not found in config", name)
	}

	// workspace 从顶层配置获取，按 agent 名称创建子目录
	if cfg.Workspace == "" {
		return nil, fmt.Errorf("agent '%s': 未配置顶层 workspace", name)
	}
	workspace := filepath.Join(cfg.Workspace, agent.Name)

	// 确定 provider
	providerName := agent.ProviderName
	if providerName == "" {
		providerName = cfg.Provider
	}
	if providerName == "" {
		return nil, fmt.Errorf("agent '%s': no provider specified and no default_provider defined", name)
	}

	// 获取 provider 配置
	provider := cfg.Providers[providerName]
	if provider == nil {
		return nil, fmt.Errorf("agent '%s': provider '%s' not found", name, providerName)
	}

	// 确定 model
	model := agent.Model
	if model == "" {
		model = provider.DefaultModel
	}
	if model == "" {
		return nil, fmt.Errorf("agent '%s': no model specified and provider '%s' has no default_model", name, providerName)
	}

	// 设置默认描述
	description := agent.Description
	if description == "" {
		description = fmt.Sprintf("Agent %s for general tasks", name)
	}

	for _, v := range cfg.MCP {
		if !v.Enabled {
			continue
		}
		if len(agent.Mcp) > 0 && !slices.Contains(agent.Mcp, v.Name) {
			continue
		}
		if agent.MCPProvider == nil {
			agent.MCPProvider = make(map[string]*MCPConfig)
		}
		agent.MCPProvider[v.Name] = v
	}
	dockerImage := agent.DockerImage
	if dockerImage == "" {
		dockerImage = cfg.DockerImage
	}

	return &AgentConfig{
		Name:         agent.Name,
		Description:  description,
		WorkSpace:    workspace,
		ProviderName: providerName,
		Provider:     provider,
		Model:        model,
		MCPProvider:  agent.MCPProvider,
		SkillDir:     filepath.Join(cfg.Workspace, "skills"),
		DockerImage:  dockerImage,
		EnableMemory: agent.EnableMemory,
	}, nil
}
