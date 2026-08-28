package channel

import (
	"context"
	"sync"

	"github.com/kinwyb/buibuiCodex/core/bus"
	"github.com/kinwyb/buibuiCodex/core/types"
)

const (
	WxComChannel string = "wxcom" //企业微信通道
)

// Channel 通道接口
type Channel interface {
	// Name 返回通道名称标识 (如 "wxcom", "telegram")
	Name() string

	// AccountID 返回账号ID（多账号场景）
	AccountID() string

	// Start 启动通道
	Start(ctx context.Context) error

	// Stop 停止通道
	Stop() error

	// IsRunning 是否运行中
	IsRunning() bool

	// Send 发送消息
	Send(ctx context.Context, event *types.Event) error

	// HandleEvent 处理聊天事件 (start, tool, complete, error, interrupt 等)
	HandleEvent(ctx context.Context, event *types.Event) error

	// IsAllowed 检查发送者是否允许
	IsAllowed(senderID string) bool

	// HandleRequest 处理同步消息请求,tp消息类型
	// 返回响应消息，nil 表示不处理该请求类型
	HandleRequest(ctx context.Context, tp types.SyncMessageType, event *types.Event) (*types.InputMessage, error)
}

// BaseChannelConfig 通道基础配置
type BaseChannelConfig struct {
	Enabled    bool     `mapstructure:"enabled" json:"enabled"`
	AccountID  string   `mapstructure:"account_id" json:"account_id"` // 账号ID
	Name       string   `mapstructure:"name" json:"name"`             // 账号显示名称
	AllowedIDs []string `mapstructure:"allowed_ids" json:"allowed_ids"`
}

// BaseChannel 通道基础实现（嵌入模式）
type BaseChannel struct {
	name         string
	accountID    string
	config       BaseChannelConfig
	bus          *bus.MessageBus
	running      bool
	stopChan     chan struct{}
	mu           sync.RWMutex
	defaultAgent string // 默认处理的 agent 名称（为空则使用全局默认）
}

// NewChannelBase 创建通道基础实现
func NewChannelBase(name string, accountID string, config BaseChannelConfig, bus *bus.MessageBus, defaultAgent string) *BaseChannel {
	return &BaseChannel{
		name:         name,
		accountID:    accountID,
		config:       config,
		bus:          bus,
		running:      false,
		stopChan:     make(chan struct{}),
		defaultAgent: defaultAgent,
	}
}

// Name 返回通道类型
func (c *BaseChannel) Name() string {
	return c.name
}

// AccountID 返回通道账号ID
func (c *BaseChannel) AccountID() string {
	return c.accountID
}

// Start 启动通道
func (c *BaseChannel) Start(ctx context.Context) error {
	if !c.config.Enabled {
		return nil
	}

	c.mu.Lock()
	c.running = true
	c.mu.Unlock()
	return nil
}

// Stop 停止通道
func (c *BaseChannel) Stop() error {
	c.mu.Lock()
	if !c.running {
		c.mu.Unlock()
		return nil
	}
	c.running = false
	c.mu.Unlock()

	close(c.stopChan)
	return nil
}

// IsRunning 检查是否运行中
func (c *BaseChannel) IsRunning() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.running
}

// IsAllowed 检查发送者是否允许
func (c *BaseChannel) IsAllowed(senderID string) bool {
	if !c.config.Enabled {
		return false
	}

	// 如果没有限制列表，允许所有
	if len(c.config.AllowedIDs) == 0 {
		return true
	}

	// 检查是否在允许列表中
	for _, id := range c.config.AllowedIDs {
		if id == senderID {
			return true
		}
	}

	return false
}

// PublishInbound 发布入站消息
func (c *BaseChannel) PublishInbound(ctx context.Context, msg *types.InputMessage) error {
	msg.Channel = c.Name()
	msg.AccountID = c.accountID
	if c.defaultAgent != "" {
		msg.AgentName = c.defaultAgent
	}
	return c.bus.PublishInput(ctx, msg)
}

// WaitForStop 等待停止信号
func (c *BaseChannel) WaitForStop() <-chan struct{} {
	return c.stopChan
}

// GetBus 获取消息总线
func (c *BaseChannel) GetBus() *bus.MessageBus {
	return c.bus
}

// GetConfig 获取配置
func (c *BaseChannel) GetConfig() BaseChannelConfig {
	return c.config
}

// HandleEvent 处理聊天事件 (默认空实现，具体 channel 可覆盖)
func (c *BaseChannel) HandleEvent(ctx context.Context, event *types.Event) error {
	// 默认实现不做任何处理，由具体 channel 实现
	return nil
}

// Send 发送消息 (默认空实现，具体 channel 必须覆盖)
func (c *BaseChannel) Send(ctx context.Context, event *types.Event) error {
	// 默认实现不做任何处理，由具体 channel 实现
	return nil
}

// HandleRequest 处理请求消息 (默认不处理)
func (c *BaseChannel) HandleRequest(ctx context.Context, tp types.SyncMessageType, event *types.Event) (*types.InputMessage, error) {
	// 默认实现不处理任何请求，由具体 channel 根据请求类型实现
	return nil, nil
}
