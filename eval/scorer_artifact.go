package eval

import (
	"encoding/json"
	"fmt"
	"skill-eval/agent"
	"skill-eval/tool"

	"github.com/sirupsen/logrus"
)

type ArtifactScorer struct{}

func NewArtifactScorer() *ArtifactScorer { return &ArtifactScorer{} }

func (s *ArtifactScorer) Name() string { return "artifact" }

func (s *ArtifactScorer) Score(trace *agent.Trace) Verdict {

	var a tool.FinishResult
	if err := json.Unmarshal([]byte(trace.ArtifactsAndResult), &a); err != nil {
		logrus.Warnf("反序列化Artifacts失败: %v", err)
	}
	if len(a.Artifacts) > 0 {
		return Verdict{Name: s.Name(), Pass: true, Score: 1, Reason: fmt.Sprintf("产出 %d 个文件", len(a.Artifacts))}
	}
	return Verdict{Name: s.Name(), Pass: false, Score: 0, Reason: "未产出任何文件"}
}
