package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"skill-eval/tool"

	"github.com/openai/openai-go/v3"
)

type Orchestrator struct {
	ChatProvider *openai.Client
	Context      *RunContext
}

func (o *Orchestrator) SetTargetSkill(name string) {
	if o.Context == nil {
		return
	}
	o.Context.TargetSkill = name
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
		openai.SystemMessage(agent.SystemPrompt),
		openai.UserMessage(agent.UserPrompt),
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
		Agent:             agent,
		Messages:          messages,
		ToolsCollections:  tools,
		CurrentIteration:  1,
		Tools:             toolsMessage,
		CompressThreshold: 20,
	}
}

type RunContext struct {
	Agent                 AgentConfig
	Messages              []openai.ChatCompletionMessageParamUnion
	Tools                 []openai.ChatCompletionToolUnionParam
	HasSelectedSkills     map[string]tool.Skill
	ToolsCollections      map[string]tool.Tool
	CurrentIteration      int
	TargetSkill           string //目标SKILL名称
	ConsecutiveNoToolCall int    // 连续无工具调用计数
	CompressThreshold     int    // 消息压缩阈值，默认20

	UsedToken int64
}

type ToolCallRecord struct {
	Name   string
	Input  string
	OutPut string
	Error  string
}

// 构建系统提示词
// loaded 表示需要被加载的技能的名称
func (rc *RunContext) buildSystemPrompt(loaded string) string {
	var sb strings.Builder
	a := rc.Agent

	// 保留已有的 SystemPrompt
	sb.WriteString(a.SystemPrompt)
	for _, s := range a.Skills {
		// 是否加载skill的content
		sb.WriteString(s.Prompt(loaded == s.Name))
	}

	rc.Messages[0] = openai.SystemMessage(sb.String())
	return sb.String()
}

func (o *Orchestrator) compressMessages() error {
	messages := o.Context.Messages
	threshold := o.Context.CompressThreshold
	if threshold <= 0 || len(messages) <= threshold {
		return nil
	}

	keepRecent := 4
	if len(messages)-2 <= keepRecent {
		return nil
	}

	middleMessages := messages[2 : len(messages)-keepRecent]

	var sb strings.Builder
	for _, m := range middleMessages {
		raw, err := json.Marshal(m)
		if err != nil {
			continue
		}
		sb.Write(raw)
		sb.WriteString("\n")
	}

	summaryReq := openai.ChatCompletionNewParams{
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage("你是一个对话摘要助手。请将以下对话历史压缩为简洁的摘要，保留关键信息、已完成的操作和重要结果。"),
			openai.UserMessage(sb.String()),
		},
		Model: o.Context.Agent.Model,
	}

	resp, err := o.ChatProvider.Chat.Completions.New(context.Background(), summaryReq)
	if err != nil {
		return fmt.Errorf("摘要请求失败: %w", err)
	}

	if len(resp.Choices) == 0 {
		return fmt.Errorf("摘要返回为空")
	}

	summary := resp.Choices[0].Message.Content
	log.Printf("[INFO] 消息压缩完成，从 %d 条压缩为 %d 条", len(messages), 2+1+keepRecent)

	compressed := make([]openai.ChatCompletionMessageParamUnion, 0, 2+1+keepRecent)
	compressed = append(compressed, messages[0], messages[1])
	compressed = append(compressed, openai.UserMessage("以下是之前对话历史的摘要：\n"+summary))
	compressed = append(compressed, messages[len(messages)-keepRecent:]...)
	o.Context.Messages = compressed

	return nil
}

