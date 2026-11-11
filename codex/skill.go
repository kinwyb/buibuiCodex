package codex

import (
	"context"
	"fmt"
)

// SkillInfo 是 Codex 当前加载的技能元信息。
type SkillInfo struct {
	Name             string `json:"name"`
	Description      string `json:"description"`
	ShortDescription string `json:"shortDescription,omitempty"`
	Path             string `json:"path"`
	Scope            string `json:"scope"`
	Enabled          bool   `json:"enabled"`
	Interface        *struct {
		DisplayName string `json:"displayName,omitempty"`
	} `json:"interface,omitempty"`
}

type skillsListParams struct {
	Cwds []string `json:"cwds,omitempty"`
}

type skillsListEntry struct {
	Skills []SkillInfo `json:"skills"`
}

type skillsListResponse struct {
	Data []skillsListEntry `json:"data"`
}

type Skill struct {
	client *Client
}

// NewSkill 创建Skill
func NewSkill(client *Client) *Skill {
	return &Skill{
		client: client,
	}
}

// List 列出指定工作目录下 Codex 实际加载的技能。
func (s *Skill) List(ctx context.Context, cwd string) ([]SkillInfo, error) {
	var result skillsListResponse
	err := s.client.Call(ctx, SkillsList, skillsListParams{
		Cwds: []string{cwd},
	}, &result)
	if err != nil {
		return nil, err
	}
	if len(result.Data) == 0 {
		return nil, fmt.Errorf("skills/list 未返回 %s 的技能数据", cwd)
	}
	return result.Data[0].Skills, nil
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
