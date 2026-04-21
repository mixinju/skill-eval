package tool

import (
	"context"
	"fmt"
)

type GetWeather struct {
}

func NewGetWeather() *GetWeather {
	return &GetWeather{}
}

// Query 定义查询天气的工具
func (w *GetWeather) Query(ctx context.Context, params map[string]any) (string, error) {
	city, ok := params["location"].(string)
	if !ok {
		return "", fmt.Errorf("param 'city' is required")
	}

	s := fmt.Sprintf(" %s The city's temperature is 23-28°C, with gusts of force 5-6. 城市的温度是23-28摄氏度,5-6级阵风", city)

	return s, nil
}

// GetTools 返回所有执行的tool
func (w *GetWeather) GetTools() []Tool {

	queryPrams := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"location": map[string]any{
				"type": "string",
			},
		},
		"required": []string{"location"},
	}
	tools := []Tool{
		NewBaseToolInfo("get_weather", "查询天气", queryPrams, w.Query),
	}

	return tools

}
