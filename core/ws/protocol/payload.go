// Package protocol 的载荷定义：按 Kind + Type 组织各类消息的 Data 结构。
//
// 流式文本的渲染约定见 AgentData 的文档。
package protocol

import (
	"encoding/json"

	"github.com/kinwyb/buibuiCodex/core/types"
)

// AgentData KindAgent 的载荷。
// 只暴露前端需要的字段，避免 channel / accountID 等内部概念外泄。
//
// # 流式渲染语义
//
// 前端渲染只需要看 Delta 一个字段，不要按事件类型自行推断：
//
//	Delta == true   增量片段，把 Content / Reasoning 「追加」到当前消息
//	Delta == false  全量快照，用 Content / Reasoning 「整体覆盖」当前消息
//
// 各事件的取值：
//
//	message_start        Delta=false，Content 通常为空，用于重置当前气泡
//	message_delta        Delta=true，Content 为增量文本，追加
//	message_completed    Delta=false，Content 为完整文本，覆盖即收敛
//	reasoning_delta      Delta=true，Reasoning 为增量思考，追加
//	reasoning_completed  Delta=false，Reasoning 为完整思考，覆盖
//	tool_start / tool_completed / approval / error / finished
//	                     Delta=false，属于状态或事件，不参与文本追加
//	token_usage          Delta=false，Content 为空，只更新 Usage 面板
//
// 该约定保证：中间 delta 即使因背压被丢弃，前端在 message_completed 时
// 用全量内容覆盖，界面仍能收敛到正确结果。
//
// 示例：
//
//	// 增量：追加「生产」
//	{"v":1,"kind":"agent","type":"message_delta","sessionID":"ws_web_c1","seq":18,"chunk":7,
//	 "data":{"event":"message_delta","delta":true,"content":"生产"}}
//
//	// 完成：用全量覆盖整条消息
//	{"v":1,"kind":"agent","type":"message_completed","sessionID":"ws_web_c1","seq":42,"chunk":31,
//	 "data":{"event":"message_completed","delta":false,"content":"生产良率为 98.2%"}}
type AgentData struct {
	// Event 事件类型，与 types.EventType 取值一致
	Event string `json:"event"`
	// Agent 产生该事件的 agent 名称
	Agent string `json:"agent,omitempty"`
	// Role 消息角色（assistant / user / system / tool）
	Role string `json:"role,omitempty"`
	// Delta 是否为增量片段。
	// true：Content / Reasoning 为增量，前端追加；
	// false：Content / Reasoning 为全量快照，前端整体覆盖。
	Delta bool `json:"delta,omitempty"`
	// Content 文本内容，含义由 Delta 决定：true 为增量片段，false 为全量快照
	Content string `json:"content,omitempty"`
	// Reasoning 思考内容，含义由 Delta 决定：true 为增量片段，false 为全量快照
	Reasoning string `json:"reasoning,omitempty"`
	// ToolCalls 工具调用（含结果）
	ToolCalls []types.ToolCall `json:"toolCalls,omitempty"`
	// Media 媒体文件
	Media []types.Media `json:"media,omitempty"`
	// Approval 审批信息，仅 approval 事件携带
	Approval *ApprovalData `json:"approval,omitempty"`
	// Usage token 用量，仅 token_usage 事件携带。
	// 同一轮会多次推送（每次模型调用后更新），前端按 eventID 覆盖即可。
	Usage *types.TokenUsage `json:"usage,omitempty"`
	// Error 错误信息，仅 error 事件携带
	Error *ErrorBody `json:"error,omitempty"`
	// Meta 透传元数据
	Meta map[string]any `json:"meta,omitempty"`
	// Truncated 内容是否被截断，仅历史回放可能出现：
	// 回放为避免一次拉爆载荷，会对超长文本（思考、命令输出、工具结果）做截断。
	// 实时事件不会置位。
	Truncated bool `json:"truncated,omitempty"`
}

// ApprovalData 审批请求
type ApprovalData struct {
	// ID 审批标识，客户端回传时使用
	ID string `json:"id"`
	// Content 审批内容描述
	Content string `json:"content,omitempty"`
	// Extra 附加信息
	Extra map[string]any `json:"extra,omitempty"`
}

// ErrorBody 统一错误结构
type ErrorBody struct {
	// Code 机器可读的错误码
	Code string `json:"code"`
	// Message 人类可读的错误描述
	Message string `json:"message"`
	// Detail 补充细节
	Detail string `json:"detail,omitempty"`
}

