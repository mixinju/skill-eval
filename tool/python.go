package tool

import (
	"context"
	"fmt"
	"os/exec"
)

type Python struct{}

func (*Python) runParams() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"code": map[string]any{
				"type":        "string",
				"description": "Python code to execute",
			},
		},
		"required": []string{"code"},
	}
}

func (*Python) Run(ctx context.Context, params map[string]any) (string, error) {
	code, ok := params["code"].(string)
	if !ok || code == "" {
		return "", fmt.Errorf("param 'code' is required")
	}

	cmd := exec.CommandContext(ctx, "python3", "-c", code)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("python execution failed: %w", err)
	}
	return string(out), nil
}

func (p *Python) GetTools() []Tool {
	return []Tool{
		NewBaseToolInfo("python", "执行 python3 代码", p.runParams(), p.Run),
	}
}
