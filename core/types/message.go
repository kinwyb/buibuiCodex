package types

import (
	"fmt"
	"strings"
	"time"
	"uuid"
)

// RoleType 消息角色
type RoleType string

const (
	// RoleAssistant 模型消息
	RoleAssistant RoleType = "assistant"
	// RoleUser 用户消息
	RoleUser RoleType = "user"
	// RoleSystem 系统消息
	RoleSystem RoleType = "system"
	// RoleTool 工具消息
	RoleTool RoleType = "tool"
)

type Message struct {
	// RoleType 消息角色
	Role RoleType `json:"role"`

	// IsDetla 是否是变化量
	IsDetla bool `json:"is_detla"`

	// ReasoningContent 思考内容
	ReasoningContent string `json:"reasoning_content,omitempty"`

	// Content 文本内容
	Content string `json:"content"`

	// ToolCalls 工具调用
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`

	// Media 媒体文件
	Media []Media `json:"media"`

	// customized information for model implementation
	Extra map[string]any `json:"extra,omitempty"`

	// 审批ID,审批时有效
	ApprovalID any `json:"approve_id,omitempty"`
}

type ToolType string

const (
	ToolTypeCommand   ToolType = "command"
	ToolTypeWebSearch ToolType = "web_search"
	ToolTypeFunction  ToolType = "function"
)

// ToolCall 工具调用
type ToolCall struct {
	// ID 调用ID
	ID string `json:"id"`
	// Type 工具类型
	Type ToolType `json:"type"`
	// Name 工具名称
	Name string `json:"name"`
	// Params 工具参数
	Params string `json:"params"`
	// Result 工具输出
	Result string `json:"result"`
	// Extra is used to store extra information for the tool call.
	Extra map[string]any `json:"extra,omitempty"`
}

type MediaType string

const (
	MediaTypeImage MediaType = "image"
	MediaTypeAudio MediaType = "audio"
	MediaTypeFile  MediaType = "document"
)

// Media 媒体文件
type Media struct {
	Type     MediaType      `json:"type"`               // image, video, audio, document
	URL      string         `json:"url"`                // 文件URL
	Base64   string         `json:"base64"`             // Base64编码内容
	MimeType string         `json:"mimetype"`           // MIME类型
	Metadata map[string]any `json:"metadata,omitempty"` // 额外元数据（如加密参数等）
}

// InputMessage 输入消息
type InputMessage struct {
	ID        string         `json:"id"`         // 请求ID
	Session   string         `json:"session"`    // session_id
	Channel   string         `json:"channel"`    // 来源渠道
	AccountID string         `json:"account_id"` // 账号ID（用于多账号场景）
	SenderID  string         `json:"sender_id"`  // 发送者ID
	ChatID    string         `json:"chat_id"`    // 聊天ID
	Content   string         `json:"content"`    // 消息内容
	Media     []Media        `json:"media"`      // 媒体文件
	Metadata  map[string]any `json:"metadata"`   // 元数据
	AgentName string         `json:"agent_name"` // agent名称
	Timestamp time.Time      `json:"timestamp"`
}

// SessionID 返回逻辑会话标识。
//
// 未显式指定 Session 时，由 channel/account/chat 和 SenderID 组成，
// 确保同一群聊中的不同发送者拥有独立会话。私聊的 ChatID 已经等于
// SenderID，此时不再重复拼接，以兼容旧的私聊会话数据。
func (input *InputMessage) SessionID() string {
	if input.Session == "" {
		sessionID := fmt.Sprintf("%s_%s_%s", input.Channel, input.AccountID, input.ChatID)
		if input.SenderID != "" && input.SenderID != input.ChatID {
			sessionID += "_" + input.SenderID
		}
		sessionID = strings.Trim(strings.TrimSpace(sessionID), "_")
		if sessionID == "" {
			sessionID = uuid.NewV4().String()
		}
		input.Session = sessionID
	}
	return input.Session
}
