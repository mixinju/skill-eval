package eval

import (
	"fmt"
	"skill-eval/agent"
	"skill-eval/providers"
)

// Unit 一个评测的最小集合
type Unit struct {
	Id      string `json:"id"`
	Name    string `json:"name"`
	Input   string `json:"input"`
	Success bool   `json:"success"`
}

func extraTrace(trace ...*agent.Trace) (firsts, second *agent.Trace, err error) {
	c := len(trace)

	if c != 1 && c != 2 {
		return nil, nil, fmt.Errorf("参数不正确,只允许传入1个或2个")
	}

	if c == 1 {
		return trace[0], nil, nil
	}
	return trace[0], trace[1], nil
}

func BuildDefaultScorer() []Scorer {

	client := providers.NewClient()
	var s []Scorer
	processScorer := NewExecProcessScorer(&client, "kimi-2.5")
	s = append(s, processScorer)

	artifactScorer := NewArtifactScorer(&client, "kimi-2.5")

	hitScorer := NewSkillHitScorer()
	s = append(s, hitScorer)
	successScorer := NewSuccessScorer()
	s = append(s, successScorer)

	s = append(s, artifactScorer)

	return s
}
