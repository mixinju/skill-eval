package runner

import (
	"context"
	"fmt"
	"strings"
	"time"

	"skill-eval/agent"
)

type externalParse struct {
	OK         bool
	Failed     bool
	Failure    string
	FinalText  string
	Iterations int
	Tokens     int64
	ExternalID string
	Model      string
	Notes      []string
	Artifacts  []string
}

type externalSpec struct {
	Kind      Type
	Name      string
	Version   string
	Task      Task
	Paths     runPaths
	SkillDest string
	Layout    string
	ExtraEnv  []string
	Build     func(task Task, paths runPaths, prompt string) commandSpec
	Parse     func(trace *agent.Trace, stdout string) externalParse
}

func runExternal(ctx context.Context, run commandFunc, spec externalSpec) (*agent.Trace, error) {
	if run == nil {
		run = realCommand
	}
	task := spec.Task
	paths := spec.Paths
	start := time.Now()
	trace := baseTrace(task, spec.Kind, start)

	skillPath := ""
	if task.SkillDir != "" {
		if err := stageSkills(task.SkillDir, spec.SkillDest); err != nil {
			tr := failedTrace(task, spec.Kind, err.Error())
			tr.Workspace = paths.Workspace
			return tr, err
		}
		skillPath = findSkillFile(spec.SkillDest, task.TargetSkill)
	}

	prompt := externalPrompt(task, skillPath, spec.Layout)
	if err := writePromptFile(paths.PromptPath, prompt); err != nil {
		tr := failedTrace(task, spec.Kind, err.Error())
		tr.Workspace = paths.Workspace
		return tr, err
	}
	before := snapshot(paths.Workspace)
	cmd := spec.Build(task, paths, prompt)
	cmd.ExtraEnv = append(cmd.ExtraEnv, spec.ExtraEnv...)

	runCtx, cancel := context.WithTimeout(ctx, task.Timeout)
	defer cancel()
	stdout, stderr, exitCode, runErr := run(runCtx, cmd)
	writeLogs(paths, stdout, stderr)

	parsed := spec.Parse(trace, stdout)
	artifacts := mergeArtifacts(parsed.Artifacts, artifactsFromText(parsed.FinalText), newFiles(paths.Workspace, before))

	trace.EndTime = time.Now()
	trace.Iterations = parsed.Iterations
	trace.TotalTokens = parsed.Tokens
	trace.ExternalID = parsed.ExternalID
	trace.Notes = append([]string{}, parsed.Notes...)
	trace.Workspace = paths.Workspace
	trace.SkillPath = skillPath
	if trace.Model == "" && parsed.Model != "" {
		trace.Model = parsed.Model
	}
	trace.ArtifactsAndResult = finishPayload(parsed.FinalText, artifacts)
	trace.RunnerMeta = &agent.RunnerMeta{
		Type:        string(spec.Kind),
		DisplayName: spec.Name,
		Binary:      cmd.Bin,
		Version:     spec.Version,
		Model:       trace.Model,
		Isolated:    true,
		SkillDir:    spec.SkillDest,
		Command:     commandForMeta(cmd.Bin, cmd.Args, prompt),
		LogPath:     paths.StdoutLog,
	}
	if trace.Spans == nil {
		trace.Spans = []*agent.Span{}
	}
	trace.Steps = agent.BuildSteps(trace.Spans)
	if note := strings.TrimSpace(stderr); note != "" {
		trace.Notes = append(trace.Notes, "stderr: "+clipLine(note))
	}

	switch {
	case runErr != nil && strings.Contains(runErr.Error(), "执行超时"):
		trace.Success = false
		trace.Status = "timeout"
		trace.FailureReason = "执行超时"
		trace.Error = trace.FailureReason
		return trace, fmt.Errorf("%s", trace.FailureReason)
	case runErr != nil && strings.Contains(runErr.Error(), "任务已取消"):
		trace.Success = false
		trace.Status = "cancelled"
		trace.FailureReason = "任务已取消"
		trace.Error = trace.FailureReason
		return trace, fmt.Errorf("%s", trace.FailureReason)
	case parsed.Failed:
		trace.Success = false
		trace.Status = "failed"
		reason := parsed.Failure
		if reason == "" {
			reason = "执行失败"
		}
		trace.FailureReason = reason
		trace.Error = reason
		return trace, nil
	case parsed.OK:
		trace.Success = true
		trace.Status = "succeeded"
		if exitCode != 0 {
			trace.Notes = append(trace.Notes, fmt.Sprintf("进程退出码为 %d，事件流表明执行已完成", exitCode))
		}
		return trace, nil
	default:
		reason := "执行结束但没有成功事件"
		if runErr != nil {
			reason = runErr.Error()
		}
		trace.Success = false
		trace.Status = "failed"
		trace.FailureReason = reason
		trace.Error = reason
		if runErr != nil {
			return trace, fmt.Errorf("%s", reason)
		}
		return trace, fmt.Errorf("%s", reason)
	}
}

func commandForMeta(bin string, args []string, prompt string) string {
	shown := append([]string{}, args...)
	if len(shown) > 0 && shown[len(shown)-1] == prompt {
		shown[len(shown)-1] = "<prompt>"
	}
	text := bin + " " + strings.Join(shown, " ")
	if len(text) > 500 {
		return text[:500] + "…"
	}
	return text
}
