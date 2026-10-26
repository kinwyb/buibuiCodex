package replay

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/kinwyb/buibuiCodex/core/db"
	"github.com/kinwyb/buibuiCodex/core/types"
	"github.com/kinwyb/buibuiCodex/core/ws/protocol"
)

// ev 构造一条落库事件，seq 由调用方递增以还原真实顺序
func ev(seq uint, method, raw string) db.TurnEvent {
	return db.TurnEvent{
		Seq:        seq,
		ThreadID:   "t1",
		TurnID:     "u1",
		Method:     method,
		RawData:    raw,
		CreateTime: time.UnixMilli(int64(1700000000000 + int(seq))),
	}
}

func decodeFrame(t *testing.T, env *protocol.Envelope) protocol.AgentData {
	t.Helper()
	var data protocol.AgentData
	if err := env.DecodeData(&data); err != nil {
		t.Fatalf("解析帧载荷失败: %v", err)
	}
	return data
}

func build(t *testing.T, events []db.TurnEvent, opts Options) ([]*protocol.Envelope, bool) {
	t.Helper()
	if opts.SessionID == "" {
		opts.SessionID = "ws_web_c1"
	}
	if opts.ReqID == "" {
		opts.ReqID = "req-1"
	}
	return Build(events, opts)
}

// 一轮完整流程：思考 → 命令 → 工具 → 回答，都应该是 agent 帧
func TestBuildFullTurnAsAgentFrames(t *testing.T) {
	frames, lossy := build(t, []db.TurnEvent{
		ev(1, "item/completed", `{"item":{"id":"i1","type":"reasoning","summary":["先看目录结构","再决定改哪里"]}}`),
		ev(2, "item/completed", `{"item":{"id":"i2","type":"commandExecution","status":"completed","command":"ls -la","aggregatedOutput":"total 0","cwd":"/work","exitCode":0,"durationMs":12}}`),
		ev(3, "item/completed", `{"item":{"id":"i3","type":"dynamicToolCall","tool":"read_file","arguments":{"path":"a.go"},"result":"package a"}}`),
		ev(4, "item/completed", `{"item":{"id":"i4","type":"agentMessage","text":"改好了"}}`),
	}, Options{})
	if lossy {
		t.Fatal("内容未超限时不应标记有损")
	}
	if len(frames) != 4 {
		t.Fatalf("应还原 4 帧，实际 %d", len(frames))
	}

	kinds := []string{
		string(types.EventReasoningCompleted),
		string(types.EventToolCompleted),
		string(types.EventToolCompleted),
		string(types.EventMessageCompleted),
	}
	for i, env := range frames {
		if env.Kind != protocol.KindAgent {
			t.Fatalf("第 %d 帧应为 agent 域，实际 %s", i, env.Kind)
		}
		if env.Type != kinds[i] {
			t.Fatalf("第 %d 帧类型应为 %s，实际 %s", i, kinds[i], env.Type)
		}
		if env.SessionID != "ws_web_c1" || env.ReqID != "req-1" {
			t.Fatalf("第 %d 帧缺少会话/请求标识: %+v", i, env)
		}
		if env.TS == 0 {
			t.Fatalf("第 %d 帧缺少时间戳", i)
		}
	}

	// 思考：summary 拼成完整文本
	if got := decodeFrame(t, frames[0]); got.Reasoning != "先看目录结构\n再决定改哪里" {
		t.Fatalf("思考内容不符合预期: %q", got.Reasoning)
	}

	// 命令：参数、输出、退出码都保留
	command := decodeFrame(t, frames[1])
	if len(command.ToolCalls) != 1 {
		t.Fatalf("命令帧应带 1 个工具调用: %+v", command)
	}
	call := command.ToolCalls[0]
	if call.Type != types.ToolTypeCommand || call.Params != "ls -la" || call.Result != "total 0" {
		t.Fatalf("命令调用不符合预期: %+v", call)
	}
	if call.Extra["cwd"] != "/work" {
		t.Fatalf("命令工作目录丢失: %+v", call.Extra)
	}

	// 动态工具：名字、参数、结果
	tool := decodeFrame(t, frames[2])
	if tool.ToolCalls[0].Name != "read_file" || tool.ToolCalls[0].Result != "package a" {
		t.Fatalf("工具调用不符合预期: %+v", tool.ToolCalls[0])
	}
	if !strings.Contains(tool.ToolCalls[0].Params, `"path":"a.go"`) {
		t.Fatalf("工具参数丢失: %s", tool.ToolCalls[0].Params)
	}
}

