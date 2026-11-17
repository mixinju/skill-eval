package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"skill-eval/tool"

	"github.com/openai/openai-go/v3"
	"github.com/sirupsen/logrus"
)

const maxModelToolResult = 24 * 1024

type Orchestrator struct {
	Chat    Completer
	Context *RunContext
	Tracer  TracerHook

	// MaxAttempts / RetryBase 控制大模型调用的重试。零值使用默认（3 次、200ms 起）。
	MaxAttempts int
	RetryBase   time.Duration
}

func (o *Orchestrator) GetTrace() *Trace {
	if o.Tracer != nil {
		return o.Tracer.GetTrace()
	}
	return nil
}

func (o *Orchestrator) SetTargetSkill(name string) {
	if o.Context == nil {
		return
	}
	o.Context.TargetSkill = name
}

func NewOrchestrator(chat Completer, agent AgentConfig) *Orchestrator {
	return &Orchestrator{
		Chat:    chat,
		Context: NewContext(agent),
	}
}

func NewContext(agent AgentConfig) *RunContext {
	messages := []openai.ChatCompletionMessageParamUnion{
		openai.SystemMessage(agent.SystemPrompt),
		openai.UserMessage(agent.UserPrompt),
	}

	tools := make(map[string]tool.Tool)
	toolsMessage := make([]openai.ChatCompletionToolUnionParam, 0)

	for _, t := range agent.Tools {
		tools[t.Name()] = t
		toolsMessage = append(toolsMessage, t.ChatCompletionToolUnionParam())
	}

	return &RunContext{
		Agent:             agent,
		Messages:          messages,
		ToolsCollections:  tools,
		CurrentIteration:  0,
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
	LastMessageIndex      int    // 上次 LLM 调用时的消息位置，用于计算增量输入

	UsedToken int64
}

type ToolCallRecord struct {
	Name   string
	Input  string
	OutPut string
	Error  string
}

// buildSystemPrompt 重建系统提示词。loaded 是需要展开正文的技能名称。
func (rc *RunContext) buildSystemPrompt(loaded string) string {
	var sb strings.Builder
	a := rc.Agent

	sb.WriteString(a.SystemPrompt)
	for _, s := range a.Skills {
		sb.WriteString(s.Prompt(loaded == s.Name))
	}

	if len(rc.Messages) == 0 {
		rc.Messages = append(rc.Messages, openai.SystemMessage(sb.String()))
	} else {
		rc.Messages[0] = openai.SystemMessage(sb.String())
	}
	return sb.String()
}

func (rc *RunContext) hasSkill(name string) bool {
	for _, s := range rc.Agent.Skills {
		if s.Name == name {
			return true
		}
	}
	return false
}

func (o *Orchestrator) emit(event TraceEvent) {
	if o.Tracer != nil {
		o.Tracer.OnEvent(event)
	}
}

func (o *Orchestrator) compress(ctx context.Context) error {
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

	o.emit(TraceEvent{Type: EventLLMCompressStart, Iteration: o.Context.CurrentIteration, MessageCount: len(middleMessages)})

	resp, err := o.complete(ctx, summaryReq)
	if err != nil {
		o.emit(TraceEvent{Type: EventLLMCompressEnd, Error: err.Error()})
		return fmt.Errorf("摘要请求失败: %w", err)
	}

	if len(resp.Choices) == 0 {
		o.emit(TraceEvent{Type: EventLLMCompressEnd, Error: "摘要返回为空"})
		return fmt.Errorf("摘要返回为空")
	}

	o.emit(TraceEvent{Type: EventLLMCompressEnd, TotalTokens: resp.Usage.TotalTokens})

	summary := resp.Choices[0].Message.Content
	logrus.Infof("消息压缩完成，从 %d 条压缩为 %d 条", len(messages), 2+1+keepRecent)

	compressed := make([]openai.ChatCompletionMessageParamUnion, 0, 2+1+keepRecent)
	compressed = append(compressed, messages[0], messages[1])
	compressed = append(compressed, openai.UserMessage("以下是之前对话历史的摘要：\n"+summary))
	compressed = append(compressed, messages[len(messages)-keepRecent:]...)
	o.Context.Messages = compressed

	return nil
}

// Run 使用背景上下文执行 agent loop。
func (o *Orchestrator) Run() error {
	return o.RunContext(context.Background())
}

// RunContext 执行 agent loop。失败时返回明确错误，trace 中同时记录原因与步骤。
func (o *Orchestrator) RunContext(ctx context.Context) error {
	if o.Chat == nil {
		return fmt.Errorf("未配置对话客户端")
	}
	if o.Context == nil {
		return fmt.Errorf("未初始化运行上下文")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	maxIterations := o.Context.Agent.MaxIterations
	if maxIterations <= 0 {
		maxIterations = 10
	}

	o.emit(TraceEvent{
		Type:        EventRunStart,
		AgentName:   o.Context.Agent.Name,
		Model:       o.Context.Agent.Model,
		UserPrompt:  o.Context.Agent.UserPrompt,
		TargetSkill: o.Context.TargetSkill,
	})

	success := false
	var runErr error
	defer func() {
		if rec := recover(); rec != nil {
			runErr = fmt.Errorf("agent loop panic: %v", rec)
			logrus.Errorf("%v", runErr)
		}
		reason := ""
		if runErr != nil {
			reason = runErr.Error()
		}
		o.emit(TraceEvent{
			Type:        EventRunEnd,
			Success:     success,
			Iteration:   o.Context.CurrentIteration,
			TotalTokens: o.Context.UsedToken,
			Error:       reason,
		})
	}()

	o.Context.buildSystemPrompt("")
	for {
		if err := ctx.Err(); err != nil {
			runErr = contextFailure(err)
			return runErr
		}
		if o.Context.CurrentIteration >= maxIterations {
			runErr = fmt.Errorf("达到最大迭代次数(%d)，任务仍未完成", maxIterations)
			logrus.Warn(runErr.Error())
			return runErr
		}
		o.Context.CurrentIteration++

		if err := o.compress(ctx); err != nil {
			logrus.Warnf("消息压缩失败: %v", err)
			o.emit(TraceEvent{Type: EventNote, Iteration: o.Context.CurrentIteration, Message: "消息压缩失败: " + err.Error()})
		}

		incrementalMessage := o.Context.Messages[o.Context.LastMessageIndex:]
		incrementalJSON, _ := json.Marshal(incrementalMessage)

		o.emit(TraceEvent{
			Type:         EventLLMStart,
			Iteration:    o.Context.CurrentIteration,
			MessageCount: len(o.Context.Messages),
			LLMInput:     string(incrementalJSON),
		})

		p := openai.ChatCompletionNewParams{
			Messages:  o.Context.Messages,
			Tools:     o.Context.Tools,
			Model:     o.Context.Agent.Model,
			MaxTokens: openai.Int(16384),
		}

		chatCompletion, chatErr := o.complete(ctx, p)
		if chatErr != nil {
			o.emit(TraceEvent{Type: EventLLMEnd, Iteration: o.Context.CurrentIteration, Error: chatErr.Error()})
			if cerr := ctx.Err(); cerr != nil {
				runErr = contextFailure(cerr)
			} else {
				runErr = fmt.Errorf("大模型对话失败: %w", chatErr)
			}
			logrus.Errorf("%v", runErr)
			return runErr
		}

		if len(chatCompletion.Choices) == 0 {
			o.emit(TraceEvent{Type: EventLLMEnd, Iteration: o.Context.CurrentIteration, Error: "Choices为空"})
			runErr = fmt.Errorf("大模型返回为空")
			logrus.Error(runErr.Error())
			return runErr
		}

		choice := chatCompletion.Choices[0]
		logrus.Infof("大模型的返回:%s", choice.RawJSON())

		o.emit(TraceEvent{
			Type:         EventLLMEnd,
			Iteration:    o.Context.CurrentIteration,
			TotalTokens:  chatCompletion.Usage.TotalTokens,
			FinishReason: choice.FinishReason,
			LLMOutput:    choice.Message.RawJSON(),
		})

		o.Context.LastMessageIndex = len(o.Context.Messages)
		o.Context.UsedToken += chatCompletion.Usage.TotalTokens
		o.Context.Messages = append(o.Context.Messages, choice.Message.ToParam())

		if len(choice.Message.ToolCalls) == 0 {
			o.Context.ConsecutiveNoToolCall++
			if o.Context.ConsecutiveNoToolCall >= 2 {
				runErr = fmt.Errorf("连续%d次无工具调用，任务未调用 finish", o.Context.ConsecutiveNoToolCall)
				logrus.Warn(runErr.Error())
				return runErr
			}
			logrus.Info("模型未调用工具，提醒使用finish工具")
			o.Context.Messages = append(o.Context.Messages, openai.UserMessage("如果任务已完成，请调用finish工具提交最终结果；如果未完成，请继续使用工具执行任务。"))
			continue
		}

		o.Context.ConsecutiveNoToolCall = 0

		for _, tc := range choice.Message.ToolCalls {
			if err := ctx.Err(); err != nil {
				runErr = contextFailure(err)
				return runErr
			}

			name := tc.Function.Name
			var params map[string]any

			if err := json.Unmarshal([]byte(tc.Function.Arguments), &params); err != nil {
				errorMsg := fmt.Sprintf("参数解析失败: %v。原始参数: %s。请检查参数格式是否正确，确保是有效的JSON对象。", err, tc.Function.Arguments)
				logrus.Errorf("%s", errorMsg)
				o.emit(TraceEvent{Type: EventToolStart, Iteration: o.Context.CurrentIteration, CallID: tc.ID, ToolName: name, ToolInput: tc.Function.Arguments})
				o.emit(TraceEvent{Type: EventToolEnd, CallID: tc.ID, Error: errorMsg})
				o.Context.Messages = append(o.Context.Messages, openai.ToolMessage(errorMsg, tc.ID))
				continue
			}

			if name == "finish" {
				logrus.Info("调用结束")
				o.emit(TraceEvent{Type: EventToolStart, Iteration: o.Context.CurrentIteration, CallID: tc.ID, ToolName: name, ToolInput: tc.Function.Arguments})
				finishTool := o.Context.ToolsCollections["finish"]
				if finishTool == nil {
					runErr = fmt.Errorf("finish 工具未注册")
					o.emit(TraceEvent{Type: EventToolEnd, CallID: tc.ID, Error: runErr.Error()})
					return runErr
				}
				finishResult, finishError := finishTool.Exec(ctx, params)
				if finishError != nil {
					o.emit(TraceEvent{Type: EventToolEnd, CallID: tc.ID, Error: finishError.Error()})
					runErr = fmt.Errorf("finish 工具执行失败: %w", finishError)
					return runErr
				}
				o.emit(TraceEvent{Type: EventToolEnd, CallID: tc.ID, ToolName: name, ToolOutput: finishResult})
				o.Context.Messages = append(o.Context.Messages, openai.ToolMessage(finishResult, tc.ID))
				success = true
				return nil
			}

			if name == "use_skill" {
				skillName, _ := params["name"].(string)
				isTarget := skillName != "" && skillName == o.Context.TargetSkill
				o.emit(TraceEvent{
					Type:      EventSkillStart,
					Iteration: o.Context.CurrentIteration,
					CallID:    tc.ID,
					ToolName:  name,
					ToolInput: tc.Function.Arguments,
					IsTarget:  isTarget,
				})
				if skillName == "" {
					msg := "参数错误: name 字段必须是字符串且不能为空"
					logrus.Error(msg)
					o.emit(TraceEvent{Type: EventSkillEnd, CallID: tc.ID, Error: msg})
					o.Context.Messages = append(o.Context.Messages, openai.ToolMessage(msg, tc.ID))
					continue
				}
				if !o.Context.hasSkill(skillName) {
					msg := "未找到 skill: " + skillName
					logrus.Error(msg)
					o.emit(TraceEvent{Type: EventSkillEnd, CallID: tc.ID, Error: msg})
					o.Context.Messages = append(o.Context.Messages, openai.ToolMessage(msg, tc.ID))
					continue
				}
				if isTarget {
					logrus.Info("命中目标SKILL")
				}
				o.Context.buildSystemPrompt(skillName)
				o.emit(TraceEvent{Type: EventSkillEnd, CallID: tc.ID, ToolName: name, ToolOutput: "SKILL.md已加载: " + skillName})
				o.Context.Messages = append(o.Context.Messages, openai.ToolMessage("SKILL.md已加载: "+skillName, tc.ID))
				continue
			}

			toolExec, ok := o.Context.ToolsCollections[name]
			if !ok {
				msg := "tool not found: " + name
				logrus.Errorf("大模型返回的工具不存在: %s", name)
				o.emit(TraceEvent{Type: EventToolStart, Iteration: o.Context.CurrentIteration, CallID: tc.ID, ToolName: name, ToolInput: tc.Function.Arguments})
				o.emit(TraceEvent{Type: EventToolEnd, CallID: tc.ID, Error: msg})
				o.Context.Messages = append(o.Context.Messages, openai.ToolMessage(msg, tc.ID))
				continue
			}

			isTarget := name == o.Context.TargetSkill && o.Context.TargetSkill != ""
			o.emit(TraceEvent{Type: EventToolStart, Iteration: o.Context.CurrentIteration, CallID: tc.ID, ToolName: name, ToolInput: tc.Function.Arguments, IsTarget: isTarget})
			toolOutPut, toolCallErr := toolExec.Exec(ctx, params)
			if toolCallErr != nil {
				logrus.Errorf("调用工具失败；%s", toolCallErr)
				o.emit(TraceEvent{Type: EventToolEnd, CallID: tc.ID, Error: toolCallErr.Error()})
				o.Context.Messages = append(o.Context.Messages, openai.ToolMessage(clipForModel("工具调用失败: "+name+" "+toolCallErr.Error()), tc.ID))
				continue
			}

			o.emit(TraceEvent{Type: EventToolEnd, CallID: tc.ID, ToolName: name, ToolOutput: toolOutPut})
			o.Context.Messages = append(o.Context.Messages, openai.ToolMessage(clipForModel(toolOutPut), tc.ID))
		}
	}
}

func contextFailure(err error) error {
	if errors.Is(err, context.DeadlineExceeded) {
		return fmt.Errorf("执行超时")
	}
	return fmt.Errorf("任务已取消")
}

func clipForModel(s string) string {
	if len(s) <= maxModelToolResult {
		return s
	}
	return s[:maxModelToolResult] + "\n...(truncated)"
}
