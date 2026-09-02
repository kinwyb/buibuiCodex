package channel

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/kinwyb/buibuiCodex/core/bus"
	"github.com/kinwyb/buibuiCodex/core/config"
	"github.com/kinwyb/buibuiCodex/core/types"
)

// Manager 通道管理器
type Manager struct {
	channels map[string]Channel // type -> Channel
	bindings *ThreadBindingService
	bus      *bus.MessageBus
	canceler Canceller // session 取消器
	// 订阅事件信息
	outSub *bus.EventSubscription

	mu     sync.RWMutex
	ctx    context.Context
	cancel context.CancelFunc
}

// NewManager 创建通道管理器
func NewManager(msgBus *bus.MessageBus) *Manager {
	ctx, cancel := context.WithCancel(context.Background())
	return &Manager{
		channels: make(map[string]Channel),
		bindings: NewThreadBindingService(),
		bus:      msgBus,
		ctx:      ctx,
		cancel:   cancel,
	}
}

// Register 注册通道
func (m *Manager) Register(ch Channel) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	channelName := ch.Name()
	if _, ok := m.channels[channelName]; ok {
		return fmt.Errorf("channel %s already registered", channelName)
	}

	m.channels[channelName] = ch
	m.bus.SubscribeSyncHandler(string(channelName), ch.HandleRequest)
	return nil
}

// RegisterWithType 使用指定类型注册通道（支持多账号同类型）
func (m *Manager) RegisterWithType(ch Channel, name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.channels[name]; ok {
		return fmt.Errorf("channel %s already registered", name)
	}

	m.channels[name] = ch
	m.bus.SubscribeSyncHandler(string(name), ch.HandleRequest)
	return nil
}

// Unregister 取消注册通道
func (m *Manager) Unregister(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	ch, ok := m.channels[name]
	if !ok {
		return fmt.Errorf("channel %s not found", name)
	}

	// 停止通道
	if ch.IsRunning() {
		ch.Stop()
	}

	m.bus.UnsubscribeSyncHandler(name)

	delete(m.channels, name)
	return nil
}

// Get 获取通道
func (m *Manager) Get(name string) (Channel, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	ch, ok := m.channels[name]
	return ch, ok
}

// List 列出所有通道名称
func (m *Manager) List() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	names := make([]string, 0, len(m.channels))
	for name := range m.channels {
		names = append(names, string(name))
	}
	return names
}

// StartAll 启动所有通道
func (m *Manager) StartAll(ctx context.Context) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, ch := range m.channels {
		if err := ch.Start(ctx); err != nil {
			slog.Error("failed to start channel", "channel", ch.Name(), "err", err)
			return fmt.Errorf("failed to start channel %s: %w", ch.Name(), err)
		}
	}

	// 启动消息分发
	m.startDispatchers(ctx)

	return nil
}

// StopAll 停止所有通道
func (m *Manager) StopAll() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 取消上下文，停止分发器
	m.cancel()

	// 取消订阅
	if m.outSub != nil {
		m.outSub.Close()
		m.outSub = nil
	}

	// 停止所有通道
	for name, ch := range m.channels {
		if ch.IsRunning() {
			if err := ch.Stop(); err != nil {
				slog.Error("failed to stop channel", "channel", ch.Name(), "err", err)
				return fmt.Errorf("failed to stop channel %s: %w", name, err)
			}
		}
	}

	return nil
}

// startDispatchers 启动消息分发器
func (m *Manager) startDispatchers(ctx context.Context) error {
	// 订阅出站消息（订阅所有）
	outSub, err := m.bus.SubscribeEvent()
	if err != nil {
		return err
	}
	m.outSub = outSub

	// 启动分发 goroutine
	go m.dispatchOutbound(ctx)
	return nil
}

// dispatchOutbound 分发出站消息
func (m *Manager) dispatchOutbound(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("Recovered in dispatchOutbound", "error", r)
			go m.dispatchOutbound(ctx)
		}
	}()
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-m.outSub.Channel:
			if !ok {
				return
			}
			// slog.Info("dispatch outbound", "msg.id", msg.ID, "msg.seq", msg.ChunkIndex)

			// 使用 ThreadBinding 解析目标通道
			targetChan := m.bindings.ResolveChannel(msg)

			// 获取对应通道
			ch, ok := m.Get(targetChan)
			if !ok {
				// 通道不存在，跳过
				continue
			}
			switch msg.Type {
			case types.EventReasoningDelta,
				types.EventReasoningCompleted,
				types.EventMessageDelta,
				types.EventMessageCompleted,
				types.EventFinished:
				if err := ch.Send(ctx, msg); err != nil {
					slog.Error("failed to send message to channel", "channel", ch.Name(), "err", err)
					// 记录错误，但不中断分发
					continue
				}
			default:
				if err := ch.HandleEvent(ctx, msg); err != nil {
					slog.Error("failed to handle event", "type", msg.Type, "err", err)
					// 记录错误，但不中断分发
					continue
				}
			}
		}
	}
}

// GetBindingService 获取 ThreadBinding 服务
func (m *Manager) GetBindingService() *ThreadBindingService {
	return m.bindings
}

// BindSession 绑定会话到目标通道
func (m *Manager) BindSession(sessionKey, targetChan string, opts ...BindingOption) error {
	return m.bindings.Bind(sessionKey, targetChan, opts...)
}

// UnbindSession 解除会话绑定
func (m *Manager) UnbindSession(sessionKey string) error {
	return m.bindings.Unbind(sessionKey)
}

// GetBus 获取消息总线
func (m *Manager) GetBus() *bus.MessageBus {
	return m.bus
}

// ChannelCount 获取通道数量
func (m *Manager) ChannelCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.channels)
}

// SetCanceller 设置 session 取消器
func (m *Manager) SetCanceller(c Canceller) {
	m.canceler = c
}

// InitializeFromConfig 从配置自动初始化所有 Channel
// 每个工厂从 ChannelsConfig 中提取自己需要的配置字段
func (m *Manager) InitializeFromConfig(ctx context.Context, cfg *config.ChannelsConfig) error {
	if cfg == nil {
		return nil
	}

	common := &CommonParams{Bus: m.bus, Canceller: m.canceler}

	// 遍历所有已注册的工厂
	for _, typeKey := range ListFactories() {
		factory, ok := GetFactory(typeKey)
		if !ok {
			continue
		}

		// 工厂自行判断是否启用并提取配置
		channels, err := factory.CreateFromConfig(ctx, cfg, common)
		if err != nil {
			return fmt.Errorf("failed to create channel '%s': %w", typeKey, err)
		}

		// 注册所有实例
		for _, ch := range channels {
			if err := m.Register(ch); err != nil {
				return err
			}
		}
	}

	// 处理 ThreadBindings
	for _, binding := range cfg.ThreadBindings {
		if binding.TargetAgent != "" {
			m.BindSession(binding.SessionKey, binding.TargetChannel, WithAgent(binding.TargetAgent), WithPriority(binding.Priority))
		} else {
			m.BindSession(binding.SessionKey, binding.TargetChannel, WithPriority(binding.Priority))
		}
	}

	return nil
}
