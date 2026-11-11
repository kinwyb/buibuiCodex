// 本文件实现 agent.skills 命令：返回指定 agent 实际加载的技能。

package command

import (
	"context"
	"slices"
	"strings"

	"github.com/kinwyb/buibuiCodex/core/ws/conn"
	"github.com/kinwyb/buibuiCodex/core/ws/protocol"
)

// agentSkills 返回 agent 当前可用的技能列表。
func (r *Registry) agentSkills(ctx context.Context, c *conn.Client, env *protocol.Envelope) {
	var data protocol.AgentSkillsData
	if err := env.DecodeData(&data); err != nil {
		c.ReplyError(env, &protocol.ErrorBody{Code: protocol.CodeBadRequest, Message: "参数解析失败", Detail: err.Error()})
		return
	}

	agentName := strings.TrimSpace(data.Agent)
	if agentName == "" {
		agentName = r.deps.DefaultAgent
	}
	if agentName == "" {
		c.ReplyError(env, &protocol.ErrorBody{Code: protocol.CodeBadRequest, Message: "请指定 agent"})
		return
	}
	if r.deps.Agents == nil {
		c.ReplyError(env, &protocol.ErrorBody{Code: protocol.CodeInternal, Message: "服务未启用技能查询"})
		return
	}
	if !slices.Contains(r.deps.Agents.ListAgents(), agentName) {
		c.ReplyError(env, &protocol.ErrorBody{Code: protocol.CodeAgentNotFound, Message: "指定的 agent 不存在", Detail: agentName})
		return
	}

	skills, err := r.deps.Agents.AgentSkills(ctx, agentName)
	if err != nil {
		c.ReplyError(env, &protocol.ErrorBody{Code: protocol.CodeInternal, Message: "读取 agent 技能失败", Detail: err.Error()})
		return
	}
	items := make([]protocol.AgentSkillInfo, 0, len(skills))
	for _, skill := range skills {
		items = append(items, protocol.AgentSkillInfo{
			Name:             skill.Name,
			DisplayName:      skill.DisplayName,
			Description:      skill.Description,
			ShortDescription: skill.ShortDescription,
			Path:             skill.Path,
			Scope:            skill.Scope,
			Enabled:          skill.Enabled,
		})
	}

	c.SendEnvelope(protocol.NewEnvelope(protocol.KindSystem, protocol.TypeAgentSkills).
		WithReplyTo(env.ID).
		WithData(&protocol.AgentSkillsData{Agent: agentName, Skills: items}))
}
