package eval

import (
	"fmt"

	"skill-eval/agent"

	"github.com/sirupsen/logrus"
)

type Report struct {
	TraceID     string    `json:"traceId"`
	AgentName   string    `json:"agentName"`
	Model       string    `json:"model"`
	UserPrompt  string    `json:"userPrompt"`
	TargetSkill string    `json:"targetSkill"`
	TotalTokens int64     `json:"totalTokens"`
	Iterations  int       `json:"iterations"`
	Duration    int64     `json:"duration"`
	Scores      []Verdict `json:"scores"`
	Pass        bool      `json:"pass"`
}

func Exec(trace *agent.Trace, scorers []Scorer) *Report {
	report := &Report{
		TraceID:     trace.ID,
		AgentName:   trace.AgentName,
		Model:       trace.Model,
		UserPrompt:  trace.UserPrompt,
		TargetSkill: trace.TargetSkill,
		TotalTokens: trace.TotalTokens,
		Iterations:  trace.Iterations,
		Duration:    trace.EndTime.Sub(trace.StartTime).Milliseconds(),
		Pass:        true,
	}

	for _, scorer := range scorers {
		logrus.Infof("[Eval] 执行评分器: %s", scorer.Item())
		result, _ := scorer.Score(trace)
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
	fmt.Printf("Tokens:      %d\n", r.TotalTokens)
	fmt.Printf("Iterations:  %d\n", r.Iterations)
	fmt.Printf("Duration:    %dms\n", r.Duration)
	fmt.Println("--------- 评分详情 ----------")
	for _, s := range r.Scores {
		status := "PASS"
		if !s.Pass {
			status = "FAIL"
		}
		fmt.Printf("  [%s] %-12s %.1f  %s\n", status, s.Info, s.Score, s.Reason)
	}
	fmt.Println("-----------------------------")
	if r.Pass {
		fmt.Println("Result:      PASS")
	} else {
		fmt.Println("Result:      FAIL")
	}
	fmt.Println("==============================")
}
