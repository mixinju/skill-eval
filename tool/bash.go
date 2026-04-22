package tool

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

type Bash struct {
	workspace string
	timeout   time.Duration
}

func NewBash(workspace string, timeout time.Duration) *Bash {
	if timeout == 0 {
		timeout = 2 * time.Minute
	}
	now := time.Now()
	if len(workspace) == 0 {
		workspace = filepath.Join("/Users/mixinju/Desktop/skill-eval-workplace/", now.Format("2006-01-02"))
	}

	return &Bash{workspace: workspace, timeout: timeout}
}

func (b *Bash) Exec(ctx context.Context, params map[string]any) (string, error) {
	command, ok := params["command"].(string)
	if !ok {
		return "", fmt.Errorf("缺少必填参数: command")
	}

	// 检查 workspace 目录是否存在，不存在则创建
	if _, err := os.Stat(b.workspace); os.IsNotExist(err) {
		if err := os.MkdirAll(b.workspace, 0755); err != nil {
			return "", fmt.Errorf("创建工作区目录失败: %w", err)
		}
	}

	ctx, cancel := context.WithTimeout(ctx, b.timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	cmd.Dir = b.workspace

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	output := stdout.String()
	if stderr.Len() > 0 {
		output += "\nSTDERR:\n" + stderr.String()
	}

	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return output, fmt.Errorf("命令执行超时，已超过 %s", b.timeout)
	}

	if err != nil {
		return output, fmt.Errorf("命令执行出错: %w", err)
	}

	return output, nil
}

func (b *Bash) GetTools() []Tool {
	params := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"command": map[string]any{
				"type":        "string",
				"description": "要执行的 shell 命令",
			},
		},
		"required": []string{"command"},
	}
	return []Tool{
		NewBaseToolInfo("bash", "执行 shell 命令", params, b.Exec),
	}
}
