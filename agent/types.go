package agent

import (
    "skill-eval/skill"

    "github.com/openai/openai-go/v3"
)

type Agent struct {
    Name         string
    Model        string
    BaseURL      string
    APIKey       string
    Skill        *skill.Skill
    MaxTokens    int
    MaxInters    int
    SystemPrompt string
}

type StopReason string

const (
    StopFinish    StopReason = "finish"
    StopMaxInters StopReason = "max_inters"
    StopMaxTokens StopReason = "max_tokens"
    StopError     StopReason = "error"
    StopTextReply StopReason = "text_reply"
)

type RunResult struct {
    FinalOutput string
    Artifacts   []string
    Messages    []openai.ChatCompletionMessageParamUnion
    ToolCalls   []ToolCallRecord
    TokensUsed  int
    Iterations  int
    StopReason  StopReason
    Error       error
}

type ToolCallRecord struct {
    Iteration int    `json:"iteration"`
    ToolName  string `json:"tool_name"`
    Input     string `json:"input"`
    Output    string `json:"output"`
    Error     string `json:"error,omitempty"`
}
