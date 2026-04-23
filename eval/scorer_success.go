package eval

import (
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
	return Verdict{}, nil
}

func (s *SuccessScorer) compare(first, second *agent.Trace) (Verdict, error) {
	return Verdict{Info: s.Item(), Pass: false}, nil
}
