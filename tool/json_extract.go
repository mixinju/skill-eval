package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

type JSONExtract struct{}

func (*JSONExtract) params() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"json_text": map[string]any{
				"type":        "string",
				"description": "Input JSON string",
			},
			"path": map[string]any{
				"type":        "string",
				"description": "Dot path like a.b.0.c",
			},
		},
		"required": []string{"json_text", "path"},
	}
}

func (*JSONExtract) Extract(ctx context.Context, params map[string]any) (string, error) {
	jsonText, ok := params["json_text"].(string)
	if !ok || jsonText == "" {
		return "", fmt.Errorf("param 'json_text' is required")
	}
	path, ok := params["path"].(string)
	if !ok || path == "" {
		return "", fmt.Errorf("param 'path' is required")
	}

	var data any
	if err := json.Unmarshal([]byte(jsonText), &data); err != nil {
		return "", fmt.Errorf("invalid json_text: %w", err)
	}

	current := data
	parts := strings.Split(path, ".")
	for _, p := range parts {
		switch node := current.(type) {
		case map[string]any:
			v, exists := node[p]
			if !exists {
				return "", fmt.Errorf("path not found at key %q", p)
			}
			current = v
		case []any:
			idx, err := strconv.Atoi(p)
			if err != nil || idx < 0 || idx >= len(node) {
				return "", fmt.Errorf("invalid array index %q", p)
			}
			current = node[idx]
		default:
			return "", fmt.Errorf("path traversal failed at %q", p)
		}
	}

	b, err := json.Marshal(current)
	if err != nil {
		return "", fmt.Errorf("marshal result failed: %w", err)
	}
	return string(b), nil
}

func (j *JSONExtract) GetTools() []Tool {
	return []Tool{
		NewBaseToolInfo("json_extract", "从 JSON 文本提取字段", j.params(), j.Extract),
	}
}