// delta 帧不该进入回放：它们既不是关键帧，也会让载荷膨胀几个数量级
func TestBuildSkipsDeltaEvents(t *testing.T) {
	frames, _ := build(t, []db.TurnEvent{
		ev(1, "item/reasoning/textDelta", `{"itemId":"i1","delta":"思"}`),
		ev(2, "item/agentMessage/delta", `{"itemId":"i2","delta":"答"}`),
		ev(3, "thread/tokenUsage/updated", `{"threadId":"t1","tokenUsage":{}}`),
	}, Options{})
	if len(frames) != 0 {
		t.Fatalf("delta / 用量事件不应产出帧，实际 %d 帧", len(frames))
	}

	// 但白名单必须包含关键帧，否则库里的逐 token 记录会拖慢查询
	keys := KeyMethods()
	want := map[string]bool{"item/completed": false, "item/started": false}
	for _, k := range keys {
		if _, ok := want[k]; ok {
			want[k] = true
		}
	}
	for k, found := range want {
		if !found {
			t.Fatalf("关键帧 %s 应在白名单里", k)
		}
	}
	for _, k := range keys {
		if strings.HasSuffix(k, "Delta") {
			t.Fatalf("白名单不应包含 delta: %s", k)
		}
	}
}

// 开始帧与完成帧共用同一个 item，前端按工具 ID 覆盖即可
func TestBuildKeepsStartedAndCompleted(t *testing.T) {
	frames, _ := build(t, []db.TurnEvent{
		ev(1, "item/started", `{"item":{"id":"i1","type":"commandExecution","status":"inProgress","command":"go test"}}`),
		ev(2, "item/completed", `{"item":{"id":"i1","type":"commandExecution","status":"completed","command":"go test","aggregatedOutput":"ok"}}`),
	}, Options{})
	if len(frames) != 2 {
		t.Fatalf("开始与完成帧都应保留，实际 %d", len(frames))
	}
	start := decodeFrame(t, frames[0])
	done := decodeFrame(t, frames[1])
	if start.Event != string(types.EventToolStart) || done.Event != string(types.EventToolCompleted) {
		t.Fatalf("事件类型不符合预期: %s / %s", start.Event, done.Event)
	}
	if start.ToolCalls[0].ID != done.ToolCalls[0].ID {
		t.Fatalf("同一 item 的 ID 应一致: %s != %s", start.ToolCalls[0].ID, done.ToolCalls[0].ID)
	}
	if done.ToolCalls[0].Result != "ok" {
		t.Fatalf("完成帧应带执行结果: %+v", done.ToolCalls[0])
	}
}

// 超长输出按上限截断，并打上有损标记
func TestBuildTruncatesLongBody(t *testing.T) {
	output := strings.Repeat("日志行\n", 300)
	raw, _ := json.Marshal(map[string]any{
		"item": map[string]any{
			"id": "i1", "type": "commandExecution", "command": "cat big.log", "aggregatedOutput": output,
		},
	})
	frames, lossy := build(t, []db.TurnEvent{ev(1, "item/completed", string(raw))}, Options{BodyLimit: 100})
	if !lossy {
		t.Fatal("内容被截断时应标记有损")
	}
	data := decodeFrame(t, frames[0])
	if !data.Truncated {
		t.Fatal("帧应带 truncated 标记")
	}
	if len(data.ToolCalls[0].Result) > 200 || !strings.Contains(data.ToolCalls[0].Result, "已截断") {
		t.Fatalf("截断结果不符合预期: %q", data.ToolCalls[0].Result)
	}
	// 截断不能把多字节字符切成乱码
	if !utf8.ValidString(data.ToolCalls[0].Result) {
		t.Fatal("截断后产生了非法 UTF-8")
	}
}

// 单轮详情传负数表示不截断
func TestBuildWithoutBodyLimit(t *testing.T) {
	output := strings.Repeat("x", 10000)
	raw, _ := json.Marshal(map[string]any{
		"item": map[string]any{"id": "i1", "type": "commandExecution", "command": "cat", "aggregatedOutput": output},
	})
	frames, lossy := build(t, []db.TurnEvent{ev(1, "item/completed", string(raw))},
		Options{BodyLimit: -1, MaxFrames: -1})
	if lossy {
		t.Fatal("不截断时不应标记有损")
	}
	if got := decodeFrame(t, frames[0]).ToolCalls[0].Result; got != output {
		t.Fatalf("不截断时应保留完整输出，实际 %d 字节", len(got))
	}
}

