package eval

import (
	"encoding/json"
	"fmt"

	"skill-eval/agent"
)

type SkillHitScorer struct{}

func NewSkillHitScorer() *SkillHitScorer { return &SkillHitScorer{} }

func (s *SkillHitScorer) Item() ScoreItem {
	return ScoreItem{
		Name: "Skill是否命中",
		Desc: "根据执行链路，评测是否命中目标Skill的展示",
	}
}

func (s *SkillHitScorer) Score(trace ...*agent.Trace) (Verdict, error) {

	first, second, e := extraTrace(trace...)
	if e != nil {
		return Verdict{}, e
	}

	if second == nil {
		return s.single(first)
	}

	return s.compare(first, second)

}

func (s *SkillHitScorer) single(trace *agent.Trace) (Verdict, error) {
	if trace == nil {
		return Verdict{Info: s.Item(), Pass: false, Score: 0, Reason: "trace 为空", Source: "deterministic", Error: "trace 为空"}, nil
	}
	if trace.TargetSkill == "" {
		return Verdict{Info: s.Item(), Pass: true, Score: 1, Reason: "未设置目标 skill，跳过检查", Source: "deterministic"}, nil
	}

	for _, span := range trace.Spans {
		if span == nil || span.Kind != agent.SpanKindToolCall {
			continue
		}
		if span.IsTarget {
			return Verdict{Info: s.Item(), Pass: true, Score: 1, Reason: "命中目标 skill: " + trace.TargetSkill, Source: "deterministic"}, nil
		}
		if span.Name != "use_skill" && span.Name != "Skill" && span.Name != "skill" {
			continue
		}
		var params map[string]any
		if err := json.Unmarshal([]byte(span.ToolInput), &params); err != nil {
			continue
		}
		name, _ := params["name"].(string)
		if name == "" {
			name, _ = params["skill"].(string)
		}
		if name == trace.TargetSkill {
			return Verdict{Info: s.Item(), Pass: true, Score: 1, Reason: "命中目标 skill: " + trace.TargetSkill, Source: "deterministic"}, nil
		}
	}

	return Verdict{Info: s.Item(), Pass: false, Score: 0, Reason: "未命中目标 skill: " + trace.TargetSkill, Source: "deterministic"}, nil
}

func (s *SkillHitScorer) compare(first, second *agent.Trace) (Verdict, error) {
	a, _ := s.single(first)
	b, _ := s.single(second)
	score := 0.0
	switch {
	case a.Pass && b.Pass:
		score = 1
	case a.Pass || b.Pass:
		score = 0.5
	}
	return Verdict{
		Info:   s.Item(),
		Pass:   a.Pass && b.Pass,
		Score:  score,
		Reason: fmt.Sprintf("A: %s；B: %s", a.Reason, b.Reason),
		Source: "deterministic",
	}, nil
}
