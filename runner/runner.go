package runner

import (
	"context"
	"fmt"
	"strings"
	"time"

	"skill-eval/agent"

	"github.com/google/uuid"
)

// Type 是稳定的执行后端标识，前端和 API 都使用这些值。
type Type string

const (
	TypeSelf       Type = "self"
	TypeCodex      Type = "codex"
	TypeClaudeCode Type = "claude-code"
)

// Task 是一次 skill 评测的输入。各 runner 把它转换成自己的执行方式，但结果都落成 agent.Trace。
type Task struct {
	ID            string
	Name          string
	Description   string
	SystemPrompt  string
	UserPrompt    string
	Model         string
	TargetSkill   string
	SkillDir      string
	Workspace     string
	MaxIterations int
	Timeout       time.Duration
	DataDir       string
}

// Availability 描述某个 runner 现在能不能跑。
type Availability struct {
	Type      Type   `json:"type"`
	Name      string `json:"name"`
	Available bool   `json:"available"`
	Detail    string `json:"detail"`
	Binary    string `json:"binary,omitempty"`
	Version   string `json:"version,omitempty"`
}

// Runner 是可插拔执行后端。
type Runner interface {
	Type() Type
	DisplayName() string
	Probe(ctx context.Context) Availability
	Run(ctx context.Context, task Task) (*agent.Trace, error)
}

// Normalize 接受 self / codex / claude-code 以及常见别名。
func Normalize(raw string) (Type, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "self", "kernel", "builtin":
		return TypeSelf, nil
	case "codex":
		return TypeCodex, nil
	case "claude", "claude-code", "claudecode", "claude_code":
		return TypeClaudeCode, nil
	default:
		return "", fmt.Errorf("未知 runner %q，可选 self、codex、claude-code", raw)
	}
}

// Registry 保存已注册的执行后端。
type Registry struct {
	runners []Runner
}

func NewDefaultRegistry() *Registry {
	return NewRegistry(NewSelfRunner(), NewCodexRunner(), NewClaudeCodeRunner())
}

func NewRegistry(runners ...Runner) *Registry {
	return &Registry{runners: runners}
}

func (r *Registry) All() []Runner {
	if r == nil {
		return nil
	}
	out := make([]Runner, len(r.runners))
	copy(out, r.runners)
	return out
}

func (r *Registry) Get(t Type) (Runner, bool) {
	if r == nil {
		return nil, false
	}
	for _, item := range r.runners {
		if item.Type() == t {
			return item, true
		}
	}
	return nil, false
}

func (r *Registry) Probe(ctx context.Context) []Availability {
	out := make([]Availability, 0, len(r.runners))
	for _, item := range r.runners {
		out = append(out, item.Probe(ctx))
	}
	return out
}

func normalizeTask(task Task) Task {
	if task.ID == "" {
		task.ID = uuid.NewString()
	}
	if task.Name == "" {
		task.Name = "skill-eval"
	}
	if strings.TrimSpace(task.SystemPrompt) == "" {
		task.SystemPrompt = "你是评测中的智能体。请使用提供的 skill 完成用户任务，并把产物写在当前工作目录内。"
	}
	if task.MaxIterations <= 0 {
		task.MaxIterations = 10
	}
	if task.Timeout <= 0 {
		task.Timeout = 15 * time.Minute
	}
	if task.DataDir == "" {
		task.DataDir = dataDir()
	}
	return task
}

func failedTrace(task Task, kind Type, reason string) *agent.Trace {
	now := time.Now()
	status := "failed"
	if strings.Contains(reason, "超时") {
		status = "timeout"
	} else if strings.Contains(reason, "取消") {
		status = "cancelled"
	}
	return &agent.Trace{
		ID:            task.ID,
		AgentName:     task.Name,
		Model:         task.Model,
		UserPrompt:    task.UserPrompt,
		TargetSkill:   task.TargetSkill,
		StartTime:     now,
		EndTime:       now,
		Success:       false,
		Status:        status,
		FailureReason: reason,
		Error:         reason,
		SchemaVersion: 1,
		Runner:        string(kind),
		Spans:         []*agent.Span{},
		Steps:         []agent.Step{},
	}
}

func baseTrace(task Task, kind Type, start time.Time) *agent.Trace {
	return &agent.Trace{
		ID:            task.ID,
		AgentName:     task.Name,
		Model:         task.Model,
		UserPrompt:    task.UserPrompt,
		TargetSkill:   task.TargetSkill,
		StartTime:     start,
		SchemaVersion: 1,
		Runner:        string(kind),
		Status:        "running",
		Spans:         []*agent.Span{},
	}
}
