package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
)

type ReadJSON struct{}

func (*ReadJSON) params() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "JSON file path",
			},
		},
		"required": []string{"path"},
	}
}

func (*ReadJSON) Read(ctx context.Context, params map[string]any) (string, error) {
	path, ok := params["path"].(string)
	if !ok || path == "" {
		return "", fmt.Errorf("param 'path' is required")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read file failed: %w", err)
	}

	var v any
	if err := json.Unmarshal(data, &v); err != nil {
		return "", fmt.Errorf("invalid json file: %w", err)
	}

	pretty, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal json failed: %w", err)
	}

	return string(pretty), nil
}

func (r *ReadJSON) GetTools() []Tool {
	return []Tool{
		NewBaseToolInfo("read_json", "读取并格式化 JSON 文件", r.params(), r.Read),
	}
}
