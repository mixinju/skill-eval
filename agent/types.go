package agent

import "strings"

// State 应该表示一次状态
type State struct {
}

// RunnerMeta 记录一次评测实际使用的执行后端，便于复核与对比。
// 不包含密钥或环境变量值。
type RunnerMeta struct {
	Type        string `json:"type"`
	DisplayName string `json:"displayName,omitempty"`
	Binary      string `json:"binary,omitempty"`
	Version     string `json:"version,omitempty"`
	Model       string `json:"model,omitempty"`
	Isolated    bool   `json:"isolated"`
	SkillDir    string `json:"skillDir,omitempty"`
	Command     string `json:"command,omitempty"`
	LogPath     string `json:"logPath,omitempty"`
}

// Step 是 Span 的紧凑视图，方便不解析完整链路时复核执行过程。
type Step struct {
	Index     int    `json:"index"`
	Kind      string `json:"kind"`
	Name      string `json:"name"`
	Iteration int    `json:"iteration"`
	OK        bool   `json:"ok"`
	Summary   string `json:"summary,omitempty"`
	Error     string `json:"error,omitempty"`
}

// BuildSteps 从 span 列表生成可复核的步骤摘要。
func BuildSteps(spans []*Span) []Step {
	steps := make([]Step, 0, len(spans))
	for i, span := range spans {
		if span == nil {
			continue
		}
		summary := span.ToolOutput
		if summary == "" {
			summary = span.LLMOutput
		}
		if summary == "" {
			summary = span.ToolInput
		}
		steps = append(steps, Step{
			Index:     i + 1,
			Kind:      string(span.Kind),
			Name:      span.Name,
			Iteration: span.Iteration,
			OK:        span.Error == "",
			Summary:   clipText(summary, 180),
			Error:     span.Error,
		})
	}
	return steps
}

func clipText(s string, n int) string {
	if n <= 0 || len(s) <= n {
		return s
	}
	cut := s[:n]
	for len(cut) > 0 && cut[len(cut)-1]&0xC0 == 0x80 {
		cut = cut[:len(cut)-1]
	}
	return strings.TrimSpace(cut) + "…"
}
