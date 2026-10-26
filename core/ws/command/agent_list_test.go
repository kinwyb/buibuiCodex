// 本文件测试 agent.list。

package command

import (
	"testing"
)

func TestAgentInfos(t *testing.T) {
	infos := AgentInfos(fakeAgents{"a", "b"})
	if len(infos) != 2 || infos[0].Name != "a" || infos[1].Name != "b" {
		t.Fatalf("agent 列表不符合预期: %+v", infos)
	}
	if AgentInfos(nil) != nil {
		t.Fatal("空来源应返回 nil")
	}
}

// ---- 历史消息 ----
