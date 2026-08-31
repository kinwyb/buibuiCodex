package types

import "time"

type SyncMessageType string

const (
	SyncMessageTypeSendFile SyncMessageType = "send_file" // 发送文件
)

type EventType string

const (
	EventReasoningStart     EventType = "reasoning_start"
	EventReasoningDelta     EventType = "reasoning_delta"
	EventReasoningCompleted EventType = "reasoning_completed"
	EventMessageStart       EventType = "message_start"
	EventMessageDelta       EventType = "message_delta"
	EventMessageCompleted   EventType = "message_completed"
	EventToolStart          EventType = "tool_start"
	EventToolCompleted      EventType = "tool_completed"
	EventError              EventType = "error"
	EventApproval           EventType = "approval"
)

type Event struct {
	EventID    string         `json:"eventID"`
	ChunkIndex int            `json:"chunkIndex"`
	SessionID  string         `json:"sessionID"`
	ReqID      string         `json:"reqID"`
	Type       EventType      `json:"type"`
	Channel    string         `json:"channel"`
	AccountID  string         `json:"accountID"`
	SenderID   string         `json:"senderID"`
	ChatID     string         `json:"chatID"`
	Message    *Message       `json:"message,omitempty"`
	Metadata   map[string]any `json:"metadata,omitempty"`
	Timestamp  time.Time      `json:"timestamp"`
}

// NewEvent creates a new event with current timestamp
func NewEvent(eventType EventType) *Event {
	return &Event{
		Type:      eventType,
		Timestamp: time.Now(),
	}
}

// WithMessage adds message to the event
func (e *Event) WithMessage(msg *Message) *Event {
	e.Message = msg
	return e
}

// WithMetadata adds metadata to the event
func (e *Event) WithMetadata(metadata map[string]any) *Event {
	e.Metadata = metadata
	return e
}
