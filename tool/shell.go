package tool

import (
	"context"
	"fmt"
	"os/exec"
)

type Shell struct{}

func (*Shell) runParams() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"command": map[string]any{
				"type":        "string",
				"description": "Shell command to execute",
			},
		},
		"required": []string{"command"},
	}
}

func (*Shell) Run(ctx context.Context, params map[string]any) (string, error) {
	command, ok := params["command"].(string)
	if !ok || command == "" {
		return "", fmt.Errorf("param 'command' is required")
	}

	cmd := exec.CommandContext(ctx, "sh", "-lc", command)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("shell command failed: %w", err)
	}
	return string(out), nil
}

func (s *Shell) GetTools() []Tool {
	return []Tool{
		NewBaseToolInfo("shell", "执行 shell 命令", s.runParams(), s.Run),
	}
}
