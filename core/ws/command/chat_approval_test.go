// 本文件测试 chat.approval。

package command

import (
	"context"
	"testing"

	"github.com/kinwyb/buibuiCodex/core/ws/conn"
	"github.com/kinwyb/buibuiCodex/core/ws/protocol"
)

// fakeApprover 记录回传调用，供断言会话 / 类型还原 / 决定取值
type fakeApprover struct {
	calls        int
	sessionID    string
	approvalID   any
	decision     string
	err          error
	inputCalls   int
	inputRequest any
	inputAnswers map[string][]string
}

func (f *fakeApprover) Approve(sessionID string, approvalID any, decision string) error {
	f.calls++
	f.sessionID = sessionID
	f.approvalID = approvalID
	f.decision = decision
	return f.err
}

func (f *fakeApprover) AnswerUserInput(sessionID string, requestID any, answers map[string][]string) error {
	f.inputCalls++
	f.sessionID = sessionID
	f.inputRequest = requestID
	f.inputAnswers = answers
	return f.err
}

func newApprovalRegistry(approver Approver) (*Registry, *conn.Hub) {
	hub := conn.NewHub()
	return NewRegistry(Deps{
		Hub:          hub,
		Approver:     approver,
		Channel:      "ws",
		DefaultAgent: "codex",
	}), hub
}

func dispatchApproval(t *testing.T, reg *Registry, c *conn.Client, data *protocol.ChatApprovalData) *protocol.Envelope {
	t.Helper()
	env := protocol.NewEnvelope(protocol.KindClient, protocol.TypeChatApproval).WithData(data)
	reg.Dispatch(context.Background(), c, env)
	raw := <-c.Outbound()
	resp, err := protocol.DecodeEnvelope(raw)
	if err != nil {
		t.Fatalf("响应信封解析失败: %v", err)
	}
	return resp
}

func TestChatApprovalAcceptRestoresNumericID(t *testing.T) {
	approver := &fakeApprover{}
	reg, hub := newApprovalRegistry(approver)
	c := newTestClientAs(hub, "alice")

	resp := dispatchApproval(t, reg, c, &protocol.ChatApprovalData{
		SessionID:  "ws_alice_c1",
		ApprovalID: "42",
		Decision:   "accept",
	})

	var ack protocol.AckData
	_ = resp.DecodeData(&ack)
	if !ack.OK {
		t.Fatalf("审批回传应受理，ack=%+v", ack)
	}
	if approver.calls != 1 || approver.sessionID != "ws_alice_c1" || approver.decision != "accept" {
		t.Fatalf("回传参数不符合预期: %+v", approver)
	}
	// 数字字符串要还原成数字类型，codex 才对得上号
	if id, ok := approver.approvalID.(int64); !ok || id != 42 {
		t.Fatalf("数字审批 ID 应还原为 int64(42)，实际 %T(%v)", approver.approvalID, approver.approvalID)
	}
}

func TestChatApprovalKeepsNonNumericID(t *testing.T) {
	approver := &fakeApprover{}
	reg, hub := newApprovalRegistry(approver)
	c := newTestClientAs(hub, "alice")

	resp := dispatchApproval(t, reg, c, &protocol.ChatApprovalData{
		SessionID:  "ws_alice_c1",
		ApprovalID: "ap-1",
		Decision:   "decline",
	})

	var ack protocol.AckData
	_ = resp.DecodeData(&ack)
	if !ack.OK {
		t.Fatalf("审批回传应受理，ack=%+v", ack)
	}
	if id, ok := approver.approvalID.(string); !ok || id != "ap-1" {
		t.Fatalf("非数字审批 ID 应按字符串回传，实际 %T(%v)", approver.approvalID, approver.approvalID)
	}
	if approver.decision != "decline" {
		t.Fatalf("决定应为 decline，实际 %s", approver.decision)
	}
}

func TestChatApprovalRejectsInvalidDecision(t *testing.T) {
	approver := &fakeApprover{}
	reg, hub := newApprovalRegistry(approver)
	c := newTestClientAs(hub, "alice")

	resp := dispatchApproval(t, reg, c, &protocol.ChatApprovalData{
		SessionID:  "ws_alice_c1",
		ApprovalID: "42",
		Decision:   "maybe",
	})
	var body protocol.ErrorBody
	_ = resp.DecodeData(&body)
	if body.Code != protocol.CodeBadRequest || approver.calls != 0 {
		t.Fatalf("非法 decision 应拒绝且不回传，code=%s calls=%d", body.Code, approver.calls)
	}
}

func TestChatApprovalRequiresApprovalID(t *testing.T) {
	approver := &fakeApprover{}
	reg, hub := newApprovalRegistry(approver)
	c := newTestClientAs(hub, "alice")

	resp := dispatchApproval(t, reg, c, &protocol.ChatApprovalData{
		SessionID: "ws_alice_c1",
		Decision:  "accept",
	})
	var body protocol.ErrorBody
	_ = resp.DecodeData(&body)
	if body.Code != protocol.CodeBadRequest || approver.calls != 0 {
		t.Fatalf("缺少 approvalID 应拒绝，code=%s calls=%d", body.Code, approver.calls)
	}
}

func TestChatApprovalRejectsForeignSession(t *testing.T) {
	approver := &fakeApprover{}
	reg, hub := newApprovalRegistry(approver)
	c := newTestClientAs(hub, "alice")

	resp := dispatchApproval(t, reg, c, &protocol.ChatApprovalData{
		SessionID:  "ws_bob_c1",
		ApprovalID: "42",
		Decision:   "accept",
	})
	var body protocol.ErrorBody
	_ = resp.DecodeData(&body)
	if body.Code != protocol.CodeForbidden || approver.calls != 0 {
		t.Fatalf("跨用户审批应被拒绝，code=%s calls=%d", body.Code, approver.calls)
	}
}

func TestChatApprovalWithoutApprover(t *testing.T) {
	reg, hub := newApprovalRegistry(nil)
	c := newTestClientAs(hub, "alice")

	resp := dispatchApproval(t, reg, c, &protocol.ChatApprovalData{
		SessionID:  "ws_alice_c1",
		ApprovalID: "42",
		Decision:   "accept",
	})
	var body protocol.ErrorBody
	_ = resp.DecodeData(&body)
	if body.Code != protocol.CodeInternal {
		t.Fatalf("未提供审批能力应返回 internal，code=%s", body.Code)
	}
}
