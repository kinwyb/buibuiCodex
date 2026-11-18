package types

import (
	"fmt"
	"strings"
	"time"
	"uuid"

	"github.com/kinwyb/buibuiCodex/codex/jsonRpc"
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

	// UserInput 结构化提问，仅 user_input 事件携带
	UserInput *UserInputRequest `json:"user_input,omitempty"`
}

// UserInputRequest 模型向用户发起的结构化提问
type UserInputRequest struct {
	// ID 上游 JSON-RPC 请求 ID，回传答案时原样带回
	ID any `json:"id"`
	// Questions 问题列表
	Questions []UserInputQuestion `json:"questions"`
	// AutoResolutionMs 非零时上游会在超时后自动以空答案落掉该请求
	AutoResolutionMs int64 `json:"autoResolutionMs,omitempty"`
}

// UserInputQuestion 单个问题；Options 为空表示纯自由文本作答
type UserInputQuestion struct {
	ID       string            `json:"id"`
	Header   string            `json:"header,omitempty"`
	Question string            `json:"question"`
	Options  []UserInputOption `json:"options,omitempty"`
}

// UserInputOption 候选项
type UserInputOption struct {
	Label       string `json:"label"`
	Description string `json:"description,omitempty"`
}

// ToolType 工具类型，前端据此区分渲染样式
type ToolType string

const (
	ToolTypeCommand   ToolType = "command"
	ToolTypeWebSearch ToolType = "web_search"
	// ToolTypeFunction 本地动态工具（codex 的 dynamicToolCall），由本服务执行
	ToolTypeFunction ToolType = "function"
	// ToolTypeMCP MCP 工具（codex 的 mcpToolCall），由 codex 直连 MCP 服务执行，
	// 服务名放在 ToolCall.Extra["server"]
	ToolTypeMCP ToolType = "mcp"
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

// MediaFromAttachments 把 userMessage 回显解析出的附件还原成 Media，
// 实时流与历史回放共用，前端按同一 media 结构渲染文件/图片。
// 文件名放 metadata.filename，与 chat.send 上传还原出的 Media 约定一致。
func MediaFromAttachments(items []jsonRpc.InputItem) []Media {
	if len(items) == 0 {
		return nil
	}
	out := make([]Media, 0, len(items))
	for _, it := range items {
		switch it.Type {
		case "image":
			out = append(out, Media{Type: MediaTypeImage, URL: it.URL})
		case "localImage":
			out = append(out, Media{Type: MediaTypeImage, URL: it.Path})
		default:
			out = append(out, Media{
				Type:     MediaTypeFile,
				URL:      it.URL,
				Metadata: map[string]any{"filename": it.Name},
			})
		}
	}
	return out
}

// Skill 是 Codex 已加载的技能，以及本轮显式选择的技能。
type Skill struct {
	Name             string `json:"name"`
	DisplayName      string `json:"displayName,omitempty"`
	Description      string `json:"description,omitempty"`
	ShortDescription string `json:"shortDescription,omitempty"`
	Path             string `json:"path"`
	Scope            string `json:"scope,omitempty"`
	Enabled          bool   `json:"enabled"`
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
	Skills    []Skill        `json:"skills"`     // 本轮显式选择的技能
	Timestamp time.Time      `json:"timestamp"`
}

const (
	// MetadataThreadID 指定要恢复的历史 thread。
	// 仅作为内部的投递提示，agent 读取后会直接在对应 thread 下开始新 turn。
	MetadataThreadID = "thread_id"
	// MetadataNoHistory 为 true 时不向新 thread 注入历史摘要
	MetadataNoHistory = "no_history"
)

// ThreadID 返回消息指定的历史 thread。
//
// 同时兼容下划线与协议字段命名，避免调用方写错元数据键时静默开新线程。
func (input *InputMessage) ThreadID() string {
	if input == nil {
		return ""
	}
	for _, key := range []string{MetadataThreadID, "threadID"} {
		if value, ok := input.Metadata[key].(string); ok {
			if threadID := strings.TrimSpace(value); threadID != "" {
				return threadID
			}
		}
	}
	return ""
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
