package eval

import "skill-eval/agent"

type SuccessScorer struct{}

func NewSuccessScorer() *SuccessScorer { return &SuccessScorer{} }

func (s *SuccessScorer) Name() string { return "success" }

func (s *SuccessScorer) Score(trace *agent.Trace) Verdict {
	if trace.Success {
		return Verdict{Name: s.Name(), Pass: true, Score: 1, Reason: "agent 调用 finish 正常结束"}
	}
	return Verdict{Name: s.Name(), Pass: false, Score: 0, Reason: "agent 未成功完成任务"}
}
