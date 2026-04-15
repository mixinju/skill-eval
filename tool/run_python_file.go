package tool

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

type RunPythonFile struct{}

func (*RunPythonFile) params() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"script_path": map[string]any{
				"type":        "string",
				"description": "Python script path",
			},
			"args": map[string]any{
				"type":        "array",
				"description": "Optional script args",
				"items": map[string]any{
					"type": "string",
				},
			},
		},
		"required": []string{"script_path"},
	}
}

func (*RunPythonFile) Run(ctx context.Context, params map[string]any) (string, error) {
	scriptPath, ok := params["script_path"].(string)
	if !ok || scriptPath == "" {
		return "", fmt.Errorf("param 'script_path' is required")
	}

	cmdArgs := []string{scriptPath}
	if arr, ok := params["args"].([]any); ok {
		for _, v := range arr {
			cmdArgs = append(cmdArgs, fmt.Sprintf("%v", v))
		}
	}

	cmd := exec.CommandContext(ctx, "python3", cmdArgs...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("run python file failed: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

func (r *RunPythonFile) GetTools() []Tool {
	return []Tool{
		NewBaseToolInfo("run_python_file", "执行 Python 脚本文件", r.params(), r.Run),
	}
}
