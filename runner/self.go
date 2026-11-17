package runner

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"skill-eval/agent"
	"skill-eval/providers"
)

// SelfRunner 使用仓库里的自研 agent loop。
type SelfRunner struct {
	completer agent.Completer
}

func NewSelfRunner() *SelfRunner { return &SelfRunner{} }

// UseCompleter 注入对话客户端。测试用它代替真实的模型服务。
func (s *SelfRunner) UseCompleter(c agent.Completer) {
	s.completer = c
}

func (s *SelfRunner) Type() Type { return TypeSelf }

func (s *SelfRunner) DisplayName() string { return "自研内核" }

func (s *SelfRunner) Probe(ctx context.Context) Availability {
	av := Availability{Type: TypeSelf, Name: s.DisplayName()}
	if s.completer != nil {
		av.Available = true
		av.Detail = "使用注入的对话客户端。"
		return av
	}
	var missing []string
	if os.Getenv("EVAL_API_KEY") == "" {
		missing = append(missing, "EVAL_API_KEY")
	}
	if os.Getenv("EVAL_BASE_URL") == "" {
		missing = append(missing, "EVAL_BASE_URL")
	}
	if len(missing) > 0 {
		av.Detail = "自研内核已内置，但缺少环境变量: " + joinComma(missing) + "。参见 docs/runners.md"
		return av
	}
	av.Available = true
	av.Detail = "自研内核可用。模型调用使用 EVAL_BASE_URL 与 EVAL_API_KEY。"
	return av
}

func (s *SelfRunner) Run(ctx context.Context, task Task) (*agent.Trace, error) {
	task = normalizeTask(task)
	paths, err := newRunPaths(task)
	if err != nil {
		tr := failedTrace(task, TypeSelf, err.Error())
		return tr, err
	}
	comp, err := s.resolveCompleter()
	if err != nil {
		tr := failedTrace(task, TypeSelf, err.Error())
		tr.Workspace = paths.Workspace
		return tr, err
	}

	skillRoot := ""
	skillPath := ""
	if task.SkillDir != "" {
		skillRoot = filepath.Join(paths.Workspace, ".skill-eval", "skills")
		if err := stageSkills(task.SkillDir, skillRoot); err != nil {
			tr := failedTrace(task, TypeSelf, err.Error())
			tr.Workspace = paths.Workspace
			return tr, err
		}
		skillPath = findSkillFile(skillRoot, task.TargetSkill)
	}

	cfg := agent.NewAgentConfig(
		agent.WithName(task.Name),
		agent.WithDescription(task.Description),
		agent.WithSystemPrompt(task.SystemPrompt+" 完成任务时必须调用 finish 工具。"),
		agent.WithUserPrompt(task.UserPrompt),
		agent.WithModel(task.Model),
		agent.WithMaxIterations(task.MaxIterations),
		agent.WithWorkspace(paths.Workspace),
	)
	cfg.RegistryDefaultTools()
	if skillRoot != "" {
		cfg.RegistrySkillsFrom(skillRoot)
	}

	orch := agent.NewOrchestrator(comp, cfg)
	tracer := agent.NewDefaultTracer(paths.LogDir)
	tracer.SetID(task.ID)
	orch.Tracer = tracer
	orch.SetTargetSkill(task.TargetSkill)

	runCtx, cancel := context.WithTimeout(ctx, task.Timeout)
	defer cancel()
	runErr := orch.RunContext(runCtx)
	trace := orch.GetTrace()
	if trace == nil {
		reason := "自研内核没有产生 trace"
		if runErr != nil {
			reason = runErr.Error()
		}
		trace = failedTrace(task, TypeSelf, reason)
	}
	trace.Runner = string(TypeSelf)
	trace.Workspace = paths.Workspace
	trace.SkillPath = skillPath
	trace.RunnerMeta = &agent.RunnerMeta{
		Type:        string(TypeSelf),
		DisplayName: s.DisplayName(),
		Model:       task.Model,
		Isolated:    true,
		SkillDir:    skillRoot,
		LogPath:     paths.StdoutLog,
	}
	tracer.Save()
	if runErr != nil && trace.FailureReason == "" {
		trace.FailureReason = runErr.Error()
		trace.Error = runErr.Error()
		if trace.Status == "" || trace.Status == "running" {
			trace.Status = "failed"
		}
	}
	return trace, runErr
}

func (s *SelfRunner) resolveCompleter() (agent.Completer, error) {
	if s.completer != nil {
		return s.completer, nil
	}
	if strings.TrimSpace(os.Getenv("EVAL_API_KEY")) == "" || strings.TrimSpace(os.Getenv("EVAL_BASE_URL")) == "" {
		return nil, fmt.Errorf("self runner 需要环境变量 EVAL_API_KEY 与 EVAL_BASE_URL。参见 docs/runners.md")
	}
	return agent.NewOpenAICompleter(providers.NewClient()), nil
}

func joinComma(items []string) string {
	out := ""
	for i, item := range items {
		if i > 0 {
			out += ", "
		}
		out += item
	}
	return out
}
