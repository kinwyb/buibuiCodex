package codex

import "context"

type Skill struct {
	client *Client
}

// NewSkill 创建Skill
func NewSkill(client *Client) *Skill {
	return &Skill{
		client: client,
	}
}

// Enable 启用skill,skillpath是指skill.md的完整路径
func (s *Skill) Enable(ctx context.Context, skillpath string) error {
	params := map[string]any{
		"path":    skillpath,
		"enabled": true,
	}
	return s.client.Call(ctx, SkillsConfigWrite, params, nil)
}

// Disable 禁用skill,skillpath是指skill.md的完整路径
func (s *Skill) Disable(ctx context.Context, skillpath string) error {
	params := map[string]any{
		"path":    skillpath,
		"enabled": false,
	}
	return s.client.Call(ctx, SkillsConfigWrite, params, nil)
}
