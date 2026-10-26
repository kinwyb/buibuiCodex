// Package protocol 定义 WebSocket 传输协议：信封结构、消息类型与载荷。
//
// 传输结构通过 Envelope.Kind 区分消息域（agent / system / client / 预留域），
// 新增非 agent 消息时只需扩展 Kind 与对应载荷，协议本身保持向后兼容。
package protocol

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/kinwyb/buibuiCodex/core/types"
)

// ProtocolVersion 当前协议版本，仅在大范围不兼容改动时递增
const ProtocolVersion = 1

// Kind 消息域（大类）。
// 前端必须按 Kind 路由，遇到未知 Kind 时忽略而不是报错，以便服务端平滑扩展。
type Kind string

const (
	// KindAgent agent 运行时产生的流式事件
	KindAgent Kind = "agent"
	// KindSystem 服务端控制类消息（握手、应答、错误、心跳）
	KindSystem Kind = "system"
	// KindClient 客户端上行请求
	KindClient Kind = "client"
	// KindCron 预留：定时任务相关通知
	KindCron Kind = "cron"
	// KindNotice 预留：业务通知/广播
	KindNotice Kind = "notice"
)

// IsValid Kind 是否属于已定义的枚举
func (k Kind) IsValid() bool {
	switch k {
	case KindAgent, KindSystem, KindClient, KindCron, KindNotice:
		return true
	default:
		return false
	}
}

// system 域消息类型
const (
	TypeHello    = "hello"    // 连接建立，下发客户端信息与能力
	TypeAck      = "ack"      // 上行请求受理确认
	TypePong     = "pong"     // 应用层心跳应答
	TypeError    = "error"    // 统一错误
	TypeShutdown = "shutdown" // 服务端即将关闭
)

// client 域消息类型（上行）
const (
	TypeChatSend       = "chat.send"           // 发送用户消息
	TypeChatCancel     = "chat.cancel"         // 中断当前会话
	TypePing           = "ping"                // 应用层心跳
	TypeAgentList      = "agent.list"          // 拉取可用 agent
	TypeSessionHistory = "session.history"     // 拉取历史消息（响应使用同名的 system 类型）
	TypeTurnReplay     = "session.turn.replay" // 回放单轮的完整执行过程（响应使用同名的 system 类型）
)

// 错误码
const (
	CodeBadRequest     = "bad_request"      // 请求结构或参数不合法
	CodeUnauthorized   = "unauthorized"     // 鉴权失败
	CodeForbidden      = "forbidden"        // 越权访问其他用户的会话
	CodeUnsupported    = "unsupported"      // 未支持的消息类型
	CodeAgentNotFound  = "agent_not_found"  // 指定的 agent 不存在
	CodeBusy           = "busy"             // 服务繁忙
	CodeTooManyClients = "too_many_clients" // 连接数超过上限
	CodeInternal       = "internal"         // 服务端内部错误
)

// Envelope 传输信封，上下行统一结构
type Envelope struct {
	// V 协议版本
	V int `json:"v"`
	// ID 消息唯一标识，由发送方生成
	ID string `json:"id"`
	// Kind 消息域，区分 agent / system / client 以及后续扩展域
	Kind Kind `json:"kind"`
	// Type 具体消息类型，含义由 Kind 决定
	Type string `json:"type"`
	// SessionID 会话标识，下行事件必带
	SessionID string `json:"sessionID,omitempty"`
	// ReqID 一次提问的唯一标识，对齐 types.Event.ReqID
	ReqID string `json:"reqID,omitempty"`
	// ReplyTo 请求响应关联，回填上行消息的 ID
	ReplyTo string `json:"replyTo,omitempty"`
	// Seq 服务端全局单调序号，用于排序与问题排查
	Seq int64 `json:"seq,omitempty"`
	// Chunk 同一流内的序号，对齐 types.Event.ChunkIndex
	Chunk int `json:"chunk,omitempty"`
	// TS 毫秒时间戳
	TS int64 `json:"ts"`
	// Data 具体载荷，按 Kind + Type 解释
	Data json.RawMessage `json:"data,omitempty"`
}

// NewEnvelope 创建一条消息
func NewEnvelope(kind Kind, typ string) *Envelope {
	return &Envelope{
		V:    ProtocolVersion,
		ID:   uuid.NewString(),
		Kind: kind,
		Type: typ,
		TS:   time.Now().UnixMilli(),
	}
}

// SetData 序列化并写入载荷
func (e *Envelope) SetData(data any) error {
	if data == nil {
		e.Data = nil
		return nil
	}
	raw, err := json.Marshal(data)
	if err != nil {
		return err
	}
	e.Data = raw
	return nil
}

// WithData 写入载荷，序列化失败时保持 Data 为空
func (e *Envelope) WithData(data any) *Envelope {
	_ = e.SetData(data)
	return e
}

// DecodeData 将载荷反序列化到目标结构
func (e *Envelope) DecodeData(out any) error {
	if len(e.Data) == 0 || out == nil {
		return nil
	}
	return json.Unmarshal(e.Data, out)
}

// WithSession 设置会话与请求标识
func (e *Envelope) WithSession(sessionID, reqID string) *Envelope {
	e.SessionID = sessionID
	e.ReqID = reqID
	return e
}

// WithReplyTo 设置响应关联的上行消息 ID
func (e *Envelope) WithReplyTo(id string) *Envelope {
	e.ReplyTo = id
	return e
}

// WithChunk 设置流内序号
func (e *Envelope) WithChunk(chunk int) *Envelope {
	e.Chunk = chunk
	return e
}

// IsDelta 是否属于可丢弃的增量帧，仅用于背压决策。
// 它与 AgentData.Delta 不是同一个概念：
//   - IsDelta 回答「网络拥塞时这条消息能否丢弃」，只有 message_delta / reasoning_delta 可以
//   - AgentData.Delta 是给前端的渲染指令（追加还是覆盖）
//
// 两者当前取值一致，但保持独立判断，避免将来新增可丢弃帧时误改渲染语义。
func IsDelta(kind Kind, typ string) bool {
	if kind != KindAgent {
		return false
	}
	return typ == string(types.EventMessageDelta) || typ == string(types.EventReasoningDelta)
}
