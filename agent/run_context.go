package agent

import "github.com/openai/openai-go/v3"

type EventType string

const (
	EventLLMCall  EventType = "llm_call"
	EventToolExec EventType = "tool_exec"
	EventFinish   EventType = "finish"
	EventError    EventType = "error"
)

type Event struct {
	Type      EventType
	Iteration int
	Data      any
}

type EventHandler func(Event)

type State struct {
	Messages   []openai.ChatCompletionMessageParamUnion
	ToolCalls  []ToolCallRecord
	TokensUsed int
	Iterations int
}

type RunContext struct {
	Agent     *Agent
	Workspace string
	State     *State
	onEvent   EventHandler
}

func NewRunContext(agent *Agent, workspace string, handler EventHandler) *RunContext {
	return &RunContext{
		Agent:     agent,
		Workspace: workspace,
		State:     &State{},
		onEvent:   handler,
	}
}

func (rc *RunContext) Emit(e Event) {
	if rc.onEvent != nil {
		rc.onEvent(e)
	}
}
