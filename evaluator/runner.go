package evaluator

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"skill-eval/providers"
)

type Runner struct {
	Config EvalConfig
}

func NewRunner(cfg EvalConfig) *Runner {
	if cfg.MaxRounds <= 0 {
		cfg.MaxRounds = 8
	}
	if cfg.DockerImage == "" {
		cfg.DockerImage = "golang:1.25"
	}
	return &Runner{Config: cfg}
}

func (r *Runner) RunSingle(ctx context.Context, skillName string, suite EvalSuite) (EvalReport, error) {
	cfg := r.Config
	r.publish(cfg, "task_started", map[string]any{
		"mode":       "single",
		"skill_name": skillName,
		"suite_id":   suite.SuiteID,
	})
	workspaceDir := ""
	if cfg.UseDocker {
		isoDir, err := prepareIsolatedWorkspace(cfg, skillName)
		if err != nil {
			return EvalReport{}, fmt.Errorf("prepare isolated workspace failed: %w", err)
		}
		workspaceDir = isoDir
		cfg.ProjectRootDir = isoDir
		defer func() {
			_ = cleanupWorkspace(isoDir, cfg.KeepWorkDir)
		}()
	}

	report := EvalReport{
		Mode:          "single",
		SuiteID:       suite.SuiteID,
		SkillName:     skillName,
		GroupResults:  make([]GroupResult, 0, len(suite.Groups)),
		GeneratedAt:   time.Now().Format(time.RFC3339),
		IsolationMode: "docker",
		WorkspaceDir:  workspaceDir,
		WorkDirKept:   cfg.KeepWorkDir,
		DockerHealth: DockerHealth{
			Enabled:   cfg.UseDocker,
			Available: false,
			CheckedAt: time.Now().Format(time.RFC3339),
		},
		FallbackByReason: map[string]int{},
	}

	var totalScore float64
	var dockerExec *DockerExecutor
	if cfg.UseDocker {
		dockerExec = NewDockerExecutor(cfg)
		report.DockerHealth = dockerExec.HealthCheck(ctx)
		r.publish(cfg, "docker_health_checked", map[string]any{
			"available": report.DockerHealth.Available,
			"error":     report.DockerHealth.Error,
		})
	}
	for _, group := range suite.Groups {
		r.publish(cfg, "group_started", map[string]any{
			"group_id": group.GroupID,
		})
		groupResult := GroupResult{
			GroupID: group.GroupID,
			Results: make([]CaseResult, 0, len(group.Cases)),
		}

		for _, tc := range group.Cases {
			r.publish(cfg, "case_started", map[string]any{
				"group_id": group.GroupID,
				"case_id":  tc.ID,
				"input":    tc.Input,
			})
			run := providers.RunResult{}
			var err error
			runnerName := "local"
			containerName := ""
			image := ""
			debugLog := ""
			liveToolCallEmitted := false

			if dockerExec != nil {
				if report.DockerHealth.Available {
					runnerName = "docker"
					run, containerName, image, debugLog, err = dockerExec.RunCase(ctx, tc.Input, r.Config.MaxRounds)
				} else {
					err = fmt.Errorf("docker unavailable: %s", report.DockerHealth.Error)
				}
				if err != nil {
					// Docker 失败时自动降级到本地执行，避免整批中断。
					runnerName = "docker-fallback-local"
					report.FallbackCount++
					reason := normalizeFallbackReason(err)
					report.FallbackByReason[reason]++
					r.publish(cfg, "fallback_triggered", map[string]any{
						"group_id": group.GroupID,
						"case_id":  tc.ID,
						"reason":   reason,
						"error":    err.Error(),
					})
					localRun, localErr := providers.ExecutePrompt(ctx, tc.Input, r.Config.MaxRounds)
					if localErr == nil {
						run = localRun
						err = nil
					}
				}
			} else {
				liveToolCallEmitted = true
				run, err = providers.ExecutePromptWithHooks(ctx, tc.Input, r.Config.MaxRounds, &providers.RunHooks{
					OnRoundStart: func(round int) {
						r.publish(cfg, "llm_round_started", map[string]any{
							"group_id": group.GroupID,
							"case_id":  tc.ID,
							"round":    round,
						})
					},
					OnRoundStep: func(step providers.RunStep) {
						r.publish(cfg, "llm_round_finished", map[string]any{
							"group_id":      group.GroupID,
							"case_id":       tc.ID,
							"round":         step.Round,
							"tool_calls":    len(step.ToolCalls),
							"assistant_raw": step.AssistantRaw,
						})
					},
					OnToolCall: func(round int, call providers.ToolCallRecord) {
						r.publish(cfg, "tool_call", map[string]any{
							"group_id": group.GroupID,
							"case_id":  tc.ID,
							"round":    round,
							"name":     call.Name,
							"args":     call.Args,
							"result":   call.Result,
							"success":  call.Success,
							"error":    call.Error,
						})
					},
					OnMessage: func(round int, role string, content string, meta map[string]any) {
						r.publish(cfg, "message", map[string]any{
							"group_id": group.GroupID,
							"case_id":  tc.ID,
							"round":    round,
							"role":     role,
							"content":  content,
							"meta":     meta,
						})
					},
				})
			}

			if err != nil && run.Error == "" {
				run.Error = err.Error()
			}

			score := scoreBySuccess(run.Success)
			caseResult := CaseResult{
				Case:       tc,
				Run:        run,
				AIEval:     defaultAIEval(run.Success, tc.Expectation),
				HumanCheck: "pending",
				Score:      score,
				Runner:     runnerName,
				Container:  containerName,
				Image:      image,
				DebugLog:   debugLog,
			}
			groupResult.Results = append(groupResult.Results, caseResult)
			if !liveToolCallEmitted {
				r.emitToolCallsFromRun(cfg, group.GroupID, tc.ID, run)
				r.emitMessagesFromRun(cfg, group.GroupID, tc.ID, run)
			}
			r.publish(cfg, "case_finished", map[string]any{
				"group_id": group.GroupID,
				"case_id":  tc.ID,
				"success":  run.Success,
				"runner":   runnerName,
				"rounds":   run.TotalRounds,
				"tokens":   run.TotalTokens,
			})
			report.TotalCases++
			report.TotalTokens += run.TotalTokens
			report.TotalRounds += run.TotalRounds
			totalScore += score
			if run.Success {
				report.SuccessCases++
			} else {
				report.FailedCases++
			}
		}

		report.GroupResults = append(report.GroupResults, groupResult)
		r.publish(cfg, "group_finished", map[string]any{
			"group_id": group.GroupID,
		})
	}

	if report.TotalCases > 0 {
		report.SuccessRate = float64(report.SuccessCases) / float64(report.TotalCases)
		report.AverageScore = totalScore / float64(report.TotalCases)
	}

	r.publish(cfg, "task_finished", map[string]any{
		"mode":          "single",
		"skill_name":    skillName,
		"success_rate":  report.SuccessRate,
		"average_score": report.AverageScore,
		"total_cases":   report.TotalCases,
	})
	return report, nil
}