// HelloData 连接建立后下发的握手信息
type HelloData struct {
	ClientID string `json:"clientID"`
	UserID   string `json:"userID"` // 当前连接的用户身份，会话键前缀为 ws_<userID>_
	// ChatID 当前对话标识。由身份解析函数返回或服务端生成，
	// 客户端后续不传 chatID 时消息都会落在这个对话里。
	ChatID       string      `json:"chatID"`
	Protocol     int         `json:"protocol"`
	Agents       []AgentInfo `json:"agents,omitempty"`
	DefaultAgent string      `json:"defaultAgent,omitempty"`
	Heartbeat    int         `json:"heartbeat,omitempty"` // 心跳间隔（秒）
}

// AgentInfo agent 基本信息
type AgentInfo struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// AckData 上行请求受理确认
type AckData struct {
	OK        bool   `json:"ok"`
	SessionID string `json:"sessionID,omitempty"`
	// ChatID 服务端实际使用的对话标识：请求未携带时由服务端生成，前端应保存以便续聊
	ChatID  string `json:"chatID,omitempty"`
	ReqID   string `json:"reqID,omitempty"`
	Message string `json:"message,omitempty"`
}

// PongData 应用层心跳应答
type PongData struct {
	ClientTS int64 `json:"clientTS,omitempty"`
	ServerTS int64 `json:"serverTS"`
}

// FileInfo 文件元信息：上传接口的响应体，也是 chat.send 里 fileIDs 指向的对象。
//
// URL 是相对地址（形如 /ws/files/<id>），前端拼上自己的 host 与 token 即可访问：
// 加 ?download=1 触发另存，不加则按 MIME 内联展示（图片、文本、PDF 可直接渲染）。
type FileInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Size int64  `json:"size"`
	// Mime 由文件名后缀推断，取不到时为空，下载时会再嗅探一次
	Mime string `json:"mime,omitempty"`
	// URL 相对访问地址
	URL string `json:"url"`
	// ExpiresAt 毫秒时间戳，过期后 fileID 失效
	ExpiresAt int64 `json:"expiresAt,omitempty"`
}

// FileListData 文件列表（GET /ws/files 的响应）
type FileListData struct {
	Items []FileInfo `json:"items"`
}

// AgentListData agent 列表
type AgentListData struct {
	Agents       []AgentInfo `json:"agents"`
	DefaultAgent string      `json:"defaultAgent,omitempty"`
}

// ChatSendData 上行发送消息
type ChatSendData struct {
	// SessionID 可选，重连后继续同一会话时回传
	SessionID string `json:"sessionID,omitempty"`
	// ChatID 可选，对话标识，为空则服务端生成
	ChatID string `json:"chatID,omitempty"`
	// Agent 可选，指定处理该消息的 agent，为空使用默认
	Agent string `json:"agent,omitempty"`
	// Content 文本内容
	Content string `json:"content,omitempty"`
	// Media 媒体文件。小图可以直接内联 base64，但单帧上限 1MB（见 MaxMessageSize），
	// 稍大的文件请先走文件上传接口拿 fileID，再用 FileIDs 引用。
	Media []types.Media `json:"media,omitempty"`
	// FileIDs 已上传文件的标识，按顺序追加到 Media 之后。
	// 服务端会取出真实路径与文件名，拼成 Media 再投递，因此帧里不需要带文件内容。
	FileIDs []string `json:"fileIDs,omitempty"`
	// Metadata 元数据
	Metadata map[string]any `json:"metadata,omitempty"`
}

// ChatCancelData 上行中断会话
type ChatCancelData struct {
	SessionID string `json:"sessionID,omitempty"`
	ChatID    string `json:"chatID,omitempty"`
}

// PingData 上行心跳
type PingData struct {
	TS int64 `json:"ts,omitempty"`
}

// ShutdownData 服务端关闭通知
type ShutdownData struct {
	Reason string `json:"reason,omitempty"`
}

// ---- 历史消息 ----

// SessionListData 上行：拉取对话列表。
// 列表以 thread 为粒度——一个 thread 是一次完整对话，其下挂多轮 turn；
// 同一会话换过 agent 会产生多个 thread，显示为多个对话。
type SessionListData struct {
	// Limit 返回条数：0 使用默认值 50，负数表示取全量（上限 200）
	Limit int `json:"limit,omitempty"`
	// BeforeLastUpdate 翻页游标：只取最后更新时间早于该时刻（毫秒）的对话
	BeforeLastUpdate int64 `json:"beforeLastUpdate,omitempty"`
	// BeforeThreadID 与 BeforeLastUpdate 组成复合游标，时间相同时用它续接
	BeforeThreadID string `json:"beforeThreadID,omitempty"`
}

