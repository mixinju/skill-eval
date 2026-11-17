package eval

import (
	"fmt"
	"strings"

	"skill-eval/agent"
)

// DeterministicProcessScorer 用 trace 上的迭代、工具错误和技能命中做规则评分。
// 它不调用大模型，同一份 trace 可以复现出相同分数。
type DeterministicProcessScorer struct{}

func NewDeterministicProcessScorer() *DeterministicProcessScorer {
	return &DeterministicProcessScorer{}
}

func (s *DeterministicProcessScorer) Item() ScoreItem {
	return ScoreItem{
		Name: "执行过程(规则)",
		Desc: "根据迭代次数、工具错误率和技能命中做可复现评分",
	}
}

func (s *DeterministicProcessScorer) Score(traces ...*agent.Trace) (Verdict, error) {
	first, second, err := extraTrace(traces...)
	if err != nil {
		return Verdict{}, err
	}
	if second == nil {
		return s.single(first), nil
	}
	a := s.single(first)
	b := s.single(second)
	return Verdict{
		Info:   s.Item(),
		Pass:   a.Pass && b.Pass,
		Score:  (a.Score + b.Score) / 2,
		Reason: fmt.Sprintf("A %.2f (%s)；B %.2f (%s)", a.Score, a.Reason, b.Score, b.Reason),
		Source: "deterministic",
	}, nil
}

func (s *DeterministicProcessScorer) single(trace *agent.Trace) Verdict {
	if trace == nil {
		return Verdict{Info: s.Item(), Pass: false, Score: 0, Reason: "trace 为空", Source: "deterministic", Error: "trace 为空"}
	}

	eff := 1.0
	switch {
	case trace.Iterations > 8:
		eff = 0.4
	case trace.Iterations > 4:
		eff = 0.7
	case trace.Iterations == 0:
		eff = 0
	}

	toolTotal, toolErr := 0, 0
	for _, span := range trace.Spans {
		if span == nil || span.Kind != agent.SpanKindToolCall {
			continue
		}
		toolTotal++
		if span.Error != "" {
			toolErr++
		}
	}
	toolScore := 1.0
	if toolTotal > 0 {
		toolScore = 1 - float64(toolErr)/float64(toolTotal)
	}

	skillScore := 1.0
	skillNote := "未设置目标 skill"
	if trace.TargetSkill != "" {
		hit, _ := NewSkillHitScorer().single(trace)
		if hit.Pass {
			skillScore = 1
			skillNote = "命中目标 skill"
		} else {
			skillScore = 0
			skillNote = "未命中目标 skill"
		}
	}

	succ := 0.0
	succNote := "执行未完成"
	if trace.Success {
		succ = 1
		succNote = "执行完成"
	}
	score := (eff + toolScore + skillScore + succ) / 4
	reason := fmt.Sprintf("效率%.2f（%d轮）、工具%.2f（%d/%d 失败）、技能%.2f（%s）、完成%.2f（%s）",
		eff, trace.Iterations, toolScore, toolErr, toolTotal, skillScore, skillNote, succ, succNote)
	return Verdict{
		Info:   s.Item(),
		Pass:   score >= 0.6,
		Score:  score,
		Reason: strings.TrimSpace(reason),
		Source: "deterministic",
	}
}