func (r *Runner) RunCompare(ctx context.Context, leftSkill string, rightSkill string, suite EvalSuite) (CompareReport, error) {
	r.publish(r.Config, "task_started", map[string]any{
		"mode":        "compare",
		"left_skill":  leftSkill,
		"right_skill": rightSkill,
		"suite_id":    suite.SuiteID,
	})
	var left EvalReport
	var right EvalReport
	var leftErr error
	var rightErr error

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		left, leftErr = r.RunSingle(ctx, leftSkill, suite)
	}()
	go func() {
		defer wg.Done()
		right, rightErr = r.RunSingle(ctx, rightSkill, suite)
	}()
	wg.Wait()

	if leftErr != nil {
		return CompareReport{}, fmt.Errorf("left skill run failed: %w", leftErr)
	}
	if rightErr != nil {
		return CompareReport{}, fmt.Errorf("right skill run failed: %w", rightErr)
	}

	winner := "draw"
	switch {
	case left.AverageScore > right.AverageScore:
		winner = leftSkill
	case right.AverageScore > left.AverageScore:
		winner = rightSkill
	}

	report := CompareReport{
		Mode:          "compare",
		SuiteID:       suite.SuiteID,
		LeftSkill:     leftSkill,
		RightSkill:    rightSkill,
		Left:          left,
		Right:         right,
		Winner:        winner,
		GeneratedAt:   time.Now().Format(time.RFC3339),
		IsolationMode: "docker",
		Notes: []string{
			"compare run executed concurrently",
			"each case prefers isolated docker execution; local fallback is enabled",
		},
		WorkspaceDirs:      []string{left.WorkspaceDir, right.WorkspaceDir},
		DockerHealth:       chooseDockerHealth(left.DockerHealth, right.DockerHealth),
		TotalFallbackCount: left.FallbackCount + right.FallbackCount,
		FallbackByReason:   mergeFallbackReasons(left.FallbackByReason, right.FallbackByReason),
	}
	r.publish(r.Config, "task_finished", map[string]any{
		"mode":                 "compare",
		"winner":               report.Winner,
		"total_fallback_count": report.TotalFallbackCount,
	})
	return report, nil
}

