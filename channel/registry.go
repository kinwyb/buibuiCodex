package channel

import (
	"fmt"
	"sync"
)

// globalRegistry 全局 Channel 工厂注册表
var globalRegistry = &Registry{
	factories: make(map[string]Factory),
}

// Registry Channel 工厂注册表
type Registry struct {
	factories map[string]Factory
	mu        sync.RWMutex
}

// RegisterFactory 注册 Channel 工厂
// 在各 Channel 包的 init() 函数中调用
func RegisterFactory(factory Factory) {
	globalRegistry.register(factory)
}

func (r *Registry) register(factory Factory) {
	r.mu.Lock()
	defer r.mu.Unlock()

	typeKey := factory.Type()
	if _, exists := r.factories[typeKey]; exists {
		// 已存在则警告但不阻止注册（允许覆盖）
		fmt.Printf("warning: channel factory '%s' already registered, overwriting\n", typeKey)
	}

	r.factories[typeKey] = factory
}

// GetFactory 获取指定类型的工厂
func GetFactory(typeKey string) (Factory, bool) {
	return globalRegistry.get(typeKey)
}

func (r *Registry) get(typeKey string) (Factory, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	factory, ok := r.factories[typeKey]
	return factory, ok
}

// ListFactories 列出所有已注册的工厂类型
func ListFactories() []string {
	return globalRegistry.list()
}

func (r *Registry) list() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	types := make([]string, 0, len(r.factories))
	for t := range r.factories {
		types = append(types, t)
	}
	return types
}
