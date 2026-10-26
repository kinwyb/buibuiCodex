// 本文件测试界面工具（ui_observe / ui_act）：请求怎么发出去、回执怎么变成模型可读的文本。
// 用假传输层顶上，不依赖 WebSocket 与网络。

package core

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/kinwyb/buibuiCodex/core/types"
	"github.com/kinwyb/buibuiCodex/core/ws/protocol"
)

// fakeUITransport 记录收到的请求并返回预置回执
type fakeUITransport struct {
	sessionID string
	req       *protocol.UIRequestData
	resp      *protocol.UIResponseData
	err       error
}

func (f *fakeUITransport) Request(_ context.Context, sessionID string, req *protocol.UIRequestData) (*protocol.UIResponseData, error) {
	f.sessionID = sessionID
	f.req = req
	return f.resp, f.err
}

func testUIBridge(transport UITransport) *UIBridge {
	bridge := NewUIBridge()
	bridge.SetTransport(transport)
	return bridge
}

// testState 会话键由调用方给定：工具就是拿它当路由键用的
func testState(sessionID string) *types.State {
	return &types.State{SessionID: sessionID}
}

func TestUIObserveReturnsSnapshot(t *testing.T) {
	snapshot := `{"view":"V0189_V0279_V0280","rev":3}`
	transport := &fakeUITransport{resp: &protocol.UIResponseData{
		OK:       true,
		Rev:      3,
		Snapshot: json.RawMessage(snapshot),
	}}
	tool := &UIObserveTool{bridge: testUIBridge(transport)}

	msg, err := tool.Execute(context.Background(), testState("ws_u1_c1"), nil)
	if err != nil {
		t.Fatalf("取快照应成功，实际失败: %v", err)
	}
	if transport.sessionID != "ws_u1_c1" {
		t.Fatalf("应按会话路由，实际 %q", transport.sessionID)
	}
	if transport.req == nil || transport.req.Op != "observe" {
		t.Fatalf("应下发 observe 请求，实际 %+v", transport.req)
	}

	var out struct {
		OK       bool            `json:"ok"`
		Rev      int64           `json:"rev"`
		Snapshot json.RawMessage `json:"snapshot"`
	}
	if err := json.Unmarshal([]byte(msg.Content), &out); err != nil {
		t.Fatalf("返回内容应是 JSON: %v (%s)", err, msg.Content)
	}
	if !out.OK || out.Rev != 3 {
		t.Fatalf("应回传 ok 与 rev，实际 %+v", out)
	}
	if string(out.Snapshot) != snapshot {
		t.Fatalf("快照应原样交给模型，实际 %s", out.Snapshot)
	}
}

func TestUIActPassesCommandParamsAndRev(t *testing.T) {
	transport := &fakeUITransport{resp: &protocol.UIResponseData{
		OK:     true,
		Rev:    4,
		Result: json.RawMessage(`"查询条件已更新"`),
	}}
	tool := &UIActTool{bridge: testUIBridge(transport)}

	msg, err := tool.Execute(context.Background(), testState("ws_u1_c1"), map[string]any{
		"cmd":    "filter.set",
		"params": map[string]any{"workPlace": "越南"},
		// 模型传来的数字经 JSON 解析后是 float64，工具需要能吃下
		"rev": float64(3),
	})
	if err != nil {
		t.Fatalf("执行命令应成功，实际失败: %v", err)
	}
	if transport.req.Op != "act" || transport.req.Cmd != "filter.set" {
		t.Fatalf("请求不符合预期: %+v", transport.req)
	}
	if transport.req.Rev != 3 {
		t.Fatalf("rev 应原样透传，实际 %d", transport.req.Rev)
	}
	var params map[string]any
	if err := json.Unmarshal(transport.req.Params, &params); err != nil {
		t.Fatalf("解析参数失败: %v", err)
	}
	if params["workPlace"] != "越南" {
		t.Fatalf("参数应原样透传，实际 %+v", params)
	}
	if !strings.Contains(msg.Content, "查询条件已更新") {
		t.Fatalf("应把执行结果交给模型，实际 %s", msg.Content)
	}
}

func TestUIActFailureIsReturnedAsContent(t *testing.T) {
	// 页面侧拒绝（rev 过期、命令不可用）属于业务失败：必须以文本形式交给模型让它调整，
	// 而不是当成 Go error 抛出去 —— 否则模型只看到「工具报错」，不知道该怎么改。
	transport := &fakeUITransport{resp: &protocol.UIResponseData{
		OK:    false,
		Error: "页面状态已变化，请重新 ui_observe",
		Rev:   9,
	}}
	tool := &UIActTool{bridge: testUIBridge(transport)}

	msg, err := tool.Execute(context.Background(), testState("ws_u1_c1"), map[string]any{"cmd": "refresh"})
	if err != nil {
		t.Fatalf("业务失败不应作为工具错误返回: %v", err)
	}
	if !strings.Contains(msg.Content, "请重新 ui_observe") {
		t.Fatalf("应把失败原因交给模型，实际 %s", msg.Content)
	}
	if !strings.Contains(msg.Content, `"ok":false`) {
		t.Fatalf("失败回执应带 ok=false，实际 %s", msg.Content)
	}
}

func TestUIRequiresSessionAndCommand(t *testing.T) {
	transport := &fakeUITransport{}
	bridge := testUIBridge(transport)

	if _, err := (&UIObserveTool{bridge: bridge}).Execute(context.Background(), testState(""), nil); err == nil {
		t.Fatal("没有会话上下文时应失败")
	}
	if _, err := (&UIActTool{bridge: bridge}).Execute(context.Background(), testState("ws_u1_c1"), map[string]any{}); err == nil {
		t.Fatal("缺少 cmd 时应失败")
	}
	// 传输层未注入（ws 还没启动）也要是可读错误，而不是 panic
	if _, err := (&UIObserveTool{bridge: NewUIBridge()}).Execute(context.Background(), testState("ws_u1_c1"), nil); err == nil {
		t.Fatal("通道未就绪时应失败")
	}
}
