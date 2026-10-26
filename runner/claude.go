package runner

import (
	"context"
	"path/filepath"

	"skill-eval/agent"
)

// ClaudeCodeRunner 通过 Claude Code CLI 的 headless JSON 流执行 skill。
type ClaudeCodeRunner struct {
	run commandFunc
}

func NewClaudeCodeRunner() *ClaudeCodeRunner { return &ClaudeCodeRunner{} }

func (c *ClaudeCodeRunner) Type() Type { return TypeClaudeCode }

func (c *ClaudeCodeRunner) DisplayName() string { return "Claude Code" }

func (c *ClaudeCodeRunner) Probe(ctx context.Context) Availability {
	av := Availability{Type: TypeClaudeCode, Name: c.DisplayName()}
	bin, err := lookupBin("CLAUDE_BIN", "claude")
	if err != nil {
		av.Detail = err.Error()
		return av
	}
	av.Available = true
	av.Binary = bin
	av.Version = commandVersion(ctx, bin)
	av.Detail = "已找到 Claude Code CLI。认证沿用进程环境中的 ANTHROPIC_API_KEY，或本机已有的 Claude 登录状态；密钥不会写入仓库。"
	return av
}

func (c *ClaudeCodeRunner) Run(ctx context.Context, task Task) (*agent.Trace, error) {
	task = normalizeTask(task)
	bin, err := lookupBin("CLAUDE_BIN", "claude")
	if err != nil && c.run == nil {
		return failedTrace(task, TypeClaudeCode, err.Error()), err
	}
	if bin == "" {
		bin = "claude"
	}
	paths, err := newRunPaths(task)
	if err != nil {
		return failedTrace(task, TypeClaudeCode, err.Error()), err
	}
	version := ""
	if c.run == nil {
		version = commandVersion(ctx, bin)
	}
	return runExternal(ctx, c.run, externalSpec{
		Kind:      TypeClaudeCode,
		Name:      c.DisplayName(),
		Version:   version,
		Task:      task,
		Paths:     paths,
		SkillDest: filepath.Join(paths.Workspace, ".claude", "skills"),
		Layout:    "Claude Code 会从工作区 .claude/skills/<name>/SKILL.md 加载 skill。",
		ExtraEnv:  []string{"CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC=1"},
		Build: func(task Task, paths runPaths, prompt string) commandSpec {
			args := []string{
				"-p",
				"--output-format", "stream-json",
				"--verbose",
				"--permission-mode", envOr("CLAUDE_PERMISSION_MODE", "bypassPermissions"),
				"--no-session-persistence",
				"--add-dir", paths.Workspace,
			}
			if task.Model != "" {
				args = append(args, "--model", task.Model)
			}
			return commandSpec{
				Bin:   bin,
				Args:  args,
				Dir:   paths.Workspace,
				Stdin: prompt,
			}
		},
		Parse: func(trace *agent.Trace, stdout string) externalParse {
			outcome := applyClaudeOutput(trace, stdout, task.TargetSkill)
			return externalParse{
				OK:         outcome.OK && !outcome.Failed,
				Failed:     outcome.Failed,
				Failure:    outcome.Failure,
				FinalText:  outcome.FinalText,
				Iterations: outcome.Iterations,
				Tokens:     outcome.Tokens,
				ExternalID: outcome.ExternalID,
				Model:      outcome.Model,
				Notes:      outcome.Notes,
				Artifacts:  outcome.Artifacts,
			}
		},
	})
}
