package tool

import (
    "context"
    "encoding/json"
    "fmt"
)

type Finish struct{}

func NewFinish() *Finish {
    return &Finish{}
}

type finishResult struct {
    Result    string   `json:"result"`
    Artifacts []string `json:"artifacts,omitempty"`
}

func (f *Finish) Exec(ctx context.Context, params map[string]any) (string, error) {
    result, _ := params["result"].(string)

    var artifacts []string
    if raw, ok := params["artifacts"]; ok {
        switch v := raw.(type) {
        case []any:
            for _, item := range v {
                if s, ok := item.(string); ok {
                    artifacts = append(artifacts, s)
                }
            }
        case []string:
            artifacts = v
        }
    }

    fr := finishResult{Result: result, Artifacts: artifacts}
    data, err := json.Marshal(fr)
    if err != nil {
        return "", fmt.Errorf("序列化完成结果失败: %w", err)
    }
    return string(data), nil
}

func (f *Finish) GetTools() []Tool {
    params := map[string]any{
        "type": "object",
        "properties": map[string]any{
            "result": map[string]any{
                "type":        "string",
                "description": "最终输出结果",
            },
            "artifacts": map[string]any{
                "type":        "array",
                "items":       map[string]any{"type": "string"},
                "description": "产物文件路径列表",
            },
        },
        "required": []string{"result"},
    }
    return []Tool{
        NewBaseToolInfo("finish", "任务完成时调用，提交最终结果和产物文件", params, f.Exec),
    }
}
