// 本文件测试 chat.user_input。

package command

import (
	"context"
	"testing"

	"github.com/kinwyb/buibuiCodex/core/ws/conn"
	"github.com/kinwyb/buibuiCodex/core/ws/protocol"
)

func dispatchUserInput(t *testing.T, reg *Registry, c *conn.Client, data *protocol.ChatUserInputData) *protocol.Envelope {
	t.Helper()
	env := protocol.NewEnvelope(protocol.KindClient, protocol.TypeChatUserInput).WithData(data)
	reg.Dispatch(context.Background(), c, env)
	raw := <-c.Outbound()
	resp, err := protocol.DecodeEnvelope(raw)
	if err != nil {
		t.Fatalf("响应信封解析失败: %v", err)
	}
	return resp
}

func TestChatUserInputForwardsAnswers(t *testing.T) {
	approver := &fakeApprover{}
	reg, hub := newApprovalRegistry(approver)
	c := newTestClientAs(hub, "alice")

	resp := dispatchUserInput(t, reg, c, &protocol.ChatUserInputData{
		SessionID: "ws_alice_c1",
		RequestID: "7",
		Answers:   map[string][]string{"q1": {"选项A"}, "q2": {"自由文本"}},
	})

	var ack protocol.AckData
	_ = resp.DecodeData(&ack)
	if !ack.OK {
		t.Fatalf("作答回传应受理，ack=%+v", ack)
	}
	if approver.inputCalls != 1 || approver.sessionID != "ws_alice_c1" {
		t.Fatalf("回传参数不符合预期: %+v", approver)
	}
	if id, ok := approver.inputRequest.(int64); !ok || id != 7 {
		t.Fatalf("数字请求 ID 应还原为 int64(7)，实际 %T(%v)", approver.inputRequest, approver.inputRequest)
	}
	if got := approver.inputAnswers["q1"]; len(got) != 1 || got[0] != "选项A" {
		t.Fatalf("答案应原样透传: %+v", approver.inputAnswers)
	}
}

func TestChatUserInputAllowsEmptyAnswers(t *testing.T) {
	approver := &fakeApprover{}
	reg, hub := newApprovalRegistry(approver)
	c := newTestClientAs(hub, "alice")

	resp := dispatchUserInput(t, reg, c, &protocol.ChatUserInputData{
		SessionID: "ws_alice_c1",
		RequestID: "7",
	})

	var ack protocol.AckData
	_ = resp.DecodeData(&ack)
	if !ack.OK || approver.inputCalls != 1 {
		t.Fatalf("空答案表示跳过，应受理并回传，ack=%+v calls=%d", ack, approver.inputCalls)
	}
}

func TestChatUserInputRequiresRequestID(t *testing.T) {
	approver := &fakeApprover{}
	reg, hub := newApprovalRegistry(approver)
	c := newTestClientAs(hub, "alice")

	resp := dispatchUserInput(t, reg, c, &protocol.ChatUserInputData{
		SessionID: "ws_alice_c1",
		Answers:   map[string][]string{"q1": {"A"}},
	})
	var body protocol.ErrorBody
	_ = resp.DecodeData(&body)
	if body.Code != protocol.CodeBadRequest || approver.inputCalls != 0 {
		t.Fatalf("缺少 requestID 应拒绝，code=%s calls=%d", body.Code, approver.inputCalls)
	}
}

func TestChatUserInputRejectsForeignSession(t *testing.T) {
	approver := &fakeApprover{}
	reg, hub := newApprovalRegistry(approver)
	c := newTestClientAs(hub, "alice")

	resp := dispatchUserInput(t, reg, c, &protocol.ChatUserInputData{
		SessionID: "ws_bob_c1",
		RequestID: "7",
	})
	var body protocol.ErrorBody
	_ = resp.DecodeData(&body)
	if body.Code != protocol.CodeForbidden || approver.inputCalls != 0 {
		t.Fatalf("跨用户作答应被拒绝，code=%s calls=%d", body.Code, approver.inputCalls)
	}
}
