package eval

import (
	"skill-eval/agent"
)

type Verdict struct {
	Info   ScoreItem `json:"info"`
	Pass   bool      `json:"pass"`
	Score  float64   `json:"score"`  // 这个字段只在单流程是有效的，对比评测下，这个分数没有意义
	Reason string    `json:"reason"` // AI评测的结论
}

type ScoreItem struct {
	Name string `json:"name"` // 评分项名称
	Desc string `json:"desc"` // 评分项描述
}
type Scorer interface {
	Item() ScoreItem
	Score(trace ...*agent.Trace) (Verdict, error)
}