// 审批、失败、错误都要能在回放里看到，否则用户判断不了「卡在哪」
func TestBuildApprovalAndFailures(t *testing.T) {
	frames, _ := build(t, []db.TurnEvent{
		ev(1, "item/commandExecution/requestApproval", `{"itemId":"i1","reason":"需要写文件","command":["rm","-rf","tmp"]}`),
		ev(2, "item/permissions/requestApproval", `{"itemId":"i2","reason":"需要网络"}`),
		ev(3, "turn/completed", `{"threadId":"t1","turn":{"id":"u1","status":"failed","durationMs":900,"error":{"message":"模型超时"}}}`),
		ev(4, "error", `{"error":{"message":"连接中断"},"willRetry":true}`),
	}, Options{})
	if len(frames) != 4 {
		t.Fatalf("应还原 4 帧，实际 %d", len(frames))
	}

	approval := decodeFrame(t, frames[0])
	if approval.Event != string(types.EventApproval) {
		t.Fatalf("审批帧类型不符合预期: %s", approval.Event)
	}
	if !strings.Contains(approval.Content, "rm -rf tmp") || !strings.Contains(approval.Content, "需要写文件") {
		t.Fatalf("审批内容应包含命令与理由: %q", approval.Content)
	}
	if approval.Approval == nil || approval.Approval.ID != "i1" {
		t.Fatalf("审批标识丢失: %+v", approval.Approval)
	}

	failed := decodeFrame(t, frames[2])
	if failed.Event != string(types.EventError) || failed.Error == nil {
		t.Fatalf("失败轮次应产出 error 帧: %+v", failed)
	}
	if failed.Error.Message != "模型超时" {
		t.Fatalf("失败原因不符合预期: %+v", failed.Error)
	}

	retried := decodeFrame(t, frames[3])
	if retried.Error == nil || retried.Error.Message != "连接中断" {
		t.Fatalf("错误帧内容不符合预期: %+v", retried)
	}
	if retried.Meta["willRetry"] != true {
		t.Fatalf("应保留 willRetry 标记: %+v", retried.Meta)
	}
}

// 正常结束不需要额外帧：message_completed 已经代表收敛
func TestBuildSkipsSuccessfulTurnCompleted(t *testing.T) {
	frames, _ := build(t, []db.TurnEvent{
		ev(1, "turn/completed", `{"threadId":"t1","turn":{"id":"u1","status":"completed","durationMs":10}}`),
	}, Options{})
	if len(frames) != 0 {
		t.Fatalf("正常结束不应产出帧，实际 %d", len(frames))
	}
}

// 未建模的 item 类型降级为工具步骤，原始内容保留
func TestBuildUnknownItemFallsBack(t *testing.T) {
	frames, _ := build(t, []db.TurnEvent{
		ev(1, "item/completed", `{"item":{"id":"i1","type":"fileChange","changes":[{"path":"a.go"}]}}`),
	}, Options{})
	if len(frames) != 1 {
		t.Fatalf("未知类型应降级为 1 帧，实际 %d", len(frames))
	}
	data := decodeFrame(t, frames[0])
	if len(data.ToolCalls) != 1 || data.ToolCalls[0].Name != "fileChange" {
		t.Fatalf("降级帧应带原始 item 名: %+v", data.ToolCalls)
	}
	if !strings.Contains(data.ToolCalls[0].Result, "a.go") {
		t.Fatalf("原始 item 内容应保留: %q", data.ToolCalls[0].Result)
	}
	if data.Meta["unparsed"] != true {
		t.Fatalf("应标记为未解析: %+v", data.Meta)
	}
}

// 帧数上限生效并标记有损
func TestBuildHonorsMaxFrames(t *testing.T) {
	events := make([]db.TurnEvent, 0, 10)
	for i := 1; i <= 10; i++ {
		events = append(events, ev(uint(i), "item/completed", `{"item":{"id":"i","type":"reasoning","summary":["想"]}}`))
	}
	frames, lossy := build(t, events, Options{MaxFrames: 3})
	if len(frames) != 3 {
		t.Fatalf("应被截到 3 帧，实际 %d", len(frames))
	}
	if !lossy {
		t.Fatal("帧数被截时应标记有损")
	}
}

// 帧的 seq 必须回填事件序号，前端要靠它做增量游标
func TestBuildCarriesEventSeq(t *testing.T) {
	frames, _ := build(t, []db.TurnEvent{
		ev(41, "item/completed", `{"item":{"id":"i1","type":"reasoning","summary":["想"]}}`),
	}, Options{})
	if frames[0].Seq != 41 {
		t.Fatalf("帧应回填事件 seq，实际 %d", frames[0].Seq)
	}
}

// 回放帧也要带实体标识，且与实时流口径一致（item ID）
func TestBuildCarriesEventID(t *testing.T) {
	frames, _ := build(t, []db.TurnEvent{
		ev(1, "item/started", `{"item":{"id":"i7","type":"commandExecution","command":"ls"}}`),
		ev(2, "item/completed", `{"item":{"id":"i7","type":"commandExecution","command":"ls","aggregatedOutput":"a.go"}}`),
		ev(3, "item/commandExecution/requestApproval", `{"itemId":"i7","reason":"需要确认"}`),
	}, Options{})
	if len(frames) != 3 {
		t.Fatalf("应还原 3 帧，实际 %d", len(frames))
	}
	if frames[0].EventID != "i7" || frames[1].EventID != "i7" {
		t.Fatalf("同一 item 的开始/完成帧应共享 eventID: %q / %q", frames[0].EventID, frames[1].EventID)
	}
	// 审批帧的标识在 data.approval.id，不重复占用 eventID
	if frames[2].EventID != "" {
		t.Fatalf("审批帧不应带 eventID，实际 %q", frames[2].EventID)
	}
	approval := decodeFrame(t, frames[2])
	if approval.Approval == nil || approval.Approval.ID != "i7" {
		t.Fatalf("审批标识应放在 data.approval.id: %+v", approval.Approval)
	}
}
