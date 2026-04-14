package tool

import "github.com/openai/openai-go/v3"

type GetWeather struct {
}

func (w *GetWeather) Prams() openai.FunctionParameters {

    f := openai.FunctionParameters{
        "type": "object",
        "properties": map[string]interface{}{
            "location": map[string]string{
                "type":        "string",
                "description": "城市名称,例如北京、上海",
            },
        },
        "required": []string{"location"},
    }

    return f
}

func (w *GetWeather) Definition() openai.FunctionDefinitionParam {
    df := openai.FunctionDefinitionParam{
        Name:        "get_weather",
        Description: openai.String("获取某一个地区的天气"),
        Parameters:  w.Prams(),
    }

    return df
}

func (w *GetWeather) ToolParams() openai.ChatCompletionToolUnionParam {

    t := openai.ChatCompletionToolUnionParam{
        OfFunction: &openai.ChatCompletionFunctionToolParam{Function: w.Definition()},
    }

    return t
}

func (w *GetWeather) Query(location string) string {
    s := "北京今天多云，最高23-38摄氏度"

    return s
}
