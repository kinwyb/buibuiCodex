// 本文件做一次端到端验证：agent 工具 -> 界面桥 -> ws 服务 -> 浏览器 -> 回执 -> 工具文本。
//
// 它覆盖的是 main.go 里那段装配（core.NewUIBridge + RegisterUITools + SetTransport(wsServer)）：
// 单元测试分别验证了工具层与传输层，这里验证两者接起来是通的。

package server

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/kinwyb/buibuiCodex/core"
	"github.com/kinwyb/buibuiCodex/core/types"
	"github.com/kinwyb/buibuiCodex/core/ws/protocol"
)

// pickTool 按名字取工具：走的是 agent 拿到工具的同一条路径（工具组注册表）
func pickTool(t *testing.T, name string) types.Tool {
	t.Helper()
	group := types.GetToolGroupByName("ui")
	if group == nil {
		t.Fatal("ui 工具组未注册")
	}
	for _, tool := range group.Tools {
		if tool.Name() == name {
			return tool
		}
	}
	t.Fatalf("未找到工具 %s", name)
	return nil
}

// answerUIRequest 扮演浏览器：读一条 ui.request 并按给定载荷回执
func answerUIRequest(t *testing.T, conn *websocket.Conn, payload *protocol.UIResponseData) {
	t.Helper()
	reqEnv := readEnvelope(t, conn)
	if reqEnv.Kind != protocol.KindUI || reqEnv.Type != protocol.TypeUIRequest {
		t.Fatalf("应收到 ui.request，实际 %s/%s", reqEnv.Kind, reqEnv.Type)
	}
	reply := protocol.NewEnvelope(protocol.KindClient, protocol.TypeUIResponse)
	reply.ReplyTo = reqEnv.ID
	reply.SessionID = reqEnv.SessionID // 回执必须把会话原样带回
	reply.WithData(payload)
	if err := conn.WriteJSON(reply); err != nil {
		t.Fatalf("回执失败: %v", err)
	}
}

type toolResult struct {
	msg *types.InputMessage
	err error
}

// callTool 在后台执行工具（浏览器侧要在测试协程里读帧，不能互相等）
func callTool(tool types.Tool, state *types.State, params map[string]any) chan toolResult {
	done := make(chan toolResult, 1)
	go func() {
		msg, err := tool.Execute(context.Background(), state, params)
		done <- toolResult{msg: msg, err: err}
	}()
	return done
}

func TestUIBridgeEndToEnd(t *testing.T) {
	s, conn := uiTestServer(t, 2*time.Second)
	sessionID := subscribeSession(t, conn)

	bridge := core.NewUIBridge()
	bridge.SetTransport(s)
	core.RegisterUITools(bridge)

	state := &types.State{SessionID: sessionID}

	// 1) ui_observe：agent 要快照
	snapshot := `{"view":"V0189_V0279_V0280","rev":5,"actions":[{"id":"refresh","label":"刷新"}]}`
	observe := callTool(pickTool(t, "ui_observe"), state, nil)
	answerUIRequest(t, conn, &protocol.UIResponseData{
		OK:       true,
		Rev:      5,
		Snapshot: json.RawMessage(snapshot),
	})

	got := <-observe
	if got.err != nil {
		t.Fatalf("ui_observe 失败: %v", got.err)
	}
	if got.msg == nil || !strings.Contains(got.msg.Content, snapshot) {
		t.Fatalf("模型应拿到快照原文，实际 %+v", got.msg)
	}

	// 2) ui_act：agent 执行命令，页面回执行结果
	act := callTool(pickTool(t, "ui_act"), state, map[string]any{
		"cmd":    "refresh",
		"rev":    float64(5),
		"params": map[string]any{},
	})
	answerUIRequest(t, conn, &protocol.UIResponseData{
		OK:     true,
		Rev:    6,
		Result: json.RawMessage(`"已刷新"`),
	})

	got = <-act
	if got.err != nil {
		t.Fatalf("ui_act 失败: %v", got.err)
	}
	if got.msg == nil || !strings.Contains(got.msg.Content, "已刷新") || !strings.Contains(got.msg.Content, `"rev":6`) {
		t.Fatalf("模型应拿到执行结果与新版本，实际 %+v", got.msg)
	}
}
