package eval

import (
	"fmt"
	"os"
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

// BuildDefaultScorer 返回默认定评分器。
// 规则评分器始终启用。配置了 EVAL_API_KEY 时，额外启用大模型裁判，模型名来自 EVAL_JUDGE_MODEL（默认 kimi-2.5）。
func BuildDefaultScorer() []Scorer {
	s := DeterministicScorers()
	if os.Getenv("EVAL_API_KEY") == "" {
		return s
	}
	client := providers.NewClient()
	model := os.Getenv("EVAL_JUDGE_MODEL")
	if model == "" {
		model = "kimi-2.5"
	}
	s = append(s, NewExecProcessScorer(&client, model))
	s = append(s, NewArtifactScorer(&client, model))
	return s
}

// DeterministicScorers 不依赖外部模型，适合本地与 CI 复核。
func DeterministicScorers() []Scorer {
	return []Scorer{
		NewSuccessScorer(),
		NewSkillHitScorer(),
		NewDeterministicProcessScorer(),
		NewStaticLintScorer(),
	}
}
