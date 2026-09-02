package docker

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/netip"
	"os"
	"path/filepath"
	"time"
	"uuid"

	"github.com/gorilla/websocket"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/mount"
	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/client"
)

type StartParam struct {
	ContainerName  string                     `description:"容器名称"`
	ImageName      string                     `description:"镜像名称"`
	WorkSpace      string                     `description:"工作目录"`
	TmpSpace       string                     `description:"临时文件目录"`
	CodexHome      string                     `description:"Codex根目录"`
	SkillDir       string                     `description:"技能目录"`
	APIBaseURL     string                     `description:"模型请求地址"`
	APIKey         string                     `description:"模型APIKey"`
	Env            []string                   `description:"环境变量"`
	ModelProviders map[string]ModelProvider   `description:"模型供应商"`
	MCP            map[string]MCPServerConfig `description:"MCP配置"`
}

func (s *StartParam) ToCodexConfig() CodexConfig {
	codexConfig := CodexConfig{
		OpenAIBaseURL:  s.APIBaseURL,
		SandboxMode:    "workspace-write",
		ModelProviders: s.ModelProviders,
		MCPServers:     s.MCP,
	}
	return codexConfig
}

type StartResult struct {
	ContainerID string `description:"容器ID"`
	ContainerIP string `description:"容器IP"`
	Token       string `description:"token"`
}

type SandboxManager struct {
	cli *client.Client
}

func NewDockerSandboxManager() (*SandboxManager, error) {
	// 从环境变量或系统默认 Socket 初始化 Docker 客户端
	cli, err := client.New(client.FromEnv, client.WithAPIVersionFromEnv())
	if err != nil {
		return nil, fmt.Errorf("初始化 Docker 客户端失败: %w", err)
	}
	return &SandboxManager{cli: cli}, nil
}

