package types

import (
	"context"
	"time"
	"uuid"
)

type State struct {
	ctx          context.Context    //上下文
	cancel       context.CancelFunc //取消函数
	EventHandler func(*Event)       //消息处理
	Input        *InputMessage      `json:"input"`
	SessionID    string             `json:"session_id"`  // 会话记录ID
	ReqID        string             `json:"req_id"`      // 输入消息ID
	NewMessage   []*Message         `json:"new_message"` // 新产生的消息
}

// NewState 创建一个消息状态
func NewState(input *InputMessage) *State {
	if input.ID == "" {
		input.ID = uuid.NewV4().String()
	}
	return &State{
		SessionID:    input.SessionID(),
		ReqID:        input.ID,
		Input:        input,
		NewMessage:   nil,
		EventHandler: nil,
	}
}

func (s *State) LastMessage() string {
	if len(s.NewMessage) == 0 {
		return ""
	}
	return s.NewMessage[len(s.NewMessage)-1].Content
}

// AddNewMessage 保存新信息
func (s *State) AddNewMessage(msg *Message) {
	s.NewMessage = append(s.NewMessage, msg)
}

func (s *State) BuildEvent() *Event {
	ret := &Event{
		SessionID: s.SessionID,
		ReqID:     s.ReqID,
		Timestamp: time.Now(),
	}
	if s.Input != nil {
		ret.Channel = s.Input.Channel
		ret.AccountID = s.Input.AccountID
		ret.SenderID = s.SessionID
		ret.ChatID = s.Input.ChatID
		ret.Metadata = s.Input.Metadata
	}
	return ret
}
