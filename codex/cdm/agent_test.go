package cdm

import (
	"testing"

	"github.com/kinwyb/buibuiCodex/codex"
	"github.com/kinwyb/buibuiCodex/core/types"
)

func TestBuildInputIncludesSelectedSkill(t *testing.T) {
	turn := codex.NewTurn(nil, nil)
	input := &types.InputMessage{
		Content: "分析订单进度",
		Skills: []types.Skill{{
			Name: "order-progress-analysis",
			Path: "/root/.codex/skills/order-progress-analysis/SKILL.md",
		}},
	}

	got := (&Agent{}).buildInput(turn, input)
	if len(got) != 2 {
		t.Fatalf("应生成技能与文本两个输入项，实际 %+v", got)
	}
	if got[0].Type != "skill" || got[0].Name != input.Skills[0].Name || got[0].Path != input.Skills[0].Path {
		t.Fatalf("技能输入项不符合预期: %+v", got[0])
	}
	if got[1].Type != "text" || got[1].Text != input.Content {
		t.Fatalf("文本输入项不符合预期: %+v", got[1])
	}
}