func (o *Orchestrator) Run() {

	maxIterations := o.Context.Agent.MaxIterations
	// 最入最大循环

	// 初始化-不加载任何一个完整的skill
	o.Context.buildSystemPrompt("")
	for o.Context.CurrentIteration < maxIterations {
		o.Context.CurrentIteration++

		// 压缩对话消息
		if err := o.compressMessages(); err != nil {
			log.Printf("[WARN] 消息压缩失败: %v", err)
		}

		p := openai.ChatCompletionNewParams{
			Messages: o.Context.Messages,
			Tools:    o.Context.Tools,
			Model:    o.Context.Agent.Model,
			// 设置最大输出token，防止响应被截断
			// 对于包含大量内容的工具调用（如写入文件），需要足够的空间
			MaxTokens: openai.Int(16384),
		}

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
		log.Default().Printf("[INFO] 迭代次数：%d, 对话返回: %v \n", o.Context.CurrentIteration, chatCompletion.RawJSON())

		choice := chatCompletion.Choices[0]

		// 更新token
		o.Context.UsedToken += chatCompletion.Usage.TotalTokens

		// 更新历史消息
		o.Context.Messages = append(o.Context.Messages, choice.Message.ToParam())

		// 工具调用
		if len(choice.Message.ToolCalls) == 0 {
			o.Context.ConsecutiveNoToolCall++
			if o.Context.ConsecutiveNoToolCall >= 2 {
				log.Printf("[WARN] 连续%d次无工具调用，强制退出循环", o.Context.ConsecutiveNoToolCall)
				return
			}
			log.Printf("[INFO] 模型未调用工具，提醒使用finish工具")
			o.Context.Messages = append(o.Context.Messages, openai.UserMessage("如果任务已完成，请调用finish工具提交最终结果；如果未完成，请继续使用工具执行任务。"))
			continue
		}

		o.Context.ConsecutiveNoToolCall = 0

		for _, tc := range choice.Message.ToolCalls {

			name := tc.Function.Name
			var params map[string]any

			if err := json.Unmarshal([]byte(tc.Function.Arguments), &params); err != nil {
				log.Printf("ERROR:  反序列化参数失败%v,原始信息:%s", err, tc.Function.RawJSON())
				continue
			}

			// 调用结束
			if name == "finish" {
				log.Printf("调用结束")
				finishTool := o.Context.ToolsCollections["finish"]
				finishResult, finishError := finishTool.Exec(context.Background(), params)
				if finishError != nil {
					return
				}
				o.Context.Messages = append(o.Context.Messages, openai.ToolMessage(finishResult, tc.ID))
				return
			}
			// 命中目标SKILL
			if name == o.Context.TargetSkill {
				log.Printf("[Info] 命中目标SKILL")
			}

			// 技能调用
			if name == "use_skill" {
				skillName, ok := params["name"].(string)
				if !ok || skillName == "" {
					log.Printf("[ERROR] use_skill 参数 name 无效或不存在")
					o.Context.Messages = append(o.Context.Messages,
						openai.ToolMessage("参数错误: name 字段必须是字符串且不能为空", tc.ID))
					continue
				}

				o.Context.buildSystemPrompt(skillName)
				o.Context.Messages = append(o.Context.Messages, openai.ToolMessage("SKILL.md已加载", tc.ID))
				continue
			}
			toolExec, ok := o.Context.ToolsCollections[name]
			if !ok {
				log.Printf("[ERROR]: 大模型返回的工具不存在: %s", name)
				o.Context.Messages = append(o.Context.Messages, openai.ToolMessage("tool not found: "+name, tc.ID))
				continue
			}

			toolOutPut, toolCallErr := toolExec.Exec(context.Background(), params)
			if toolCallErr != nil {
				log.Printf("ERROR，调用工具失败；%s", toolCallErr)
				o.Context.Messages = append(o.Context.Messages, openai.ToolMessage("工具调用失败: "+name+toolCallErr.Error(), tc.ID))
				continue
			}

			// 构建工具执行结果信息
			o.Context.Messages = append(o.Context.Messages, openai.ToolMessage(toolOutPut, tc.ID))
		}
	}

	log.Printf("[WARN] 达到最大迭代次数(%d)，任务仍未完成", maxIterations)
}