// SessionListItem 对话列表项，对应一个 thread
type SessionListItem struct {
	// ThreadID 线程标识，拉历史时作为 session.history 的 threadID
	ThreadID string `json:"threadID"`
	// SessionID 所属会话，继续对话时作为 chat.send 的 sessionID
	SessionID string `json:"sessionID"`
	// Agent 该对话使用的 agent
	Agent string `json:"agent,omitempty"`
	// Title 对话标题，取首轮提问（截断）；仅含媒体时为空
	Title string `json:"title,omitempty"`
	// Preview 最后一条回复（截断），列表页做副标题
	Preview string `json:"preview,omitempty"`
	// Turns 轮次数量
	Turns int64 `json:"turns"`
	// Completed 末轮是否正常结束，false 表示被中断或仍在生成
	Completed bool `json:"completed"`
	// CreatedAt / LastUpdate 毫秒时间戳
	CreatedAt  int64 `json:"createdAt"`
	LastUpdate int64 `json:"lastUpdate"`
	// Usage 该对话的累计用量，取 Usage.Total（Last 为空），用于列表直接展示消耗
	Usage *types.TokenUsage `json:"usage,omitempty"`
}

// SessionListResult 对话列表响应，按 lastUpdate 倒序
type SessionListResult struct {
	Items []SessionListItem `json:"items"`
	// HasMore 是否还有更早的对话
	HasMore bool `json:"hasMore,omitempty"`
	// NextBeforeLastUpdate / NextBeforeThreadID 下一页游标，回传即可续拉
	NextBeforeLastUpdate int64  `json:"nextBeforeLastUpdate,omitempty"`
	NextBeforeThreadID   string `json:"nextBeforeThreadID,omitempty"`
}

// SessionHistoryData 上行：拉取会话历史。
// 层级为 session -> thread -> turn：不指定 threadID 时返回该会话下所有线程及其轮次。
type SessionHistoryData struct {
	// SessionID 会话标识，与 ChatID 二选一
	SessionID string `json:"sessionID,omitempty"`
	// ChatID 对话标识，用于推导出 sessionID
	ChatID string `json:"chatID,omitempty"`
	// ThreadID 指定线程，不传则返回该会话下所有线程
	ThreadID string `json:"threadID,omitempty"`
	// BeforeSeq 翻页游标：只取 seq 小于该值的轮次（加载更早的消息）
	BeforeSeq uint `json:"beforeSeq,omitempty"`
	// Since 增量补发：毫秒时间戳，只取创建时间不早于该时刻的轮次
	Since int64 `json:"since,omitempty"`
	// Limit 每线程返回条数：0 使用默认值 200，负数表示取全量（上限 1000）
	Limit int `json:"limit,omitempty"`
	// IncludeFrames 是否附带逐帧执行过程（思考、工具调用、审批、错误）。
	// 附带时结果里的 frames 就是实时流的同构复刻，前端可直接复用同一套渲染管线。
	IncludeFrames bool `json:"includeFrames,omitempty"`
}

// HistoryTurn 历史轮次，Seq 可直接作为下一次翻页的 beforeSeq。
//
// Question / Answer 是轮次摘要，便于列表页直接渲染；
// 需要中间过程时看 SessionHistoryResult.Frames，两处内容会有重叠，按需取其一。
type HistoryTurn struct {
	Seq         uint   `json:"seq"`
	ReqID       string `json:"reqID"`
	ThreadID    string `json:"threadID,omitempty"`
	TurnID      string `json:"turnID,omitempty"`
	Agent       string `json:"agent,omitempty"`
	Question    string `json:"question,omitempty"`
	Answer      string `json:"answer,omitempty"`
	Completed   bool   `json:"completed"`
	CreatedAt   int64  `json:"createdAt"`
	CompletedAt int64  `json:"completedAt,omitempty"`
	// Usage 本轮 token 用量：只有 Last（本轮）与 ModelContextWindow 有值，
	// Total 留空，对话累计看 HistoryThread.Usage。老数据没有记录时为 nil。
	Usage *types.TokenUsage `json:"usage,omitempty"`
}

// HistoryThread 一个线程及其轮次
type HistoryThread struct {
	ThreadID   string        `json:"threadID"`
	Agent      string        `json:"agent,omitempty"`
	CreatedAt  int64         `json:"createdAt,omitempty"`
	LastUpdate int64         `json:"lastUpdate,omitempty"`
	Turns      []HistoryTurn `json:"turns"`
	HasMore    bool          `json:"hasMore,omitempty"`
	// Usage 该对话的累计用量，取 Usage.Total；Last 无意义（为空）。
	Usage *types.TokenUsage `json:"usage,omitempty"`
}

