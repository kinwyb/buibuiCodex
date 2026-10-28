// 本文件测试 token 用量在历史与列表接口里的暴露：
// 实时帧之外，前端刷新页面后也要能拿到「本对话累计」与「每轮消耗」。

package command

import (
	"context"
	"testing"

	"github.com/kinwyb/buibuiCodex/core/db"
	"github.com/kinwyb/buibuiCodex/core/ws/protocol"
)

func TestSessionHistoryExposesTokenUsage(t *testing.T) {
	reader := &fakeTurnReader{
		threads: []db.SessionThread{{
			ThreadID:  "t1",
			SessionID: "s1",
			Agent:     "codex",
			TokenUsage: db.TokenUsage{
				TotalTokens: 900, InputTokens: 700, OutputTokens: 200,
			},
		}},
		turns: map[string][]db.SessionTurn{
			"t1": {{
				ReqID: "r1", ThreadID: "t1", TurnID: "u1", Seq: 1,
				TokenUsage:         db.TokenUsage{TotalTokens: 120, InputTokens: 100, OutputTokens: 20},
				ModelContextWindow: 200000,
			}},
		},
	}
	reg, hub := newTestRegistryWithTurns(nil, nil, reader)
	c := newTestClient(hub)

	resp := historyRequest(t, reg, c, protocol.SessionHistoryData{SessionID: "ws_web_s1"})
	var result protocol.SessionHistoryResult
	if err := resp.DecodeData(&result); err != nil {
		t.Fatalf("解析载荷失败: %v", err)
	}
	if len(result.Threads) != 1 {
		t.Fatalf("线程数量不符: %+v", result.Threads)
	}
	thread := result.Threads[0]
	// 线程级给累计用量，前端在会话头部展示
	if thread.Usage == nil || thread.Usage.Total.TotalTokens != 900 {
		t.Fatalf("线程累计用量不符: %+v", thread.Usage)
	}
	if len(thread.Turns) != 1 {
		t.Fatalf("轮次数量不符: %+v", thread.Turns)
	}
	turn := thread.Turns[0]
	// 轮次级给本轮用量与上下文窗口，前端挂在单条回答上
	if turn.Usage == nil || turn.Usage.Last.TotalTokens != 120 {
		t.Fatalf("本轮用量不符: %+v", turn.Usage)
	}
	if turn.Usage.ModelContextWindow != 200000 {
		t.Fatalf("上下文窗口不符: %d", turn.Usage.ModelContextWindow)
	}
	if turn.Usage.Total.TotalTokens != 0 {
		t.Fatalf("轮次载荷不应重复累计用量: %+v", turn.Usage.Total)
	}
}

func TestSessionListExposesTokenUsage(t *testing.T) {
	reader := &fakeTurnReader{
		pageThreads: []db.SessionThread{{
			ThreadID:   "t1",
			SessionID:  "s1",
			Agent:      "codex",
			TokenUsage: db.TokenUsage{TotalTokens: 4321, InputTokens: 4000, OutputTokens: 321},
		}},
		summaries: map[string]db.ThreadSummary{
			"t1": {ThreadID: "t1", Turns: 1, FirstQuestion: "q1", LastAnswer: "a1", LastCompleted: true},
		},
	}
	reg, hub := newTestRegistryWithTurns(nil, nil, reader)
	c := newTestClient(hub)

	env := protocol.NewEnvelope(protocol.KindClient, protocol.TypeSessionList)
	reg.Dispatch(context.Background(), c, env)
	raw := <-c.Outbound()
	resp, body := protocol.DecodeEnvelope(raw)
	if body != nil {
		t.Fatalf("解析响应失败: %+v", body)
	}
	var result protocol.SessionListResult
	if err := resp.DecodeData(&result); err != nil {
		t.Fatalf("解析载荷失败: %v", err)
	}
	if len(result.Items) != 1 {
		t.Fatalf("列表项数量不符: %+v", result.Items)
	}
	if result.Items[0].Usage == nil || result.Items[0].Usage.Total.TotalTokens != 4321 {
		t.Fatalf("列表项累计用量不符: %+v", result.Items[0].Usage)
	}
}

// TestUsageHelpersSkipEmpty 确认没有用量记录时返回 nil，
// 避免列表/历史里出现满屏的「0 tokens」。
func TestUsageHelpersSkipEmpty(t *testing.T) {
	if got := threadUsageOf(db.SessionThread{ThreadID: "t"}); got != nil {
		t.Fatalf("空用量应返回 nil: %+v", got)
	}
	if got := turnUsageOf(db.SessionTurn{ReqID: "r"}); got != nil {
		t.Fatalf("空用量应返回 nil: %+v", got)
	}
}
