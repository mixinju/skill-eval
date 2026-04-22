package eval

import (
	"skill-eval/agent"
)

type Verdict struct {
	Name   string  `json:"name"`
	Pass   bool    `json:"pass"`
	Score  float64 `json:"score"`
	Reason string  `json:"reason"`
}

type Scorer interface {
	Name() string
	Score(trace *agent.Trace) Verdict
}