// SessionHistoryResult 历史消息响应，threads 按创建时间升序，turns 按 seq 升序。
//
// Frames 是按时间升序铺平的 agent 信封，与实时推送结构完全一致：
// 前端可以直接把它们逐条喂给同一个消息处理器，不需要为历史单独写分支。
// 每条帧的 seq 即事件序号，可作为增量拉取的游标。
type SessionHistoryResult struct {
	SessionID string          `json:"sessionID"`
	Threads   []HistoryThread `json:"threads"`
	// Frames 仅在 includeFrames=true 时返回
	Frames []*Envelope `json:"frames,omitempty"`
	// FramesTruncated 是否有轮次的过程因条数或体积上限被截断
	FramesTruncated bool `json:"framesTruncated,omitempty"`
}

// TurnReplayData 上行：回放单轮完整执行过程。
// 用于历史列表先渲染摘要、用户点开某一轮时再拉完整过程（含未截断的工具输出）。
type TurnReplayData struct {
	// SessionID 会话标识，与 ChatID 二选一
	SessionID string `json:"sessionID,omitempty"`
	// ChatID 对话标识，用于推导出 sessionID
	ChatID string `json:"chatID,omitempty"`
	// TurnID 轮次标识，必填
	TurnID string `json:"turnID"`
	// AfterSeq 翻页游标：只取 seq 大于该值的事件
	AfterSeq uint `json:"afterSeq,omitempty"`
	// Limit 返回帧数：0 使用默认值 500，负数表示取全量（上限 20000）
	Limit int `json:"limit,omitempty"`
}

// TurnReplayResult 单轮回放响应
type TurnReplayResult struct {
	SessionID string `json:"sessionID"`
	TurnID    string `json:"turnID"`
	// Frames 按时间升序的 agent 信封，与实时流同构
	Frames []*Envelope `json:"frames"`
	// HasMore 是否还有更晚的帧，续拉时把 nextAfterSeq 作为 afterSeq
	HasMore bool `json:"hasMore,omitempty"`
	// NextAfterSeq 下一次翻页的游标，等于最后一帧的 seq
	NextAfterSeq uint `json:"nextAfterSeq,omitempty"`
	// Truncated 是否有内容因体积上限被截断
	Truncated bool `json:"truncated,omitempty"`
}

// ---- 界面操作（ui 域） ----
//
// 服务端 ↔ 浏览器之间的反向 RPC：agent 的界面工具通过这条通道，
// 向「用户当前打开的那个页面」取快照、下发命令。
//
// Snapshot / Result / Params 对服务端是**透传**的 json.RawMessage：服务端不理解业务结构，
// 只负责按会话投递、关联回执与超时。结构语义分别由两端各自负责——
// 生产者是前端（快照由页面注册表生成），消费者是模型（读工具描述与返回的 JSON）。
//
// 这样后端不需要为每个业务页面改代码：页面结构变化只影响前端与工具描述。

// UIRequestData 服务端下发的界面操作请求（载荷在 Envelope.Data 里）
type UIRequestData struct {
	// Op 操作类型：observe 取快照 | act 执行一条界面命令 | navigate 打开/切换页面
	Op string `json:"op"`
	// Cmd act 要执行的命令 ID，取值必须来自最近一次快照的 actions[].id
	Cmd string `json:"cmd,omitempty"`
	// View navigate 的目标页面标识（菜单的 View 字段）
	View string `json:"view,omitempty"`
	// Query navigate 的检索词：只给 query 表示「搜菜单」，不导航
	Query string `json:"query,omitempty"`
	// Params act 的命令参数 / navigate 传给目标页面的参数，
	// 结构由页面自己的命令或页面入参决定
	Params json.RawMessage `json:"params,omitempty"`
	// Rev 期望的页面版本，来自最近一次快照；不匹配由前端拒绝并要求重新观察
	// （navigate 不使用：跨页面的版本没有意义）
	Rev int64 `json:"rev,omitempty"`
}

// UIResponseData 浏览器回传的操作结果（上行 kind=client / type=ui.response）
type UIResponseData struct {
	// OK 是否执行成功
	OK bool `json:"ok"`
	// Error 失败原因，OK=false 时必须有值，会原样交给模型
	Error string `json:"error,omitempty"`
	// Rev 执行后页面的版本号，供下一轮命令引用
	Rev int64 `json:"rev,omitempty"`
	// Snapshot 页面快照：observe / navigate 的返回，或 act 里声明 returns=full 的命令
	Snapshot json.RawMessage `json:"snapshot,omitempty"`
	// Result 命令自身的执行结果
	Result json.RawMessage `json:"result,omitempty"`
	// Patch 快照补丁：act 的默认回执（写命令只回变化的那几段，模型自己合进上一份快照）
	Patch json.RawMessage `json:"patch,omitempty"`
}
