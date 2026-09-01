package pathmap

import (
	"path/filepath"
	"strings"
)

// PathMapper 处理容器路径与宿主机路径之间的双向转换。
// 在 Docker 运行场景下，应用内部使用的路径（容器路径）与 LLM/外部系统看到的路径（宿主机路径）不同，
// 需要通过 volume 挂载关系进行映射。
//
// 示例：Docker 挂载 -v /home/user/data:/app/data
//
//	hostPath = "/home/user/data"
//	containerPath = "/app/data"
//
// 非 Docker 场景下，两者相同，不配置即可透传。
type PathMapper struct {
	mappings []mapping
}

type mapping struct {
	host      string // 宿主机路径（已 Clean）
	container string // 容器路径（已 Clean）
}

// Config 路径映射配置
type Config struct {
	// Mappings 定义宿主机路径到容器路径的映射关系。
	// 在 Docker 场景下对应 -v hostPath:containerPath 的挂载关系。
	// 非 Docker 场景留空即可，所有路径直接透传。
	Mappings []PathMapping `json:"mappings,omitempty"`
}

// PathMapping 单条路径映射
type PathMapping struct {
	Host      string `json:"host"`      // 宿主机上的实际路径
	Container string `json:"container"` // 容器内的挂载路径
}

// New 创建 PathMapper。
// 如果 cfg 为 nil 或 Mappings 为空，则所有路径原样透传（非 Docker 模式）。
func New(cfg *Config) *PathMapper {
	pm := &PathMapper{}
	if cfg == nil {
		return pm
	}
	for _, m := range cfg.Mappings {
		if m.Host == "" || m.Container == "" {
			continue
		}
		pm.mappings = append(pm.mappings, mapping{
			host:      filepath.Clean(m.Host),
			container: filepath.Clean(m.Container),
		})
	}
	return pm
}

// ToContainer 将宿主机路径转换为容器路径。
// 如果路径不匹配任何映射规则，原样返回。
func (pm *PathMapper) ToContainer(hostPath string) string {
	if len(pm.mappings) == 0 || strings.HasPrefix(hostPath, "http") {
		return hostPath
	}
	cleaned := filepath.Clean(hostPath)
	for _, m := range pm.mappings {
		if after, ok := strings.CutPrefix(cleaned, m.host); ok {
			rel := after
			return filepath.Join(m.container, rel)
		}
	}
	return hostPath
}

// ToHost 将容器路径转换为宿主机路径。
// 如果路径不匹配任何映射规则，原样返回。
func (pm *PathMapper) ToHost(containerPath string) string {
	if len(pm.mappings) == 0 || strings.HasPrefix(containerPath, "http") {
		return containerPath
	}
	cleaned := filepath.Clean(containerPath)
	for _, m := range pm.mappings {
		if after, ok := strings.CutPrefix(cleaned, m.container); ok {
			rel := after
			return filepath.Join(m.host, rel)
		}
	}
	return containerPath
}

// IsNoop 是否为空映射（非 Docker 模式）
func (pm *PathMapper) IsNoop() bool {
	return len(pm.mappings) == 0
}
