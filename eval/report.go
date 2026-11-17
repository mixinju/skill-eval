package eval

import (
	"fmt"

	"skill-eval/agent"

	"github.com/sirupsen/logrus"
)

type Report struct {
	TraceID       string            `json:"traceId"`
	AgentName     string            `json:"agentName"`
	Model         string            `json:"model"`
	UserPrompt    string            `json:"userPrompt"`
	TargetSkill   string            `json:"targetSkill"`
	TotalTokens   int64             `json:"totalTokens"`
	Iterations    int               `json:"iterations"`
	Duration      int64             `json:"duration"`
	Scores        []Verdict         `json:"scores"`
	Pass          bool              `json:"pass"`
	Runner        string            `json:"runner,omitempty"`
	RunnerMeta    *agent.RunnerMeta `json:"runnerMeta,omitempty"`
	Status        string            `json:"status,omitempty"`
	FailureReason string            `json:"failureReason,omitempty"`
	Workspace     string            `json:"workspace,omitempty"`
	SkillPath     string            `json:"skillPath,omitempty"`
}

func Exec(trace *agent.Trace, scorers []Scorer) *Report {
	if trace == nil {
		return &Report{
			Pass:   false,
			Status: "failed",
			Scores: []Verdict{{
				Info:   ScoreItem{Name: "trace", Desc: "trace 缺失"},
				Pass:   false,
				Reason: "trace 为空，无法评分",
				Source: "deterministic",
				Error:  "trace 为空",
			}},
			FailureReason: "trace 为空",
		}
	}
	report := &Report{
		TraceID:       trace.ID,
		AgentName:     trace.AgentName,
		Model:         trace.Model,
		UserPrompt:    trace.UserPrompt,
		TargetSkill:   trace.TargetSkill,
		TotalTokens:   trace.TotalTokens,
		Iterations:    trace.Iterations,
		Duration:      trace.EndTime.Sub(trace.StartTime).Milliseconds(),
		Pass:          true,
		Scores:        []Verdict{},
		Runner:        trace.Runner,
		RunnerMeta:    trace.RunnerMeta,
		Status:        trace.Status,
		FailureReason: trace.FailureReason,
		Workspace:     trace.Workspace,
		SkillPath:     trace.SkillPath,
	}

	for _, scorer := range scorers {
		item := scorer.Item()
		logrus.Infof("[Eval] 执行评分器: %s", item.Name)
		result, err := scorer.Score(trace)
		if result.Info.Name == "" {
			result.Info = item
		}
		if err != nil {
			result.Pass = false
			result.Error = err.Error()
			if result.Reason == "" {
				result.Reason = "评分器执行失败: " + err.Error()
			}
			if result.Source == "" {
				result.Source = "deterministic"
			}
		}
		report.Scores = append(report.Scores, result)
		if !result.Pass {
			report.Pass = false
		}
	}

	return report
}

func (r *Report) Print() {
	fmt.Println("========== 评测报告 ==========")
	fmt.Printf("Trace ID:    %s\n", r.TraceID)
	fmt.Printf("Agent:       %s\n", r.AgentName)
	fmt.Printf("Model:       %s\n", r.Model)
	fmt.Printf("Prompt:      %s\n", r.UserPrompt)
	fmt.Printf("Target Skill:%s\n", r.TargetSkill)
	if r.Runner != "" {
		fmt.Printf("Runner:      %s\n", r.Runner)
	}
	if r.Status != "" {
		fmt.Printf("Status:      %s\n", r.Status)
	}
	if r.FailureReason != "" {
		fmt.Printf("Failure:     %s\n", r.FailureReason)
	}
	fmt.Printf("Tokens:      %d\n", r.TotalTokens)
	fmt.Printf("Iterations:  %d\n", r.Iterations)
	fmt.Printf("Duration:    %dms\n", r.Duration)
	fmt.Println("--------- 评分详情 ----------")
	for _, s := range r.Scores {
		status := "PASS"
		if !s.Pass {
			status = "FAIL"
		}
		fmt.Printf("  [%s] %-16s %.1f  %s", status, s.Info.Name, s.Score, s.Reason)
		if s.Source != "" {
			fmt.Printf("  (source=%s)", s.Source)
		}
		fmt.Println()
	}
	fmt.Println("-----------------------------")
	if r.Pass {
		fmt.Println("Result:      PASS")
	} else {
		fmt.Println("Result:      FAIL")
	}
	fmt.Println("==============================")
}
