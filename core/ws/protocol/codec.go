package protocol

import (
	"encoding/json"
	"strconv"
)

// MaxMessageSize 单条上行消息最大字节数，防止恶意超大帧
const MaxMessageSize = 1 << 20

// DecodeEnvelope 解析上行信封并做基础校验。
// 校验失败时返回原始信封（可能为 nil）和错误体，便于回执时带上 replyTo。
func DecodeEnvelope(raw []byte) (*Envelope, *ErrorBody) {
	var env Envelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil, &ErrorBody{Code: CodeBadRequest, Message: "消息不是合法的 JSON", Detail: err.Error()}
	}
	if env.V == 0 {
		env.V = ProtocolVersion
	}
	if env.V != ProtocolVersion {
		return &env, &ErrorBody{
			Code:    CodeBadRequest,
			Message: "不支持的协议版本",
			Detail:  "server protocol version is " + strconv.Itoa(ProtocolVersion),
		}
	}
	if env.ID == "" {
		return &env, &ErrorBody{Code: CodeBadRequest, Message: "缺少消息 ID"}
	}
	if env.Kind == "" || !env.Kind.IsValid() {
		return &env, &ErrorBody{Code: CodeBadRequest, Message: "缺少或非法的 kind 字段", Detail: string(env.Kind)}
	}
	if env.Type == "" {
		return &env, &ErrorBody{Code: CodeBadRequest, Message: "缺少 type 字段"}
	}
	return &env, nil
}

// EncodeEnvelope 序列化信封，序列化失败返回 nil
func EncodeEnvelope(env *Envelope) []byte {
	if env == nil {
		return nil
	}
	raw, err := json.Marshal(env)
	if err != nil {
		return nil
	}
	return raw
}
