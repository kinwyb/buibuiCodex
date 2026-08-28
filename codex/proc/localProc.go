package proc

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"syscall"
	"time"
)

// LocalProcessConfig 控制 app-server 子进程启动参数
type LocalProcessConfig struct {
	BinPath    string            // codex 可执行文件路径，默认 "codex"
	ListenAddr string            // 监听地址，例如 "127.0.0.1:4500"
	ConfigPath string            // 可选：指定配置文件路径 (--config)
	Env        map[string]string // 环境变量注入
	Stdout     io.Writer         // 日志重定向（默认 os.Stdout）
	Stderr     io.Writer         // 日志重定向（默认 os.Stderr）
}

type localProcessManager struct {
	cmd    *exec.Cmd
	cfg    LocalProcessConfig
	cancel context.CancelFunc
}

func (l *localProcessManager) Start(ctx context.Context) (*ProcessResult, error) {
	procCtx, cancel := context.WithCancel(context.Background())
	l.cancel = cancel

	// 1. 构造启动命令行参数
	args := []string{"app-server", "--listen", l.cfg.ListenAddr, "-c", `sandbox_mode="danger-full-access"`}
	if l.cfg.ConfigPath != "" {
		args = append(args, "--config", l.cfg.ConfigPath)
	}

	l.cmd = exec.CommandContext(procCtx, l.cfg.BinPath, args...)
	l.cmd.Stdout = l.cfg.Stdout
	l.cmd.Stderr = l.cfg.Stderr
	// 当 Context 被 cancel 时，自动向整个进程组发送 SIGTERM
	l.cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	l.cmd.Cancel = func() error {
		pgid, err := syscall.Getpgid(l.cmd.Process.Pid)
		if err == nil {
			return syscall.Kill(-pgid, syscall.SIGTERM)
		}
		return l.cmd.Process.Signal(syscall.SIGTERM)
	}

	// 2. 注入环境变量
	if len(l.cfg.Env) > 0 {
		env := os.Environ()
		for k, v := range l.cfg.Env {
			env = append(env, fmt.Sprintf("%s=%s", k, v))
		}
		l.cmd.Env = env
	}

	// 3. 异步启动子进程
	log.Printf("[AppServerManager] 正在启动 app-server: %s %v", l.cfg.BinPath, args)
	if err := l.cmd.Start(); err != nil {
		return nil, fmt.Errorf("start app-server process failed: %w", err)
	}

	// 监控进程异常退出
	go func() {
		err := l.cmd.Wait()
		if err != nil && procCtx.Err() == nil {
			log.Printf("[AppServerManager] ERROR: app-server 进程意外退出: %v", err)
		}
	}()

	// 4. 健康检查：轮询等待 TCP 端口就绪
	if err := l.waitForPortReady(ctx, l.cfg.ListenAddr); err != nil {
		_ = l.Stop()
		return nil, fmt.Errorf("app-server start failed (port not ready): %w", err)
	}

	log.Printf("[AppServerManager] app-server 启动成功并就绪于: %s", l.cfg.ListenAddr)
	return &ProcessResult{
		WsUrl: l.cfg.ListenAddr,
	}, nil
}

// NewLocalProcessManager 初始化进程管理器
func NewLocalProcessManager(cfg LocalProcessConfig) (Process, error) {
	if cfg.BinPath == "" {
		cfg.BinPath = "codex"
	}
	if cfg.ListenAddr == "" {
		cfg.ListenAddr = "127.0.0.1:4500"
	}
	if cfg.Stdout == nil {
		cfg.Stdout = os.Stdout
	}
	if cfg.Stderr == nil {
		cfg.Stderr = os.Stderr
	}
	cfg.ListenAddr = fmt.Sprintf("ws://%s", cfg.ListenAddr)
	return &localProcessManager{
		cfg: cfg,
	}, nil
}

// Stop 优雅关闭 app-server 子进程
func (l *localProcessManager) Stop() error {
	if l.cancel != nil {
		l.cancel() // 触发 Context 终止进程
	}
	if l.cmd != nil && l.cmd.Process != nil {
		// 1. 先尝试平滑关闭 (SIGTERM)
		pgid, err := syscall.Getpgid(l.cmd.Process.Pid)
		if err == nil {
			_ = syscall.Kill(-pgid, syscall.SIGTERM)
		} else {
			_ = l.cmd.Process.Signal(syscall.SIGTERM)
		}

		// 2. 等待进程优雅退出（超时限制 2 秒）
		done := make(chan error, 1)
		go func() {
			done <- l.cmd.Wait()
		}()

		select {
		case <-done:
			// 优雅退出成功，Socket 干净释放
			return nil
		case <-time.After(2 * time.Second):
			// 3. 超时依然未退出，强制 Kill 进程组
			if err == nil {
				_ = syscall.Kill(-pgid, syscall.SIGKILL)
			} else {
				_ = l.cmd.Process.Kill()
			}
			return fmt.Errorf("app-server 响应 SIGTERM 超时，已强制 Kill")
		}
	}
	return nil
}

// 内部方法：探测 TCP 端口是否开启
func (l *localProcessManager) waitForPortReady(ctx context.Context, addr string) error {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			conn, err := net.DialTimeout("tcp", addr, 200*time.Millisecond)
			if err == nil {
				_ = conn.Close()
				return nil
			}
		}
	}
}
