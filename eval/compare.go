package eval

import (
	"fmt"

	"skill-eval/agent"
)

// CompareReport 与前端对比视图的字段对齐。
type CompareReport struct {
	TraceA     *agent.Trace `json:"traceA"`
	TraceB     *agent.Trace `json:"traceB"`
	ReportA    *Report      `json:"reportA"`
	ReportB    *Report      `json:"reportB"`
	Scores     []Verdict    `json:"scores"`
	Conclusion string       `json:"conclusion"`
}

// Compare 用两份已经落盘的报告做可复核的对比，不额外调用大模型。
func Compare(traceA, traceB *agent.Trace, reportA, reportB *Report) (*CompareReport, error) {
	if traceA == nil || traceB == nil || reportA == nil || reportB == nil {
		return nil, fmt.Errorf("对比需要两份完整的 trace 和 report")
	}
	labelA := sideLabel("A", reportA)
	labelB := sideLabel("B", reportB)

	path := comparePath(labelA, labelB, reportA, reportB, traceA, traceB)
	artifact := compareArtifacts(labelA, labelB, reportA, reportB)
	scores := []Verdict{path, artifact}

	winner := "两者表现接近"
	switch {
	case reportA.Pass && !reportB.Pass:
		winner = labelA + " 通过评测，" + labelB + " 未通过"
	case reportB.Pass && !reportA.Pass:
		winner = labelB + " 通过评测，" + labelA + " 未通过"
	case reportA.Iterations < reportB.Iterations:
		winner = fmt.Sprintf("%s 迭代更少（%d 轮 vs %d 轮）", labelA, reportA.Iterations, reportB.Iterations)
	case reportB.Iterations < reportA.Iterations:
		winner = fmt.Sprintf("%s 迭代更少（%d 轮 vs %d 轮）", labelB, reportB.Iterations, reportA.Iterations)
	}

	conclusion := fmt.Sprintf("综合对比：%s。%s %s；%s %s。判定来自规则对比（source=deterministic），依据是两边 report 的通过状态、迭代次数、Token 和产物分。",
		winner, labelA, oneLine(reportA), labelB, oneLine(reportB))

	return &CompareReport{
		TraceA:     traceA,
		TraceB:     traceB,
		ReportA:    reportA,
		ReportB:    reportB,
		Scores:     scores,
		Conclusion: conclusion,
	}, nil
}

func sideLabel(side string, report *Report) string {
	runner := report.Runner
	if runner == "" {
		runner = "unknown"
	}
	model := report.Model
	if model == "" {
		model = "default"
	}
	return fmt.Sprintf("%s(%s/%s)", side, runner, model)
}

func oneLine(report *Report) string {
	status := "未通过"
	if report.Pass {
		status = "通过"
	}
	return fmt.Sprintf("%s，%d 轮，%d token", status, report.Iterations, report.TotalTokens)
}

func comparePath(labelA, labelB string, reportA, reportB *Report, traceA, traceB *agent.Trace) Verdict {
	errA, errB := toolErrors(traceA), toolErrors(traceB)
	better := "路径成本接近"
	pass := true
	score := 0.8
	if reportA.Iterations > reportB.Iterations*2 || errA > errB+2 {
		better = labelB + " 的执行路径更短或工具失败更少"
		pass = false
		score = 0.45
	} else if reportB.Iterations > reportA.Iterations*2 || errB > errA+2 {
		better = labelA + " 的执行路径更短或工具失败更少"
		pass = false
		score = 0.45
	}
	reason := fmt.Sprintf("%s：%d 轮 / %d token / 工具错误 %d。%s：%d 轮 / %d token / 工具错误 %d。%s。",
		labelA, reportA.Iterations, reportA.TotalTokens, errA,
		labelB, reportB.Iterations, reportB.TotalTokens, errB, better)
	return Verdict{
		Info:   ScoreItem{Name: "执行路径对比", Desc: "对比迭代、Token 和工具错误"},
		Pass:   pass,
		Score:  score,
		Reason: reason,
		Source: "deterministic",
	}
}

func compareArtifacts(labelA, labelB string, reportA, reportB *Report) Verdict {
	sa, ra := scoreByName(reportA, "产物评分")
	sb, rb := scoreByName(reportB, "产物评分")
	if ra == "" && rb == "" {
		sa, ra = boolScore(reportA.Pass), "无产物评分，改用总评通过状态"
		sb, rb = boolScore(reportB.Pass), "无产物评分，改用总评通过状态"
	}
	score := (sa + sb) / 2
	return Verdict{
		Info:   ScoreItem{Name: "产物质量对比", Desc: "对比两边产物评分或总评"},
		Pass:   sa >= 0.6 && sb >= 0.6,
		Score:  score,
		Reason: fmt.Sprintf("%s 产物 %.2f（%s）；%s 产物 %.2f（%s）", labelA, sa, ra, labelB, sb, rb),
		Source: "deterministic",
	}
}

func scoreByName(report *Report, name string) (float64, string) {
	for _, s := range report.Scores {
		if s.Info.Name == name {
			return s.Score, s.Reason
		}
	}
	return 0, ""
}

func boolScore(pass bool) float64 {
	if pass {
		return 1
	}
	return 0
}

func toolErrors(trace *agent.Trace) int {
	if trace == nil {
		return 0
	}
	n := 0
	for _, span := range trace.Spans {
		if span != nil && span.Kind == agent.SpanKindToolCall && span.Error != "" {
			n++
		}
	}
	return n
}
