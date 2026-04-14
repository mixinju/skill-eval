package providers

import (
    "context"
    "encoding/json"
    "fmt"
    "log"

    "skill-eval/tool"

    "github.com/openai/openai-go/v3"
    "github.com/openai/openai-go/v3/option"
)

const (
    baseUrl = "https://open.bigmodel.cn/api/paas/v4"
    apiKey  = "412354e1dc5e4570b60665b46bde29fe.9n288zD658ySKAo1"
)

func Chat() {

    client := openai.NewClient(
        option.WithAPIKey(apiKey),
        option.WithBaseURL(baseUrl),
    )

    var messages []openai.ChatCompletionMessageParamUnion

    messages = append(messages, openai.UserMessage("今天北京的天气怎么样"))

    // 拼接数组
    var tools []openai.ChatCompletionToolUnionParam

    weather := tool.GetWeather{}
    tools = append(tools, weather.ToolParams())

    // 发起调用
    chatCompletion, err := client.Chat.Completions.New(context.TODO(), openai.ChatCompletionNewParams{
        Messages: messages,
        Model:    "glm-5",
        Tools:    tools,
    })

    if err != nil {
        log.Printf("调用客户端失败:{%s}", err)
    }

    if len(chatCompletion.Choices) == 0 {
        log.Printf("返回内容为空")
    }

    //fmt.Println(chatCompletion.RawJSON())

    assistantMessage := chatCompletion.Choices[0].Message

    log.Printf("第一次调用结果:%s", assistantMessage.RawJSON())

    if len(assistantMessage.ToolCalls) <= 0 {
        log.Printf("工具调用失败")
    }

    // 维护历史消息
    messages = append(messages, assistantMessage.ToParam())

    for _, tc := range assistantMessage.ToolCalls {
        if tc.Function.Name == "get_weather" {
            var args map[string]string
            if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
                log.Printf("解析参数失败")
            }

            location := args["location"]

            r := weather.Query(location)

            messages = append(messages, openai.ToolMessage(r, tc.ID))
        }
    }

    second, err := client.Chat.Completions.New(context.TODO(), openai.ChatCompletionNewParams{
        Messages: messages,
        Tools:    tools,
        Model:    "glm-5",
    })
    if err != nil {
        log.Fatalf("第二次调用失败")

    }

    fmt.Println("=====第二次调用结果=====")

    log.Printf("第二次结果 %s", second.RawJSON())

}
