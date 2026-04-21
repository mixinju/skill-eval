package agent

import "time"

type EventType string

const (
	EventRunStart         EventType = "run_start"
	EventRunEnd           EventType = "run_end"
	EventLLMStart         EventType = "llm_start"
	EventLLMEnd           EventType = "llm_end"
	EventLLMCompressStart EventType = "llm_compress_start"
	EventLLMCompressEnd   EventType = "llm_compress_end"
	EventToolStart        EventType = "tool_start"
	EventToolEnd          EventType = "tool_end"
	EventTargetSkillHit   EventType = "target_skill_hit"
)

type TraceEvent struct {
	Type      EventType `json:"type"`
	Timestamp time.Time `json:"timestamp"`
	Iteration int       `json:"iteration,omitempty"`

	// Run
	AgentName  string `json:"agentName,omitempty"`
	Model      string `json:"model,omitempty"`
	UserPrompt string `json:"userPrompt,omitempty"`
	Success    bool   `json:"success,omitempty"`

	// LLM
	MessageCount int    `json:"messageCount,omitempty"`
	TotalTokens  int64  `json:"totalTokens,omitempty"`
	FinishReason string `json:"finishReason,omitempty"`
	LLMInput     string `json:"llmInput,omitempty"`
	LLMOutput    string `json:"llmOutput,omitempty"`

	// Tool
	CallID     string `json:"callId,omitempty"`
	ToolName   string `json:"toolName,omitempty"`
	ToolInput  string `json:"toolInput,omitempty"`
	ToolOutput string `json:"toolOutput,omitempty"`

	Error string `json:"error,omitempty"`
}

type SpanKind string

const (
	SpanKindLLMCall     SpanKind = "llm_call"
	SpanKindLLMCompress SpanKind = "llm_compress"
	SpanKindToolCall    SpanKind = "tool_call"
)

type Span struct {
	SpanID    string        `json:"spanId"`
	ParentID  string        `json:"parentId,omitempty"`
	Kind      SpanKind      `json:"kind"`
	Name      string        `json:"name"`
	Iteration int           `json:"iteration"`
	StartTime time.Time     `json:"startTime"`
	EndTime   time.Time     `json:"endTime,omitempty"`
	Duration  time.Duration `json:"duration,omitempty"`

	InputMessages int    `json:"inputMessages,omitempty"`
	TotalTokens   int64  `json:"totalTokens,omitempty"`
	FinishReason  string `json:"finishReason,omitempty"`
	LLMInput      string `json:"llmInput,omitempty"`
	LLMOutput     string `json:"llmOutput,omitempty"`

	ToolInput  string `json:"toolInput,omitempty"`
	ToolOutput string `json:"toolOutput,omitempty"`

	Error string `json:"error,omitempty"`
}

type Trace struct {
	ID                string    `json:"id"`
	AgentName         string    `json:"agentName"`
	Model             string    `json:"model"`
	UserPrompt        string    `json:"userPrompt"`
	StartTime         time.Time `json:"startTime"`
	EndTime           time.Time `json:"endTime,omitempty"`
	TotalTokens       int64     `json:"totalTokens"`
	Iterations        int       `json:"iterations"`
	Success           bool      `json:"success"`
	TargetSkillHit    bool      `json:"targetSkillHit"`
	TargetSkillName   string    `json:"targetSkillName,omitempty"`
	TargetSkillHitAt  int       `json:"targetSkillHitAt,omitempty"`
	Spans             []*Span   `json:"spans"`
}

type TracerHook interface {
	OnEvent(event TraceEvent)
	Id() string
}
