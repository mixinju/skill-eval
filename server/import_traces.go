package server

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"skill-eval/agent"
	"skill-eval/eval"
	"skill-eval/runner"

	"github.com/sirupsen/logrus"
)

// ImportLegacyTraces 把 DefaultTracer 落下的 logs/traces/*.json 收成评测记录。
// 已有同 id 的记录不会覆盖。评分只用确定性规则，不调用外部模型。
func (s *Server) ImportLegacyTraces() (int, error) {
	dir := filepath.Join(s.dataDir, "traces")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}
	imported := 0
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			continue
		}
		var trace agent.Trace
		if err := json.Unmarshal(data, &trace); err != nil || strings.TrimSpace(trace.ID) == "" {
			continue
		}
		if _, err := s.store.Get(trace.ID); err == nil {
			continue
		}
		normalizeImportedTrace(&trace)
		report := eval.Exec(&trace, eval.DeterministicScorers())
		status := trace.Status
		if status == "" {
			if trace.Success {
				status = "succeeded"
			} else {
				status = "failed"
			}
		}
		created := trace.StartTime
		if created.IsZero() {
			created = time.Now().UTC()
		}
		rec := &Record{
			ID:        trace.ID,
			Type:      "single",
			CreatedAt: created.UTC(),
			Status:    status,
			Runner:    trace.Runner,
			Report:    report,
			Trace:     &trace,
		}
		if err := s.store.Save(rec); err != nil {
			logrus.Warnf("导入历史 trace %s 失败: %v", trace.ID, err)
			continue
		}
		imported++
	}
	return imported, nil
}

func normalizeImportedTrace(trace *agent.Trace) {
	if trace.SchemaVersion == 0 {
		trace.SchemaVersion = 1
	}
	if strings.TrimSpace(trace.Runner) == "" {
		trace.Runner = string(runner.TypeSelf)
		if trace.Notes == nil {
			trace.Notes = []string{}
		}
		trace.Notes = append(trace.Notes, "历史 Trace 未标注 runner，导入时归为 self")
	} else if normalized, err := runner.Normalize(trace.Runner); err == nil {
		trace.Runner = string(normalized)
	}
	if trace.Status == "" {
		if trace.Success {
			trace.Status = "succeeded"
		} else {
			trace.Status = "failed"
		}
	}
	if len(trace.Steps) == 0 {
		trace.Steps = agent.BuildSteps(trace.Spans)
	}
	if strings.TrimSpace(trace.ArtifactsAndResult) == "" {
		for i := len(trace.Spans) - 1; i >= 0; i-- {
			span := trace.Spans[i]
			if span == nil || span.Kind != agent.SpanKindToolCall || span.Name != "finish" {
				continue
			}
			if strings.TrimSpace(span.ToolOutput) != "" {
				trace.ArtifactsAndResult = span.ToolOutput
			} else if strings.TrimSpace(span.ToolInput) != "" {
				trace.ArtifactsAndResult = span.ToolInput
			}
			break
		}
	}
}
