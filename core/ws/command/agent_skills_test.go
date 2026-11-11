// 本文件测试 agent.skills。

package command

import (
	"context"
	"testing"

	"github.com/kinwyb/buibuiCodex/core/types"
	"github.com/kinwyb/buibuiCodex/core/ws/protocol"
)

type fakeSkillAgents struct {
	names  []string
	skills map[string][]types.Skill
}

func (f fakeSkillAgents) ListAgents() []string { return f.names }
func (f fakeSkillAgents) AgentSkills(_ context.Context, agent string) ([]types.Skill, error) {
	return f.skills[agent], nil
}

func TestAgentSkillsReturnsLoadedSkills(t *testing.T) {
	reg, hub := newTestRegistry(nil, fakeSkillAgents{
		names: []string{"codex"},
		skills: map[string][]types.Skill{
			"codex": {{
				Name:             "order-progress-analysis",
				DisplayName:      "订单进度分析",
				Description:      "查询订单进度",
				ShortDescription: "订单分析",
				Path:             "/root/.codex/skills/order-progress-analysis/SKILL.md",
				Scope:            "user",
				Enabled:          true,
			}},
		},
	})
	c := newTestClient(hub)

	env := protocol.NewEnvelope(protocol.KindClient, protocol.TypeAgentSkills).
		WithData(&protocol.AgentSkillsData{Agent: "codex"})
	reg.Dispatch(context.Background(), c, env)

	raw := <-c.Outbound()
	resp, body := protocol.DecodeEnvelope(raw)
	if body != nil {
		t.Fatalf("解析响应失败: %+v", body)
	}
	if resp.Type != protocol.TypeAgentSkills || resp.ReplyTo != env.ID {
		t.Fatalf("agent.skills 响应不符合预期: %+v", resp)
	}
	var data protocol.AgentSkillsData
	if err := resp.DecodeData(&data); err != nil {
		t.Fatalf("解析载荷失败: %v", err)
	}
	if data.Agent != "codex" || len(data.Skills) != 1 {
		t.Fatalf("技能列表不符合预期: %+v", data)
	}
	if data.Skills[0].Name != "order-progress-analysis" || !data.Skills[0].Enabled {
		t.Fatalf("技能字段不符合预期: %+v", data.Skills[0])
	}
	if data.Skills[0].DisplayName != "订单进度分析" {
		t.Fatalf("技能展示名不符合预期: %+v", data.Skills[0])
	}
}

func TestAgentSkillsUsesDefaultAndRejectsUnknownAgent(t *testing.T) {
	reg, hub := newTestRegistry(nil, fakeSkillAgents{names: []string{"codex"}})
	c := newTestClient(hub)

	env := protocol.NewEnvelope(protocol.KindClient, protocol.TypeAgentSkills)
	reg.Dispatch(context.Background(), c, env)
	raw := <-c.Outbound()
	resp, _ := protocol.DecodeEnvelope(raw)
	var data protocol.AgentSkillsData
	_ = resp.DecodeData(&data)
	if data.Agent != "codex" {
		t.Fatalf("未指定 agent 时应使用默认值，实际 %q", data.Agent)
	}

	env = protocol.NewEnvelope(protocol.KindClient, protocol.TypeAgentSkills).
		WithData(&protocol.AgentSkillsData{Agent: "missing"})
	reg.Dispatch(context.Background(), c, env)
	raw = <-c.Outbound()
	resp, _ = protocol.DecodeEnvelope(raw)
	var errorBody protocol.ErrorBody
	_ = resp.DecodeData(&errorBody)
	if errorBody.Code != protocol.CodeAgentNotFound {
		t.Fatalf("错误码应为 agent_not_found，实际 %s", errorBody.Code)
	}
}
