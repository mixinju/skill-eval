package agent

import (
    "context"
    "encoding/json"
    "fmt"

    "skill-eval/tool"

    "github.com/openai/openai-go/v3"
)

type Orchestrator struct{}

func NewOrchestrator() *Orchestrator {
    return &Orchestrator{}
}

type ChatFunc func(ctx context.Context, messages []openai.ChatCompletionMessageParamUnion, tools []tool.Tool) (*openai.ChatCompletion, error)

func (o *Orchestrator) Run(ctx context.Context, ag *Agent, input string, workspace string, onEvent EventHandler, chat ChatFunc) *RunResult {
    rc := NewRunContext(ag, workspace, onEvent)

    systemMsg := ag.SystemPrompt
    if ag.Skill != nil {
        systemMsg += "\n\n## Skill\n" + ag.Skill.Content
    }
    rc.State.Messages = append(rc.State.Messages, openai.SystemMessage(systemMsg))
    rc.State.Messages = append(rc.State.Messages, openai.UserMessage(input))

    tools := buildTools(workspace, ag.Skill)

    toolMap := make(map[string]tool.Tool)
    for _, t := range tools {
        toolMap[t.Name()] = t
    }

    for rc.State.Iterations < ag.MaxInters {
        rc.State.Iterations++

        completion, err := chat(ctx, rc.State.Messages, tools)
        if err != nil {
            rc.Emit(Event{Type: EventError, Iteration: rc.State.Iterations, Data: err})
            return buildResult(rc, StopError, "", nil, err)
        }

        if len(completion.Choices) == 0 {
            err := fmt.Errorf("llm returned empty choices")
            rc.Emit(Event{Type: EventError, Iteration: rc.State.Iterations, Data: err})
            return buildResult(rc, StopError, "", nil, err)
        }

        choice := completion.Choices[0]
        rc.State.TokensUsed += int(completion.Usage.TotalTokens)
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
                var args map[string]any
                json.Unmarshal([]byte(tc.Function.Arguments), &args)
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

            t, ok := toolMap[tc.Function.Name]
            var toolOutput string
            if !ok {
                toolOutput = fmt.Sprintf("Error: unknown tool %q", tc.Function.Name)
                record.Error = toolOutput
            } else {
                var params map[string]any
                if err := json.Unmarshal([]byte(tc.Function.Arguments), &params); err != nil {
                    toolOutput = fmt.Sprintf("Error: invalid arguments: %s", err)
                    record.Error = toolOutput
                } else {
                    output, err := t.Exec(ctx, params)
                    if err != nil {
                        toolOutput = fmt.Sprintf("Error: %s", err)
                        record.Error = err.Error()
                    } else {
                        toolOutput = output
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

func buildTools(workspace string, sk any) []tool.Tool {
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
