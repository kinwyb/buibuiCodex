// 本文件测试 session.list。

package command

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/kinwyb/buibuiCodex/core/db"
	"github.com/kinwyb/buibuiCodex/core/ws/conn"
	"github.com/kinwyb/buibuiCodex/core/ws/protocol"
)

func listRequest(t *testing.T, reg *Registry, c *conn.Client, data protocol.SessionListData) *protocol.Envelope {
	t.Helper()
	env := protocol.NewEnvelope(protocol.KindClient, protocol.TypeSessionList).WithData(data)
	reg.Dispatch(context.Background(), c, env)

	select {
	case raw := <-c.Outbound():
		resp, body := protocol.DecodeEnvelope(raw)
		if body != nil {
			t.Fatalf("解析响应失败: %+v", body)
		}
		return resp
	default:
		t.Fatal("未返回响应")
		return nil
	}
}

// 一个 thread 就是一条对话，列表项要能直接拿去续聊和拉历史
func TestSessionListReturnsThreads(t *testing.T) {
	reader := &fakeTurnReader{
		pageThreads: []db.SessionThread{
			{ThreadID: "t2", SessionID: "ws_web_c2", Agent: "codex", CreateTime: time.UnixMilli(1000), LastUpdate: time.UnixMilli(3000)},
			{ThreadID: "t1", SessionID: "ws_web_c1", Agent: "coder", CreateTime: time.UnixMilli(500), LastUpdate: time.UnixMilli(2000)},
		},
		summaries: map[string]db.ThreadSummary{
			"t2": {ThreadID: "t2", Turns: 3, FirstQuestion: "分析良率", LastAnswer: "良率 98.2%", LastCompleted: true},
			"t1": {ThreadID: "t1", Turns: 1, FirstQuestion: "看下代码", LastAnswer: "", LastCompleted: false},
		},
	}
	reg, hub := newTestRegistryWithTurns(nil, nil, reader)
	c := newTestClientAs(hub, "alice")

	resp := listRequest(t, reg, c, protocol.SessionListData{})
	if resp.Type != protocol.TypeSessionList {
		t.Fatalf("响应类型不符合预期: %s", resp.Type)
	}
	var result protocol.SessionListResult
	if err := resp.DecodeData(&result); err != nil {
		t.Fatalf("解析载荷失败: %v", err)
	}
	if len(result.Items) != 2 {
		t.Fatalf("应返回 2 条对话，实际 %d", len(result.Items))
	}

	first := result.Items[0]
	if first.ThreadID != "t2" || first.SessionID != "ws_web_c2" || first.Agent != "codex" {
		t.Fatalf("列表项标识不符合预期: %+v", first)
	}
	if first.Title != "分析良率" || first.Preview != "良率 98.2%" || first.Turns != 3 || !first.Completed {
		t.Fatalf("摘要不符合预期: %+v", first)
	}
	if first.CreatedAt != 1000 || first.LastUpdate != 3000 {
		t.Fatalf("时间戳不符合预期: %+v", first)
	}
	if result.Items[1].Completed {
		t.Fatal("末轮未完成时应标记为未完成")
	}

	// 只能看自己的对话
	if len(reader.pageQueries) != 1 {
		t.Fatalf("应查询一次，实际 %d", len(reader.pageQueries))
	}
	q := reader.pageQueries[0]
	if q.AccountID != "alice" || q.Channel != "ws" {
		t.Fatalf("应按用户与渠道过滤: %+v", q)
	}
}

func TestSessionListPaginates(t *testing.T) {
	reader := &fakeTurnReader{
		pageThreads: []db.SessionThread{
			{ThreadID: "t2", SessionID: "s", LastUpdate: time.UnixMilli(3000)},
			{ThreadID: "t1", SessionID: "s", LastUpdate: time.UnixMilli(2000)},
		},
		summaries: map[string]db.ThreadSummary{},
	}
	reg, hub := newTestRegistryWithTurns(nil, nil, reader)
	c := newTestClient(hub)

	resp := listRequest(t, reg, c, protocol.SessionListData{
		Limit:            2,
		BeforeLastUpdate: 9000,
		BeforeThreadID:   "t9",
	})
	var result protocol.SessionListResult
	_ = resp.DecodeData(&result)

	if !result.HasMore {
		t.Fatal("条数触顶应标记还有下一页")
	}
	if result.NextBeforeLastUpdate != 2000 || result.NextBeforeThreadID != "t1" {
		t.Fatalf("下一页游标应指向最后一条: %+v", result)
	}

	q := reader.pageQueries[0]
	if q.Limit != 2 || q.BeforeThreadID != "t9" || q.BeforeLastUpdate.UnixMilli() != 9000 {
		t.Fatalf("翻页参数未透传: %+v", q)
	}
}

func TestSessionListTruncatesLongText(t *testing.T) {
	longQuestion := strings.Repeat("问", 100)
	longAnswer := strings.Repeat("答", 200)
	reader := &fakeTurnReader{
		pageThreads: []db.SessionThread{{ThreadID: "t1", SessionID: "s", LastUpdate: time.UnixMilli(1)}},
		summaries: map[string]db.ThreadSummary{
			"t1": {Turns: 1, FirstQuestion: longQuestion, LastAnswer: longAnswer, LastCompleted: true},
		},
	}
	reg, hub := newTestRegistryWithTurns(nil, nil, reader)
	c := newTestClient(hub)

	resp := listRequest(t, reg, c, protocol.SessionListData{})
	var result protocol.SessionListResult
	_ = resp.DecodeData(&result)

	item := result.Items[0]
	if n := len([]rune(item.Title)); n != listTitleRunes+1 {
		t.Fatalf("标题应被截到 %d 字符（含省略号），实际 %d", listTitleRunes+1, n)
	}
	if n := len([]rune(item.Preview)); n != listPreviewRunes+1 {
		t.Fatalf("预览应被截到 %d 字符（含省略号），实际 %d", listPreviewRunes+1, n)
	}
	if !strings.HasSuffix(item.Title, "…") {
		t.Fatal("截断后应有省略号")
	}
}

// 纯媒体轮次的 question 存的是 media JSON，当标题没有意义
func TestSessionListSkipsMediaTitle(t *testing.T) {
	reader := &fakeTurnReader{
		pageThreads: []db.SessionThread{{ThreadID: "t1", SessionID: "s"}},
		summaries: map[string]db.ThreadSummary{
			"t1": {Turns: 1, FirstQuestion: `[{"type":"image","url":"/tmp/a.png"}]`},
		},
	}
	reg, hub := newTestRegistryWithTurns(nil, nil, reader)
	c := newTestClient(hub)

	resp := listRequest(t, reg, c, protocol.SessionListData{})
	var result protocol.SessionListResult
	_ = resp.DecodeData(&result)
	if result.Items[0].Title != "" {
		t.Fatalf("媒体 JSON 不应作为标题: %q", result.Items[0].Title)
	}
}

func TestSessionListWithoutReader(t *testing.T) {
	reg, hub := newTestRegistryWithTurns(nil, nil, nil)
	c := newTestClient(hub)

	resp := listRequest(t, reg, c, protocol.SessionListData{})
	var body protocol.ErrorBody
	_ = resp.DecodeData(&body)
	if body.Code != protocol.CodeInternal {
		t.Fatalf("未启用时应返回 internal，实际 %s", body.Code)
	}
}
