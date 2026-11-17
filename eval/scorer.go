package eval

import (
	"skill-eval/agent"
)

type Verdict struct {
	Info   ScoreItem `json:"info"`
	Pass   bool      `json:"pass"`
	Score  float64   `json:"score"`  // 这个字段只在单流程是有效的，对比评测下，这个分数没有意义
	Reason string    `json:"reason"` // 评分结论
	// Source 标明判定来源：deterministic（规则）、llm（模型裁判）、static（技能静态检查）。
	Source string `json:"source,omitempty"`
	// Error 是评分器自身失败的原因，与被测 skill 的失败分开记录。
	Error string `json:"error,omitempty"`
}

type ScoreItem struct {
	Name string `json:"name"` // 评分项名称
	Desc string `json:"desc"` // 评分项描述
}
type Scorer interface {
	Item() ScoreItem
	Score(trace ...*agent.Trace) (Verdict, error)
}