// StartCodexContainer 唤起（创建并启动）用户专属的 Codex 沙箱容器
func (m *SandboxManager) StartCodexContainer(ctx context.Context, opts StartParam) (*StartResult, error) {
	if opts.ContainerName == "" {
		opts.ContainerName = uuid.NewV4().String()
	}
	if opts.ImageName == "" {
		return nil, errors.New("ImageName is required")
	}

	containerName := fmt.Sprintf("codex_sandbox_%s", opts.ContainerName)
	imageName := opts.ImageName
	token := m.sha256(containerName)

	var containerID string
	// 1. 尝试查找是否已经存在同名容器（包含运行中和已停止的）
	existingContainer, err := m.findContainerByName(ctx, containerName)
	if err == nil && existingContainer != nil {
		slog.Info(fmt.Sprintf("发现已存在的容器 [%s] | ID: %s | 当前状态: %s", containerName, existingContainer.ID[:12], existingContainer.State))
		if existingContainer.State == container.StateRunning {
			containerID = existingContainer.ID
		} else {
			err = m.StopAndRemoveContainer(ctx, existingContainer.ID)
			if err != nil {
				return nil, fmt.Errorf("container name is used and stop container failed: %w", err)
			}
		}
	}

	if containerID == "" { // 创建并启动新容器
		// A. 容器基础配置
		config := &container.Config{
			Image: imageName,
			// 容器启动后运行 Codex app-server，并监听在 Unix Socket 上
			Cmd: []string{
				"codex", "app-server",
				"--listen", "ws://0.0.0.0:80",
				"--ws-auth", "capability-token",
				"--ws-token-sha256", token,
			},
			Env: opts.Env,
		}

		// B. 宿主机挂载配置
		hostConfig := &container.HostConfig{
			PortBindings: network.PortMap{
				network.MustParsePort("80/tcp"): []network.PortBinding{
					{
						HostIP:   netip.MustParseAddr("127.0.0.1"),
						HostPort: "", // 留空 = 动态随机分配宿主机端口
					},
				},
			},
			AutoRemove: false, // 退出后不立即删除，方便排查日志，由管理器统一清理
			Resources: container.Resources{
				Memory:   1024 * 1024 * 1024, // 限制最大使用 1GB 内存，防止撑爆宿主机
				NanoCPUs: 2000000000,         // 限制最大使用 2 核 CPU
			},
		}
		if opts.WorkSpace != "" {
			if err = os.MkdirAll(opts.WorkSpace, 0755); err != nil {
				return nil, fmt.Errorf("创建 Workspace 目录失败: %w", err)
			}
			// B. 挂载用户代码/文档工作区
			hostConfig.Mounts = append(hostConfig.Mounts, mount.Mount{
				Type:   mount.TypeBind,
				Source: opts.WorkSpace,
				Target: "/workspace",
			})
		}
		if opts.TmpSpace != "" {
			if err = os.MkdirAll(opts.TmpSpace, 0755); err != nil {
				return nil, fmt.Errorf("创建 Tmp 目录失败: %w", err)
			}
			// C. 挂载临时文件地址
			hostConfig.Mounts = append(hostConfig.Mounts, mount.Mount{
				Type:   mount.TypeBind,
				Source: opts.TmpSpace,
				Target: "/tmp",
			})
		}
		if opts.CodexHome != "" {
			if err = ensureCodexConfig(opts.CodexHome, opts.ToCodexConfig()); err != nil {
				slog.Error("codex config.toml set openai_base_url failed", "error", err.Error())
				return nil, fmt.Errorf("确保 Codex 配置失败: %w", err)
			}
			// codex home添加auth.json,没有这个文件后续即使配置了模型也会异常
			authFile := filepath.Join(opts.CodexHome, "auth.json")
			if _, err = os.Stat(authFile); os.IsNotExist(err) {
				os.WriteFile(authFile, []byte(fmt.Sprintf("{\n  \"OPENAI_API_KEY\": \"%s\"\n}", opts.APIKey)), os.ModePerm)
			}
			// C. 【关键】持久化 Codex 的历史与配置目录！
			hostConfig.Mounts = append(hostConfig.Mounts, mount.Mount{
				Type:   mount.TypeBind,
				Source: opts.CodexHome, // 宿主机保存该用户历史的目录
				Target: "/root/.codex",
			})
		}

		if opts.SkillDir != "" {
			// 加载技能目录
			dirInfo, err := os.Stat(opts.SkillDir)
			if err == nil || os.IsExist(err) {
				if dirInfo.IsDir() {
					// D. 挂载技能目录
					hostConfig.Mounts = append(hostConfig.Mounts, mount.Mount{
						Type:   mount.TypeBind,
						Source: opts.SkillDir,
						Target: "/root/.codex/skills",
					})
				}
			}
		}

		// 挂载工作区的技能目录
		workspaceSkillDir := filepath.Join(opts.WorkSpace, "skills")
		dirInfo, err := os.Stat(workspaceSkillDir)
		if err == nil || os.IsExist(err) {
			if dirInfo.IsDir() {
				// D. 挂载技能目录
				hostConfig.Mounts = append(hostConfig.Mounts, mount.Mount{
					Type:   mount.TypeBind,
					Source: workspaceSkillDir,
					Target: "/workspace/skills",
				})
			}
		}

		// C. 创建容器
		createOpt := client.ContainerCreateOptions{
			Config:           config,
			HostConfig:       hostConfig,
			NetworkingConfig: nil,
			Platform:         nil,
			Name:             containerName,
		}
		resp, err := m.cli.ContainerCreate(ctx, createOpt)
		if err != nil {
			return nil, fmt.Errorf("创建容器失败: %w", err)
		}

		// D. 启动容器
		_, err = m.cli.ContainerStart(ctx, resp.ID, client.ContainerStartOptions{})
		if err != nil {
			return nil, fmt.Errorf("启动容器失败: %w", err)
		}

		containerID = resp.ID
	}

	inspectResp, err := m.cli.ContainerInspect(ctx, containerID, client.ContainerInspectOptions{})
	if err != nil {
		return nil, fmt.Errorf("获取容器信息失败: %w", err)
	}

	// 3. 提取容器的 IP 地址 (默认 bridge 网络)
	containerPort := inspectResp.Container.NetworkSettings.Ports[network.MustParsePort("80/tcp")][0].HostPort

	if containerPort == "" {
		return nil, fmt.Errorf("未获取到容器开放端口，请检查 Docker 网络设置")
	}
	slog.Info(fmt.Sprintf("容器启动成功 | ID: %s | 名称: %s | Port: %s", containerID[:12], containerName, containerPort))
	containerIP := fmt.Sprintf("ws://127.0.0.1:%s", containerPort)
	// 检测ws通讯是否正常
	if err = m.waitForPortReady(ctx, containerIP); err != nil {
		// 移除容器
		_ = m.StopAndRemoveContainer(ctx, containerID)
		return nil, fmt.Errorf("容器服务网络通信超时")
	}
	return &StartResult{
		ContainerID: containerID[:12],
		Token:       containerName,
		ContainerIP: containerIP,
	}, nil
}

