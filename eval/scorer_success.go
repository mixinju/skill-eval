package eval

import (
	"fmt"

	"skill-eval/agent"
)

type SuccessScorer struct{}

func NewSuccessScorer() *SuccessScorer { return &SuccessScorer{} }

func (s *SuccessScorer) Item() ScoreItem {
	return ScoreItem{
		Name: "是否执行成功",
		Desc: "整个智能体执行流程是否完成任务",
	}
}

func (s *SuccessScorer) Score(trace ...*agent.Trace) (Verdict, error) {

	first, second, e := extraTrace(trace...)
	if e != nil {
		return Verdict{}, e
	}

	if second == nil {
		return s.single(first)
	}

	return s.compare(first, second)

}

func (s *SuccessScorer) single(trace *agent.Trace) (Verdict, error) {
	if trace == nil {
		return Verdict{Info: s.Item(), Pass: false, Score: 0, Reason: "trace 为空", Source: "deterministic", Error: "trace 为空"}, nil
	}
	if trace.Success {
		return Verdict{Info: s.Item(), Pass: true, Score: 1, Reason: "执行流程正常完成", Source: "deterministic"}, nil
	}
	reason := "执行未完成"
	if trace.FailureReason != "" {
		reason = trace.FailureReason
	} else if trace.Error != "" {
		reason = trace.Error
	}
	return Verdict{Info: s.Item(), Pass: false, Score: 0, Reason: reason, Source: "deterministic"}, nil
}

func (s *SuccessScorer) compare(first, second *agent.Trace) (Verdict, error) {
	a, _ := s.single(first)
	b, _ := s.single(second)
	pass := a.Pass && b.Pass
	score := 0.0
	switch {
	case a.Pass && b.Pass:
		score = 1
	case a.Pass || b.Pass:
		score = 0.5
	}
	reason := fmt.Sprintf("A: %s；B: %s", a.Reason, b.Reason)
	return Verdict{Info: s.Item(), Pass: pass, Score: score, Reason: reason, Source: "deterministic"}, nil
}
