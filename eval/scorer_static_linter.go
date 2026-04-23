package eval

import (
	"skill-eval/agent"

	"github.com/openai/openai-go/v3"
)

// StaticLintScorer 静态评分器
// 侧重与文档质量工程
type StaticLintScorer struct {
	client *openai.Client
	Model  string
}

func (s StaticLintScorer) Item() ScoreItem {
	//TODO implement me
	panic("implement me")
}

func (s StaticLintScorer) Score(trace ...*agent.Trace) (Verdict, error) {
	//TODO implement me
	panic("implement me")
}
