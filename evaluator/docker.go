package evaluator

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"skill-eval/providers"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
)

type DockerExecutor struct {
	Image      string
	ProjectDir string
	once       sync.Once
	cli        *client.Client
	initErr    error
}

func NewDockerExecutor(cfg EvalConfig) *DockerExecutor {
	image := cfg.DockerImage
	if image == "" {
		image = "golang:1.25"
	}
	projectDir := cfg.ProjectRootDir
	if projectDir == "" {
		wd, _ := os.Getwd()
		projectDir = wd
	}
	return &DockerExecutor{
		Image:      image,
		ProjectDir: projectDir,
	}
}

func (d *DockerExecutor) RunCase(ctx context.Context, input string, maxRounds int) (providers.RunResult, string, string, string, error) {
	cli, err := d.getClient()
	if err != nil {
		return providers.RunResult{}, "", d.Image, "", err
	}
	if err := d.ensureImage(ctx, cli); err != nil {
		return providers.RunResult{}, "", d.Image, "", err
	}

	containerName := fmt.Sprintf("skill-eval-%d", time.Now().UnixNano())
	cfg := &container.Config{
		Image: d.Image,
		Env: []string{
			fmt.Sprintf("EVAL_INPUT=%s", input),
			fmt.Sprintf("EVAL_MAX_ROUNDS=%d", maxRounds),
		},
		WorkingDir: "/workspace",
		Cmd:        []string{"sh", "-lc", "go run . eval-case"},
	}
	if v := os.Getenv("OPENAI_API_KEY"); v != "" {
		cfg.Env = append(cfg.Env, fmt.Sprintf("OPENAI_API_KEY=%s", v))
	}
	if v := os.Getenv("GLM_API_KEY"); v != "" {
		cfg.Env = append(cfg.Env, fmt.Sprintf("GLM_API_KEY=%s", v))
	}

	hostCfg := &container.HostConfig{
		Binds: []string{
			fmt.Sprintf("%s:/workspace", filepath.Clean(d.ProjectDir)),
		},
	}

	resp, err := cli.ContainerCreate(ctx, cfg, hostCfg, nil, nil, containerName)
	if err != nil {
		return providers.RunResult{}, containerName, d.Image, "", fmt.Errorf("create container failed: %w", err)
	}
	defer func() {
		rmCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		_ = cli.ContainerRemove(rmCtx, resp.ID, container.RemoveOptions{Force: true, RemoveVolumes: true})
	}()

	if err := cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return providers.RunResult{}, containerName, d.Image, "", fmt.Errorf("start container failed: %w", err)
	}

	waitCh, errCh := cli.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
	var waitStatus int64
	select {
	case err := <-errCh:
		if err != nil {
			return providers.RunResult{}, containerName, d.Image, "", fmt.Errorf("wait container failed: %w", err)
		}
	case status := <-waitCh:
		waitStatus = status.StatusCode
	}

	raw, logErr := d.readContainerLogs(ctx, cli, resp.ID)
	if logErr != nil {
		raw = fmt.Sprintf("read logs failed: %v", logErr)
	}
	raw = strings.TrimSpace(raw)

	if waitStatus != 0 {
		return providers.RunResult{}, containerName, d.Image, raw, fmt.Errorf("container exited with code %d", waitStatus)
	}

	result, parseErr := parseRunResult(raw)
	if parseErr != nil {
		return providers.RunResult{}, containerName, d.Image, raw, parseErr
	}
	return result, containerName, d.Image, raw, nil
}

func (d *DockerExecutor) HealthCheck(ctx context.Context) DockerHealth {
	h := DockerHealth{
		Enabled:   true,
		Available: false,
		CheckedAt: time.Now().Format(time.RFC3339),
	}

	cli, err := d.getClient()
	if err != nil {
		h.Error = err.Error()
		return h
	}

	ping, err := cli.Ping(ctx)
	if err != nil {
		h.Error = err.Error()
		return h
	}

	serverVersion, err := cli.ServerVersion(ctx)
	if err != nil {
		h.Error = err.Error()
		return h
	}

	fillHealthFromPing(&h, ping)
	h.Available = true
	h.ServerVersion = serverVersion.Version
	h.OperatingSystem = serverVersion.Os
	return h
}

func fillHealthFromPing(h *DockerHealth, ping types.Ping) {
	h.APIVersion = ping.APIVersion
}

func (d *DockerExecutor) getClient() (*client.Client, error) {
	d.once.Do(func() {
		d.cli, d.initErr = client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	})
	if d.initErr != nil {
		return nil, fmt.Errorf("init docker client failed: %w", d.initErr)
	}
	return d.cli, nil
}

func (d *DockerExecutor) ensureImage(ctx context.Context, cli *client.Client) error {
	_, _, err := cli.ImageInspectWithRaw(ctx, d.Image)
	if err == nil {
		return nil
	}

	reader, pullErr := cli.ImagePull(ctx, d.Image, image.PullOptions{})
	if pullErr != nil {
		return fmt.Errorf("pull image %s failed: %w", d.Image, pullErr)
	}
	defer reader.Close()
	_, _ = io.Copy(io.Discard, reader)
	return nil
}

func (d *DockerExecutor) readContainerLogs(ctx context.Context, cli *client.Client, containerID string) (string, error) {
	reader, err := cli.ContainerLogs(ctx, containerID, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
	})
	if err != nil {
		return "", err
	}
	defer reader.Close()

	var stdout strings.Builder
	var stderr strings.Builder
	if _, err := stdcopy.StdCopy(&stdout, &stderr, reader); err != nil {
		return "", err
	}
	out := strings.TrimSpace(stdout.String())
	errOut := strings.TrimSpace(stderr.String())
	if errOut == "" {
		return out, nil
	}
	if out == "" {
		return errOut, nil
	}
	return out + "\n[stderr]\n" + errOut, nil
}

func parseRunResult(raw string) (providers.RunResult, error) {
	var result providers.RunResult
	if err := json.Unmarshal([]byte(raw), &result); err == nil {
		return result, nil
	}

	// 兜底: 一些运行时日志可能夹杂在 stdout，尝试提取最后一段 JSON。
	start := strings.LastIndex(raw, "{")
	if start >= 0 {
		if err := json.Unmarshal([]byte(raw[start:]), &result); err == nil {
			return result, nil
		}
	}
	return providers.RunResult{}, fmt.Errorf("parse docker output as run result failed")
}
