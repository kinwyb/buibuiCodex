package channel

import (
	"context"

	"github.com/kinwyb/buibuiCodex/core/bus"
	"github.com/kinwyb/buibuiCodex/core/config"
)

// Factory Channel 工厂接口
// 每种 Channel 类型实现此接口，在 init() 中注册到全局注册表
type Factory interface {
	// Type 返回 Channel 类型标识 (如 "wxcom", "telegram")
	Type() string

	// CreateFromConfig 从 ChannelsConfig 创建 Channel 实例
	// 工厂从 cfg 中提取自己需要的字段进行解析
	// common: 公共参数（MessageBus 等）
	// 返回: Channel 实例列表（支持多账号），未启用时返回 nil
	CreateFromConfig(ctx context.Context, cfg *config.ChannelsConfig, common *CommonParams) ([]Channel, error)
}

// CommonParams 创建 Channel 的公共参数
type CommonParams struct {
	Bus       *bus.MessageBus
	Canceller Canceller
}

// Canceller 用于外部调用 agent session 取消
type Canceller interface {
	Cancel(sessionKey string)
}
