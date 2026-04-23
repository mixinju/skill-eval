package eval

import "skill-eval/agent"

type SuccessScorer struct{}

func NewSuccessScorer() *SuccessScorer { return &SuccessScorer{} }

func (s *SuccessScorer) Item() ScoreItem {
	return ScoreItem{
		Name: "是否执行成功",
		Desc: "整个智能体执行流程是否完成任务",
	}
}

func (s *SuccessScorer) Score(trace *agent.Trace) Verdict {
	if trace.Success {
		return Verdict{Info: s.Item(), Pass: true, Score: 1, Reason: "agent 调用 finish 正常结束"}
	}
	return Verdict{Info: s.Item(), Pass: false, Score: 0, Reason: "agent 未成功完成任务"}
}
