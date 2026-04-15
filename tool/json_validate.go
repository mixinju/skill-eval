package tool

import (
	"context"
	"encoding/json"
	"fmt"
)

type JSONValidate struct{}

func (*JSONValidate) params() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"json_text": map[string]any{
				"type":        "string",
				"description": "JSON text to validate",
			},
		},
		"required": []string{"json_text"},
	}
}

func (*JSONValidate) Validate(ctx context.Context, params map[string]any) (string, error) {
	jsonText, ok := params["json_text"].(string)
	if !ok || jsonText == "" {
		return "", fmt.Errorf("param 'json_text' is required")
	}

	var v any
	if err := json.Unmarshal([]byte(jsonText), &v); err != nil {
		return "invalid json: " + err.Error(), nil
	}
	return "valid json", nil
}

func (j *JSONValidate) GetTools() []Tool {
	return []Tool{
		NewBaseToolInfo("json_validate", "校验 JSON 文本是否合法", j.params(), j.Validate),
	}
}
