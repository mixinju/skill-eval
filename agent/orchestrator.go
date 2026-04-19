package agent

import (
    "context"
    "encoding/json"
    "fmt"
    "log"

    "skill-eval/tool"

    "github.com/openai/openai-go/v3"
)

type Orchestrator struct {
    ChatProvider *openai.Client
    Context      *RunContext
}

func NewOrchestrator(chatProvider *openai.Client, agent AgentConfig) *Orchestrator {

    return &Orchestrator{
        ChatProvider: chatProvider,
        Context:      NewContext(agent),
    }
}

func NewContext(agent AgentConfig) *RunContext {

    // 构建系统消息
    messages := []openai.ChatCompletionMessageParamUnion{
        openai.UserMessage(agent.SystemPrompt),
    }

    // 构建Tool调用

    tools := make(map[string]tool.Tool)
    toolsMessage := make([]openai.ChatCompletionToolUnionParam, 0)

    for _, t := range agent.Tools {
        // 构建工具Map，方便直接调用
        tools[t.Name()] = t

        // 构建工具 message
        toolsMessage = append(toolsMessage, t.ChatCompletionToolUnionParam())
    }

    return &RunContext{
        Agent:            agent,
        Messages:         messages,
        ToolsCollections: tools,
        CurrentIteration: 1,
        Tools:            toolsMessage,
    }
}

type RunContext struct {
    Agent            AgentConfig
    Messages         []openai.ChatCompletionMessageParamUnion
    Tools            []openai.ChatCompletionToolUnionParam
    ToolsCollections map[string]tool.Tool
    CurrentIteration int
    TargetSkill      tool.Skill

    UsedToken int64
}

type ToolCallRecord struct {
    Name   string
    Input  string
    OutPut string
    Error  string
}

func (o Orchestrator) Run() {

    maxIterations := o.Context.Agent.MaxIterations
    // 最入最大循环
    for o.Context.CurrentIteration < maxIterations {
        o.Context.CurrentIteration++

        //userMessage := []openai.ChatCompletionMessageParamUnion{
        //    openai.UserMessage("你好呀"),
        //}
        //for _, m := range o.Context.Messages {
        //    fmt.Println(m.OfSystem)
        //}
        p := openai.ChatCompletionNewParams{
            Messages: o.Context.Messages,
            Tools:    o.Context.Tools,
            Model:    "glm-5",
        }

        original, _ := json.Marshal(p)
        fmt.Println(string(original))

        chatCompletion, chatErr := o.ChatProvider.Chat.Completions.New(
            context.Background(),
            p,
        )

        if chatErr != nil {
            log.Default().Printf("[ERROR] 大模型对话失败: %v", chatErr)
            return
        }

        if len(chatCompletion.Choices) == 0 {
            log.Printf("[INFO] Choices 数组为空")
            return
        }

        // 打印原始输出
        log.Default().Printf("[INFO] 迭代次数：%s, 对话返回: %v", o.Context.CurrentIteration, chatCompletion.RawJSON())

        choice := chatCompletion.Choices[0]

        // 更新token
        o.Context.UsedToken += chatCompletion.Usage.TotalTokens

        // 更新历史消息
        o.Context.Messages = append(o.Context.Messages, choice.Message.ToParam())

        for _, tc := range choice.Message.ToolCalls {

            name := tc.Function.Name

            // 调用结束
            if name == "finish" {
                log.Printf("调用结束")
                return
            }
            // 命中目标SKILL
            if name == o.Context.TargetSkill.Name {
                log.Printf("[Info] 命中目标SKILL")
            }
            toolExec, ok := o.Context.ToolsCollections[name]
            if !ok {
                log.Printf("[ERROR]: 大模型返回的工具不存在: %s", name)
                o.Context.Messages = append(o.Context.Messages, openai.ToolMessage("tool not found: "+name, tc.ID))
                continue
            }

            var params map[string]any

            if err := json.Unmarshal([]byte(tc.Function.Arguments), &params); err != nil {
                log.Printf("ERROR:  反序列化参数失败%v,原始信息:%s", err, tc.Function.RawJSON())
                continue
            }

            toolOutPut, toolCallErr := toolExec.Exec(context.Background(), params)
            if toolCallErr != nil {
                log.Printf("ERROR，调用工具失败；%s", toolCallErr)
            }

            // 构建工具执行结果信息
            o.Context.Messages = append(o.Context.Messages, openai.ToolMessage(toolOutPut, tc.ID))

        }
    }
}
