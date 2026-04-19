package agent

import (
    "encoding/json"
    "fmt"
    "log"

    "skill-eval/providers"
    "skill-eval/skill"
    "skill-eval/tool"

    "github.com/openai/openai-go/v3"
)

type Orchestrator struct {
    ModelProvider providers.OpenAIProvider
}

func NewOrchestrator() *Orchestrator {
    return &Orchestrator{}
}

type ChatFunc func(messages []openai.ChatCompletionMessageParamUnion, tools []tool.Tool) (*openai.ChatCompletion, error)

func (o *Orchestrator) Run(ag *Agent, input string, workspace string) *RunResult {
    rc := NewRunContext(ag, workspace)

    systemMsg := ag.SystemPrompt

    rc.State.Messages = append(rc.State.Messages, openai.SystemMessage(systemMsg))
    rc.State.Messages = append(rc.State.Messages, openai.UserMessage(input))

    // 构建系统默认工具
    tools := buildTools(workspace, *ag.Skill)

    toolMap := make(map[string]tool.Tool)
    for _, t := range tools {
        toolMap[t.Name()] = t
    }

    for rc.State.Iterations < ag.MaxInters {
        rc.State.Iterations++

        log.Printf("[%s] === Iteration %d/%d ===", ag.Name, rc.State.Iterations, ag.MaxInters)
        log.Printf("[%s] Sending %d messages to LLM", ag.Name, len(rc.State.Messages))
        for i, msg := range rc.State.Messages {
            msgJSON, _ := json.Marshal(msg)
            summary := string(msgJSON)
            if len(summary) > 500 {
                summary = summary[:500] + "...(truncated)"
            }
            log.Printf("[%s]   msg[%d]: %s", ag.Name, i, summary)
        }

        completion, err := o.ModelProvider.Chat(nil, ag.Model, rc.State.Messages, tools)
        if err != nil {
            log.Printf("[%s] LLM error: %v", ag.Name, err)
            rc.Emit(Event{Type: EventError, Iteration: rc.State.Iterations, Data: err.Error()})
            return buildResult(rc, StopError, "", nil, err)
        }

        if len(completion.Choices) == 0 {
            err := fmt.Errorf("llm returned empty choices")
            log.Printf("[%s] LLM error: %v", ag.Name, err)
            rc.Emit(Event{Type: EventError, Iteration: rc.State.Iterations, Data: err.Error()})
            return buildResult(rc, StopError, "", nil, err)
        }

        choice := completion.Choices[0]
        rc.State.TokensUsed += int(completion.Usage.TotalTokens)

        contentPreview := choice.Message.Content
        if len(contentPreview) > 300 {
            contentPreview = contentPreview[:300] + "...(truncated)"
        }
        var toolNames []string
        for _, tc := range choice.Message.ToolCalls {
            toolNames = append(toolNames, tc.Function.Name)
        }
        log.Printf("[%s] LLM response: tokens=%d, content=%q, tool_calls=%v",
            ag.Name, completion.Usage.TotalTokens, contentPreview, toolNames)
        rc.Emit(Event{Type: EventLLMCall, Iteration: rc.State.Iterations, Data: map[string]any{
            "content":    choice.Message.Content,
            "tool_calls": choice.Message.ToolCalls,
            "tokens":     completion.Usage.TotalTokens,
        }})

        if rc.State.TokensUsed >= ag.MaxTokens {
            return buildResult(rc, StopMaxTokens, choice.Message.Content, nil, nil)
        }

        if len(choice.Message.ToolCalls) == 0 {
            return buildResult(rc, StopTextReply, choice.Message.Content, nil, nil)
        }

        rc.State.Messages = append(rc.State.Messages, choice.Message.ToParam())

        for _, tc := range choice.Message.ToolCalls {
            if tc.Function.Name == "finish" {
                log.Printf("[%s] Tool call: finish, args: %s", ag.Name, tc.Function.Arguments)
                var args map[string]any
                err := json.Unmarshal([]byte(tc.Function.Arguments), &args)
                if err != nil {
                    return nil
                }
                result, _ := args["result"].(string)
                var artifacts []string
                if raw, ok := args["artifacts"]; ok {
                    if arr, ok := raw.([]any); ok {
                        for _, item := range arr {
                            if s, ok := item.(string); ok {
                                artifacts = append(artifacts, s)
                            }
                        }
                    }
                }
                rc.Emit(Event{Type: EventFinish, Iteration: rc.State.Iterations, Data: tc})
                return buildResult(rc, StopFinish, result, artifacts, nil)
            }

            record := ToolCallRecord{
                Iteration: rc.State.Iterations,
                ToolName:  tc.Function.Name,
                Input:     tc.Function.Arguments,
            }

            inputPreview := tc.Function.Arguments
            if len(inputPreview) > 300 {
                inputPreview = inputPreview[:300] + "...(truncated)"
            }
            log.Printf("[%s] Tool call: %s, input: %s", ag.Name, tc.Function.Name, inputPreview)

            t, ok := toolMap[tc.Function.Name]
            var toolOutput string
            if !ok {
                toolOutput = fmt.Sprintf("Error: unknown tool %q", tc.Function.Name)
                record.Error = toolOutput
                log.Printf("[%s] Tool error: %s", ag.Name, toolOutput)
            } else {
                var params map[string]any
                if err := json.Unmarshal([]byte(tc.Function.Arguments), &params); err != nil {
                    toolOutput = fmt.Sprintf("Error: invalid arguments: %s", err)
                    record.Error = toolOutput
                    log.Printf("[%s] Tool error: %s", ag.Name, toolOutput)
                } else {
                    output, err := t.Exec(ctx, params)
                    if err != nil {
                        toolOutput = fmt.Sprintf("Error: %s", err)
                        record.Error = err.Error()
                        log.Printf("[%s] Tool %s error: %v", ag.Name, tc.Function.Name, err)
                    } else {
                        toolOutput = output
                        outputPreview := output
                        if len(outputPreview) > 300 {
                            outputPreview = outputPreview[:300] + "...(truncated)"
                        }
                        log.Printf("[%s] Tool %s output: %s", ag.Name, tc.Function.Name, outputPreview)
                    }
                    record.Output = output
                }
            }

            rc.Emit(Event{Type: EventToolExec, Iteration: rc.State.Iterations, Data: record})
            rc.State.ToolCalls = append(rc.State.ToolCalls, record)
            rc.State.Messages = append(rc.State.Messages, openai.ToolMessage(toolOutput, tc.ID))
        }
    }

    return buildResult(rc, StopMaxInters, "", nil, nil)
}

func buildResult(rc *RunContext, reason StopReason, finalOutput string, artifacts []string, err error) *RunResult {
    return &RunResult{
        FinalOutput: finalOutput,
        Artifacts:   artifacts,
        Messages:    rc.State.Messages,
        ToolCalls:   rc.State.ToolCalls,
        TokensUsed:  rc.State.TokensUsed,
        Iterations:  rc.State.Iterations,
        StopReason:  reason,
        Error:       err,
    }
}

func buildTools(workspace string, skill skill.Skill) []tool.Tool {
    var tools []tool.Tool

    fs := tool.NewFileSystem(workspace)
    tools = append(tools, fs.GetTools()...)

    bash := tool.NewBash(workspace, 0)
    tools = append(tools, bash.GetTools()...)

    finish := tool.NewFinish()
    tools = append(tools, finish.GetTools()...)

    weather := tool.NewGetWeather()
    tools = append(tools, weather.GetTools()...)

    return tools
}
