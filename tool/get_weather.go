package tool

import (
	"context"
	"fmt"
)

type GetWeather struct{}

func NewGetWeather() *GetWeather {
	return &GetWeather{}
}

func (w *GetWeather) Query(ctx context.Context, params map[string]any) (string, error) {
	city, ok := params["location"].(string)
	if !ok {
		return "", fmt.Errorf("param 'location' is required")
	}
	return fmt.Sprintf("%s 城市的温度是23-28摄氏度", city), nil
}

func (w *GetWeather) GetTools() []Tool {
	params := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"location": map[string]any{
				"type":        "string",
				"description": "城市名称",
			},
		},
		"required": []string{"location"},
	}
	return []Tool{
		NewBaseToolInfo("get_weather", "查询天气", params, w.Query),
	}
}
