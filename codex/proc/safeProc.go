package proc

import (
	"context"
	"log/slog"

	"github.com/kinwyb/buibuiCodex/codex/proc/docker"
)

type SafeProcConfig = docker.StartParam

type safeProc struct {
	dockerManager *docker.SandboxManager
	param         *SafeProcConfig
	containerInfo *docker.StartResult
}

// NewSafeProc 创建安全进程,docker方式运行
func NewSafeProc(param SafeProcConfig) (Process, error) {
	dockerManager, err := docker.NewDockerSandboxManager()
	if err != nil {
		return nil, err
	}
	return &safeProc{dockerManager: dockerManager, param: &param}, nil
}

func (s *safeProc) Start(ctx context.Context) (*ProcessResult, error) {
	var containerInfo *docker.StartResult
	var err error
	if containerInfo, err = s.dockerManager.StartCodexContainer(ctx, *s.param); err != nil {
		slog.Error("start docker container error", slog.String("error", err.Error()))
		return nil, err
	}
	s.containerInfo = containerInfo
	return &ProcessResult{
		WsUrl: containerInfo.ContainerIP,
		Token: containerInfo.Token,
	}, nil
}

func (s *safeProc) Stop() error {
	return s.dockerManager.StopAndRemoveContainer(context.Background(), s.containerInfo.ContainerID)
}
