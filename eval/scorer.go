package eval

import (
	"skill-eval/agent"
)

type Verdict struct {
	Info   ScoreItem `json:"info"`
	Pass   bool      `json:"pass"`
	Score  float64   `json:"score"`
	Reason string    `json:"reason"`
}

type ScoreItem struct {
	Name string `json:"name"` // 评分项名称
	Desc string `json:"desc"` // 评分项描述
}
type Scorer interface {
	Item() ScoreItem
	Score(trace ...*agent.Trace) (Verdict, error)
}