// 内部方法：探测 TCP 端口是否开启
func (m *SandboxManager) waitForPortReady(ctx context.Context, addr string) error {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			// 使用 websocket 建立实际长连接探测
			conn, resp, err := websocket.DefaultDialer.DialContext(ctx, addr, nil)
			if err == nil {
				_ = conn.Close()
				return nil // WebSocket 连接成功，服务完全就绪
			}
			// 如果没有拿到 err == nil，但返回了 401/403，说明 Codex 核心已就绪（仅鉴权拦截）
			if resp != nil {
				_ = resp.Body.Close()
				if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
					return nil
				}
			}
		}
	}
}

// findContainerByName 辅助函数：按精准名称查找容器
func (m *SandboxManager) findContainerByName(ctx context.Context, containerName string) (*container.Summary, error) {
	// Docker 内部容器名称前面通常带有斜杠 '/'，过滤时需匹配精确名称
	filterArgs := client.Filters{}
	filterArgs.Add("name", containerName) // 使用正则全匹配，避免部分匹配到名称相似的容器
	containers, err := m.cli.ContainerList(ctx, client.ContainerListOptions{
		All:     true, // 💡 关键：必须为 true，才能查到已停止（exited）的容器
		Filters: filterArgs,
	})
	if err != nil {
		return nil, err
	}

	if len(containers.Items) > 0 {
		return &containers.Items[0], nil
	}
	return nil, nil // 未找到
}

func (m *SandboxManager) sha256(content string) string {
	// 我们在这里生成一个新的哈希
	sha := sha256.New()
	// Write方法期望字节数据。如果您有一个字符串s，可以使用[]byte(s)将其强制转换为字节。
	sha.Write([]byte(content))
	// 这将最终的哈希结果作为字节切片获取。Sum的参数可用于追加到现有的字节切片：通常不需要这样做。
	bs := sha.Sum(nil)
	return fmt.Sprintf("%x", bs)
}

// StopAndRemoveContainer 结束（停止并删除）指定的 Codex 沙箱容器
func (m *SandboxManager) StopAndRemoveContainer(ctx context.Context, containerID string) error {
	// 超时时间：先给 3 秒进行 SIGTERM 优雅退出
	timeoutSec := 3

	// A. 停止容器
	_, err := m.cli.ContainerStop(ctx, containerID, client.ContainerStopOptions{Timeout: &timeoutSec})
	if err != nil {
		slog.Warn(fmt.Sprintf("停止容器警告 (可能已退出): %v", err))
	}

	// B. 强制删除容器及其挂载卷
	_, err = m.cli.ContainerRemove(ctx, containerID, client.ContainerRemoveOptions{
		Force:         true, // 如果还没死透，强行 kill
		RemoveVolumes: true, // 清理卷资源
	})
	if err != nil {
		return fmt.Errorf("删除容器失败: %w", err)
	}

	slog.Info(fmt.Sprintf("容器已成功销毁 | ID: %s", containerID[:12]))
	return nil
}
