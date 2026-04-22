package eval

import (
	"encoding/json"
	"skill-eval/agent"
)

type SkillHitScorer struct{}

func NewSkillHitScorer() *SkillHitScorer { return &SkillHitScorer{} }

func (s *SkillHitScorer) Name() string { return "skill_hit" }

func (s *SkillHitScorer) Score(trace *agent.Trace) Verdict {
	if trace.TargetSkill == "" {
		return Verdict{Name: s.Name(), Pass: true, Score: 1, Reason: "未设置目标 skill，跳过检查"}
	}

	for _, span := range trace.Spans {
		if span.Kind != agent.SpanKindToolCall || span.Name != "use_skill" {
			continue
		}
		var params map[string]any
		if err := json.Unmarshal([]byte(span.ToolInput), &params); err != nil {
			continue
		}
		if name, _ := params["name"].(string); name == trace.TargetSkill {
			return Verdict{Name: s.Name(), Pass: true, Score: 1, Reason: "命中目标 skill: " + trace.TargetSkill}
		}
	}

	return Verdict{Name: s.Name(), Pass: false, Score: 0, Reason: "未命中目标 skill: " + trace.TargetSkill}
}
