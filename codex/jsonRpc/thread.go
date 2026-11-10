package jsonRpc

import "encoding/json"

// StartThreadParams 创建 Thread 请求参数
type StartThreadParams struct {
	Workspace     string        `json:"workspace"`
	Config        *ThreadConfig `json:"config,omitempty"`
	Sandbox       SandboxMode   `json:"sandbox,omitempty"`
	DynamicTools  []DynamicTool `json:"dynamicTools,omitempty"`
	Instructions  string        `json:"baseInstructions,omitempty"`
	ModelProvider string        `json:"modelProvider,omitempty"`
}

// DynamicTool 动态函数信息
type DynamicTool struct {
	Type        string            `json:"type"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Tools       []ToolDescription `json:"tools"`
}

type ToolDescription struct {
	Type         string    `json:"type"`
	Name         string    `json:"name"`
	Description  string    `json:"description"`
	DeferLoading bool      `json:"deferLoading"`
	InputSchema  ToolInput `json:"inputSchema"`
}

type ToolInput struct {
	Type       string                    `json:"type"`
	Properties map[string]ToolProperties `json:"properties"`
	Required   []string                  `json:"required,omitempty"`
}

type ToolProperties struct {
	Type        string `json:"type"`
	Description string `json:"description"`
}

type SandboxMode string

const (
	SandboxModeDangerFullAccess SandboxMode = "danger-full-access"
	SandboxModeReadOnly         SandboxMode = "read-only"
	SandboxModeWorkspaceWrite   SandboxMode = "workspace-write"
)

// ThreadConfig 动态注入的 API 供应商与模型配置
type ThreadConfig struct {
	Model          string                       `json:"model,omitempty"`
	OpenAIBaseURL  string                       `json:"openai_base_url,omitempty"`
	OpenAIAPIKey   string                       `json:"openai_api_key,omitempty"`
	MCPServers     map[string]MCPServer         `json:"mcp_servers,omitempty"`
	ModelProviders map[string]ModelProviderInfo `json:"model_providers,omitempty"`
}

// ModelProviderInfo 模型供应商配置
type ModelProviderInfo struct {
	// Base URL for the provider's OpenAI-compatible API.
	BaseURL string `json:"base_url,omitempty"`
	// Environment variable that stores the user's API key for this provider.
	EnvKey string `json:"env_key,omitempty"`
	// Friendly display name.
	Name string `json:"name,omitempty"`
	// Whether this provider supports the Responses API WebSocket transport.
	SupportsWebsockets *bool `json:"supports_websockets,omitempty"`
}

type MCPServer struct {
	// Stdio 模式字段
	Enabled *bool             `json:"enabled,omitempty"`
	Command string            `json:"command,omitempty"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
	// SSE 模式字段
	URL               string            `json:"url,omitempty"`
	Headers           map[string]string `json:"http_headers,omitempty"`
	BearerTokenEnvVar string            `json:"bearer_token_env_var,omitempty"`
}

// StartThreadResult 创建 Thread 返回结果
type StartThreadResult struct {
	Thread Thread `json:"thread"`
	Model  string `json:"model"`
}

type Thread struct {
	ID        string `json:"id"`
	SessionID string `json:"sessionId"`
	Status    struct {
		Type string `json:"type"`
	} `json:"status,omitempty"`
	Turns []Turn `json:"turns,omitempty"`
}

// ThreadReadParams 读取已存储 thread 的参数。
type ThreadReadParams struct {
	ThreadID     string `json:"threadId"`
	IncludeTurns bool   `json:"includeTurns,omitempty"`
}

// ThreadReadResult 读取 thread 的返回结果，includeTurns=true 时 Turns 包含完整历史。
type ThreadReadResult struct {
	Thread Thread `json:"thread"`
}

// ThreadTurnsListParams 分页读取已存储 thread 的 turn 历史。
type ThreadTurnsListParams struct {
	ThreadID      string `json:"threadId"`
	Limit         int    `json:"limit,omitempty"`
	SortDirection string `json:"sortDirection,omitempty"`
	ItemsView     string `json:"itemsView,omitempty"`
	Cursor        string `json:"cursor,omitempty"`
}

// ThreadTurnsPage 是 thread/turns/list 的一页结果。
type ThreadTurnsPage struct {
	Data            []Turn  `json:"data"`
	NextCursor      *string `json:"nextCursor,omitempty"`
	BackwardsCursor *string `json:"backwardsCursor,omitempty"`
}

// ThreadItem 的原始载荷保留 app-server 返回的完整字段，供事件补全时原样落库。
type ThreadItem = json.RawMessage
