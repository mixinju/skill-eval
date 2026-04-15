package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type WriteJSON struct{}

func (*WriteJSON) params() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "Target JSON file path",
			},
			"json_text": map[string]any{
				"type":        "string",
				"description": "JSON string content",
			},
		},
		"required": []string{"path", "json_text"},
	}
}

func (*WriteJSON) Write(ctx context.Context, params map[string]any) (string, error) {
	path, ok := params["path"].(string)
	if !ok || path == "" {
		return "", fmt.Errorf("param 'path' is required")
	}
	jsonText, ok := params["json_text"].(string)
	if !ok || jsonText == "" {
		return "", fmt.Errorf("param 'json_text' is required")
	}

	var v any
	if err := json.Unmarshal([]byte(jsonText), &v); err != nil {
		return "", fmt.Errorf("invalid json_text: %w", err)
	}

	pretty, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal json failed: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", fmt.Errorf("create dir failed: %w", err)
	}
	if err := os.WriteFile(path, pretty, 0o644); err != nil {
		return "", fmt.Errorf("write file failed: %w", err)
	}

	return fmt.Sprintf("json written to %s", path), nil
}

func (w *WriteJSON) GetTools() []Tool {
	return []Tool{
		NewBaseToolInfo("write_json", "校验后写入 JSON 文件", w.params(), w.Write),
	}
}
