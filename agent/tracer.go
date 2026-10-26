package agent

import "time"

// Runner 标识一次评测是在哪个执行后端上跑的。
// 目前只有自研内核（agent.Orchestrator）会写入 Trace。
// codex 与 claude-code 先作为契约保留，仓库里还没有对应执行器。
type Runner string

const (
	RunnerKernel     Runner = "kernel"
	RunnerCodex      Runner = "codex"
	RunnerClaudeCode Runner = "claude-code"
)

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
	EventSkillStart       EventType = "skill_start"
	EventSkillEnd         EventType = "skill_end"
	EventMarkArtifacts    EventType = "mark_artifacts"
)

type TraceEvent struct {
	Type      EventType `json:"type"`
	Timestamp time.Time `json:"timestamp"`
	Iteration int       `json:"iteration,omitempty"`

	// Run
	AgentName  string `json:"agentName,omitempty"`
	Model      string `json:"model,omitempty"`
	UserPrompt string `json:"userPrompt,omitempty"`
	Runner     Runner `json:"runner,omitempty"`
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

	// SKILL专属
	IsTarget    bool   `json:"isTarget,omitempty"`
	TargetSkill string `json:"targetSkill,omitempty"`

	Error string `json:"error,omitempty"`
}

type SpanKind string

const (
	SpanKindLLMCall     SpanKind = "llm_call"
	SpanKindLLMCompress SpanKind = "llm_compress"
	SpanKindToolCall    SpanKind = "tool_call"
)

type Span struct {
	SpanID    string    `json:"spanId"`
	ParentID  string    `json:"parentId,omitempty"`
	Kind      SpanKind  `json:"kind"`
	Name      string    `json:"name"`
	Iteration int       `json:"iteration"`
	StartTime time.Time `json:"startTime"`
	EndTime   time.Time `json:"endTime,omitempty"`
	Duration  int64     `json:"duration,omitempty"`

	InputMessages int    `json:"inputMessages,omitempty"`
	TotalTokens   int64  `json:"totalTokens,omitempty"`
	FinishReason  string `json:"finishReason,omitempty"`
	LLMInput      string `json:"llmInput,omitempty"`
	LLMOutput     string `json:"llmOutput,omitempty"`

	ToolInput  string `json:"toolInput,omitempty"`
	ToolOutput string `json:"toolOutput,omitempty"`
	IsTarget   bool   `json:"isTarget,omitempty"`

	Error string `json:"error,omitempty"`
}

type Trace struct {
	ID                 string    `json:"id"`
	AgentName          string    `json:"agentName"`
	Model              string    `json:"model"`
	UserPrompt         string    `json:"userPrompt"` //用户的首次输入
	TargetSkill        string    `json:"targetSkill,omitempty"`
	StartTime          time.Time `json:"startTime"`
	EndTime            time.Time `json:"endTime,omitempty"`
	TotalTokens        int64     `json:"totalTokens"`
	Iterations         int       `json:"iterations"`
	Success            bool      `json:"success"`
	Runner             Runner    `json:"runner,omitempty"`
	Spans              []*Span   `json:"spans"`
	ArtifactsAndResult string    `json:"artifactsAndResult,omitempty"`
}

type TracerHook interface {
	OnEvent(event TraceEvent)
	Id() string
	GetTrace() *Trace
}
