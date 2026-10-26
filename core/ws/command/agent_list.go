// 本文件实现 agent.list 命令：返回可用的 agent 与默认 agent。

package command

import (
	"context"

	"github.com/kinwyb/buibuiCodex/core/ws/conn"
	"github.com/kinwyb/buibuiCodex/core/ws/protocol"
)

// AgentInfos 汇总可用的 agent 列表
func AgentInfos(src AgentSource) []protocol.AgentInfo {
	if src == nil {
		return nil
	}
	names := src.ListAgents()
	infos := make([]protocol.AgentInfo, 0, len(names))
	for _, name := range names {
		infos = append(infos, protocol.AgentInfo{Name: name})
	}
	return infos
}

// agentList 返回可用 agent 列表
func (r *Registry) agentList(_ context.Context, c *conn.Client, env *protocol.Envelope) {
	c.SendEnvelope(protocol.NewEnvelope(protocol.KindSystem, protocol.TypeAgentList).
		WithReplyTo(env.ID).
		WithData(&protocol.AgentListData{
			Agents:       AgentInfos(r.deps.Agents),
			DefaultAgent: r.deps.DefaultAgent,
		}))
}
