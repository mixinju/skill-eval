package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"skill-eval/agent"
	"skill-eval/runner"
)

type stubRunner struct {
	kind runner.Type
}

func (s stubRunner) Type() runner.Type   { return s.kind }
func (s stubRunner) DisplayName() string { return string(s.kind) }
func (s stubRunner) Probe(context.Context) runner.Availability {
	return runner.Availability{Type: s.kind, Name: s.DisplayName(), Available: true, Detail: "stub"}
}
func (s stubRunner) Run(_ context.Context, task runner.Task) (*agent.Trace, error) {
	now := time.Now()
	return &agent.Trace{
		ID: task.ID, AgentName: task.Name, Model: task.Model, UserPrompt: task.UserPrompt,
		TargetSkill: task.TargetSkill, StartTime: now.Add(-time.Millisecond), EndTime: now,
		Iterations: 1, Success: true, Status: "succeeded", SchemaVersion: 1, Runner: string(s.kind),
		Spans:              []*agent.Span{{SpanID: "span-1", Kind: agent.SpanKindToolCall, Name: "use_skill", IsTarget: true, ToolInput: `{"name":"pdf"}`}},
		ArtifactsAndResult: `{"result":"ok","artifacts":[]}`,
		RunnerMeta:         &agent.RunnerMeta{Type: string(s.kind), Isolated: true, LogPath: filepath.Join(task.DataDir, "runs", task.ID, "logs", "stdout.log")},
	}, nil
}

func TestEvalAPI(t *testing.T) {
	t.Setenv("EVAL_API_KEY", "")
	dir := t.TempDir()
	reg := runner.NewRegistry(stubRunner{kind: runner.TypeSelf}, stubRunner{kind: runner.TypeCodex})
	srv, err := New(dir, reg)
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	body := []byte(`{"runner":"self","userPrompt":"写 pdf","model":"test","targetSkill":"pdf","wait":true}`)
	res, err := http.Post(ts.URL+"/api/evals", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", res.StatusCode)
	}
	var rec Record
	if err := json.NewDecoder(res.Body).Decode(&rec); err != nil {
		t.Fatal(err)
	}
	if rec.Runner != "self" || rec.Status != "succeeded" || rec.Report == nil || !rec.Report.Pass || rec.Trace == nil || len(rec.Trace.Spans) != 1 {
		t.Fatalf("记录不完整: status=%s pass=%v spans=%d err=%s", rec.Status, rec.Report != nil && rec.Report.Pass, len(rec.Trace.Spans), rec.Error)
	}
	if rec.Report.Runner != "self" || rec.Trace.SchemaVersion != 1 {
		t.Fatalf("契约字段缺失: runner=%s schema=%d", rec.Report.Runner, rec.Trace.SchemaVersion)
	}

	logPath := rec.LogPath
	if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(logPath, []byte("hello-log"), 0o644); err != nil {
		t.Fatal(err)
	}
	logRes, err := http.Get(ts.URL + "/api/evals/" + rec.ID + "/logs")
	if err != nil {
		t.Fatal(err)
	}
	defer logRes.Body.Close()
	if logRes.StatusCode != http.StatusOK {
		t.Fatalf("logs status=%d", logRes.StatusCode)
	}

	bodyB := []byte(`{"runner":"codex","userPrompt":"写 pdf","model":"codex-model","targetSkill":"pdf","wait":true}`)
	resB, err := http.Post(ts.URL+"/api/evals", "application/json", bytes.NewReader(bodyB))
	if err != nil {
		t.Fatal(err)
	}
	defer resB.Body.Close()
	var recB Record
	if err := json.NewDecoder(resB.Body).Decode(&recB); err != nil {
		t.Fatal(err)
	}
	cmpBody := []byte(`{"evalIdA":"` + rec.ID + `","evalIdB":"` + recB.ID + `"}`)
	cmpRes, err := http.Post(ts.URL+"/api/evals/compare", "application/json", bytes.NewReader(cmpBody))
	if err != nil {
		t.Fatal(err)
	}
	defer cmpRes.Body.Close()
	if cmpRes.StatusCode != http.StatusOK {
		t.Fatalf("compare status=%d", cmpRes.StatusCode)
	}
	var cmp Record
	if err := json.NewDecoder(cmpRes.Body).Decode(&cmp); err != nil {
		t.Fatal(err)
	}
	if cmp.Type != "compare" || cmp.CompareReport == nil || cmp.CompareReport.Conclusion == "" {
		t.Fatalf("对比记录异常: %+v", cmp.CompareReport)
	}

	runnersRes, err := http.Get(ts.URL + "/api/runners")
	if err != nil {
		t.Fatal(err)
	}
	defer runnersRes.Body.Close()
	var listed struct {
		Runners []runner.Availability `json:"runners"`
	}
	if err := json.NewDecoder(runnersRes.Body).Decode(&listed); err != nil {
		t.Fatal(err)
	}
	if len(listed.Runners) != 2 {
		t.Fatalf("runners=%d", len(listed.Runners))
	}
}