func chooseDockerHealth(left DockerHealth, right DockerHealth) DockerHealth {
	if left.Available {
		return left
	}
	if right.Available {
		return right
	}
	if left.Enabled {
		return left
	}
	return right
}

func (r *Runner) publish(cfg EvalConfig, eventType string, data map[string]any) {
	if cfg.Publisher == nil || cfg.TaskID == "" {
		return
	}
	cfg.Publisher.Publish(EvalEvent{
		TaskID: cfg.TaskID,
		Type:   eventType,
		Time:   time.Now().Format(time.RFC3339),
		Data:   data,
	})
}

func (r *Runner) emitToolCallsFromRun(cfg EvalConfig, groupID string, caseID string, run providers.RunResult) {
	for _, step := range run.Steps {
		for _, call := range step.ToolCalls {
			r.publish(cfg, "tool_call", map[string]any{
				"group_id": groupID,
				"case_id":  caseID,
				"round":    step.Round,
				"name":     call.Name,
				"args":     call.Args,
				"result":   call.Result,
				"success":  call.Success,
				"error":    call.Error,
			})
		}
	}
}

func (r *Runner) emitMessagesFromRun(cfg EvalConfig, groupID string, caseID string, run providers.RunResult) {
	r.publish(cfg, "message", map[string]any{
		"group_id": groupID,
		"case_id":  caseID,
		"round":    0,
		"role":     "user",
		"content":  run.Input,
		"meta":     map[string]any{},
	})

	for _, step := range run.Steps {
		r.publish(cfg, "message", map[string]any{
			"group_id": groupID,
			"case_id":  caseID,
			"round":    step.Round,
			"role":     "assistant",
			"content":  "",
			"meta": map[string]any{
				"assistant_raw": step.AssistantRaw,
			},
		})
		for _, call := range step.ToolCalls {
			r.publish(cfg, "message", map[string]any{
				"group_id": groupID,
				"case_id":  caseID,
				"round":    step.Round,
				"role":     "tool",
				"content":  call.Result,
				"meta": map[string]any{
					"name":    call.Name,
					"success": call.Success,
					"error":   call.Error,
					"args":    call.Args,
				},
			})
		}
	}
}

func normalizeFallbackReason(err error) string {
	if err == nil {
		return "unknown"
	}
	msg := err.Error()
	switch {
	case strings.Contains(msg, "docker unavailable"):
		return "docker_unavailable"
	case strings.Contains(msg, "pull image"):
		return "image_pull_failed"
	case strings.Contains(msg, "create container"):
		return "container_create_failed"
	case strings.Contains(msg, "start container"):
		return "container_start_failed"
	case strings.Contains(msg, "wait container"):
		return "container_wait_failed"
	case strings.Contains(msg, "exited with code"):
		return "container_non_zero_exit"
	case strings.Contains(msg, "parse docker output"):
		return "result_parse_failed"
	default:
		return "docker_runtime_error"
	}
}

func mergeFallbackReasons(left map[string]int, right map[string]int) map[string]int {
	out := map[string]int{}
	for k, v := range left {
		out[k] += v
	}
	for k, v := range right {
		out[k] += v
	}
	return out
}

func SaveReport(path string, report any) error {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal report failed: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create report dir failed: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write report failed: %w", err)
	}
	return nil
}

func scoreBySuccess(success bool) float64 {
	if success {
		return 1.0
	}
	return 0.0
}

func defaultAIEval(success bool, expectation string) string {
	if success {
		return "AI初判通过：任务完成，建议人工确认输出质量是否满足期望"
	}
	return "AI初判失败：任务未完成或执行异常，建议人工确认失败原因与重试价值"
}
