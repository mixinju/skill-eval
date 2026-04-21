package tool

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/google/uuid"
)

type Bash struct {
	workspace string
	timeout   time.Duration
}

func NewBash(workspace string, timeout time.Duration) *Bash {
	if timeout == 0 {
		timeout = 2 * time.Minute
	}
	id, _ := uuid.NewUUID()
	now := time.Now()
	if len(workspace) == 0 {
		workspace = filepath.Join("~/Desktop/skil-eval-workplace/", now.Format("20060102")+id.String())
	}

	return &Bash{workspace: workspace, timeout: timeout}
}

func (b *Bash) Exec(ctx context.Context, params map[string]any) (string, error) {
	command, ok := params["command"].(string)
	if !ok {
		return "", fmt.Errorf("command parameter is required")
	}

	ctx, cancel := context.WithTimeout(ctx, b.timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "zsh", "-c", command)
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
		return output, fmt.Errorf("command timed out after %s", b.timeout)
	}

	if err != nil {
		return output, fmt.Errorf("command exited with error: %w", err)
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
