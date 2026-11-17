package runner

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"skill-eval/agent"
)

// CodexRunner 通过 Codex CLI 的 `codex exec --json` 执行 skill。
type CodexRunner struct {
	run commandFunc
}

func NewCodexRunner() *CodexRunner { return &CodexRunner{} }

func (c *CodexRunner) Type() Type { return TypeCodex }

func (c *CodexRunner) DisplayName() string { return "Codex" }

func (c *CodexRunner) Probe(ctx context.Context) Availability {
	av := Availability{Type: TypeCodex, Name: c.DisplayName()}
	bin, err := lookupBin("CODEX_BIN", "codex")
	if err != nil {
		av.Detail = err.Error()
		return av
	}
	av.Available = true
	av.Binary = bin
	av.Version = commandVersion(ctx, bin)
	av.Detail = "已找到 Codex CLI。认证沿用进程环境中的 OPENAI_API_KEY，或本机已有的 Codex 登录状态；密钥不会写入仓库。"
	return av
}

func (c *CodexRunner) Run(ctx context.Context, task Task) (*agent.Trace, error) {
	task = normalizeTask(task)
	bin, err := lookupBin("CODEX_BIN", "codex")
	if err != nil && c.run == nil {
		tr := failedTrace(task, TypeCodex, err.Error())
		return tr, err
	}
	if bin == "" {
		bin = "codex"
	}
	paths, err := newRunPaths(task)
	if err != nil {
		tr := failedTrace(task, TypeCodex, err.Error())
		return tr, err
	}
	if err := gitInit(paths.Workspace); err != nil && c.run == nil {
		tr := failedTrace(task, TypeCodex, err.Error())
		tr.Workspace = paths.Workspace
		return tr, err
	}
	version := ""
	if c.run == nil {
		version = commandVersion(ctx, bin)
	}
	return runExternal(ctx, c.run, externalSpec{
		Kind:      TypeCodex,
		Name:      c.DisplayName(),
		Version:   version,
		Task:      task,
		Paths:     paths,
		SkillDest: filepath.Join(paths.Workspace, ".agents", "skills"),
		Layout:    "Codex 会从工作区 .agents/skills/<name>/SKILL.md 发现 skill。",
		Build: func(task Task, paths runPaths, prompt string) commandSpec {
			args := []string{
				"exec",
				"--json",
				"--skip-git-repo-check",
				"--sandbox", envOr("CODEX_SANDBOX", "workspace-write"),
				"--ask-for-approval", envOr("CODEX_APPROVAL", "never"),
				"-C", paths.Workspace,
				"--output-last-message", filepath.Join(paths.LogDir, "last-message.txt"),
			}
			if task.Model != "" {
				args = append(args, "--model", task.Model)
			}
			if len(prompt) > 100000 {
				args = append(args, "请阅读并完成任务文件 "+paths.PromptPath)
				return commandSpec{Bin: bin, Args: args, Dir: paths.Workspace}
			}
			args = append(args, prompt)
			return commandSpec{Bin: bin, Args: args, Dir: paths.Workspace}
		},
		Parse: func(trace *agent.Trace, stdout string) externalParse {
			outcome := applyCodexOutput(trace, stdout, task.TargetSkill)
			if outcome.FinalText == "" {
				if raw, err := os.ReadFile(filepath.Join(paths.LogDir, "last-message.txt")); err == nil {
					outcome.FinalText = strings.TrimSpace(string(raw))
				}
			}
			ok := outcome.SawCompleted && !outcome.Failed
			return externalParse{
				OK: ok, Failed: outcome.Failed, Failure: outcome.Failure,
				FinalText: outcome.FinalText, Iterations: outcome.Iterations, Tokens: outcome.Tokens,
				ExternalID: outcome.ExternalID, Model: outcome.Model, Notes: outcome.Notes, Artifacts: outcome.Artifacts,
			}
		},
	})
}
