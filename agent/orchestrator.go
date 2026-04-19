package agent

import (
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

    messages := []openai.ChatCompletionMessageParamUnion{
        openai.SystemMessage(agent.SystemPrompt),
    }

    tools := make(map[string]tool.Tool)

    for _, t := range agent.Tools {
        tools[t.Name()] = t
    }

    return &RunContext{
        Agent:            agent,
        Messages:         messages,
        ToolsCollections: tools,
        CurrentIteration: 1,
    }
}

type RunContext struct {
    Agent            AgentConfig
    Messages         []openai.ChatCompletionMessageParamUnion
    ToolsCollections map[string]tool.Tool
    CurrentIteration int
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

    }
}
