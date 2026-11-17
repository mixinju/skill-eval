package agent

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/openai/openai-go/v3"
)

type scriptCompleter struct {
	responses []*openai.ChatCompletion
	errs      []error
	calls     int
}

func (s *scriptCompleter) Complete(ctx context.Context, params openai.ChatCompletionNewParams) (*openai.ChatCompletion, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	idx := s.calls
	s.calls++
	if idx < len(s.errs) && s.errs[idx] != nil {
		return nil, s.errs[idx]
	}
	if idx >= len(s.responses) {
		return nil, errors.New("没有更多脚本响应")
	}
	return s.responses[idx], nil
}

func mustCompletion(t *testing.T, raw string) *openai.ChatCompletion {
	t.Helper()
	var out openai.ChatCompletion
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		t.Fatalf("解析 completion: %v", err)
	}
	return &out
}

func toolCompletion(name, args string) string {
	return `{
	  "id":"chatcmpl-test",
	  "object":"chat.completion",
	  "created":1,
	  "model":"test-model",
	  "choices":[{
	    "index":0,
	    "finish_reason":"tool_calls",
	    "message":{
	      "role":"assistant",
	      "content":"ok",
	      "tool_calls":[{
	        "id":"call_` + name + `",
	        "type":"function",
	        "function":{"name":"` + name + `","arguments":` + jsonString(args) + `}
	      }]
	    }
	  }],
	  "usage":{"prompt_tokens":3,"completion_tokens":4,"total_tokens":7}
	}`
}

func jsonString(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

func testOrchestrator(t *testing.T, comp Completer, skillDir string) (*Orchestrator, string) {
	t.Helper()
	ws := t.TempDir()
	cfg := NewAgentConfig(
		WithName("tester"),
		WithSystemPrompt("完成任务后调用 finish"),
		WithUserPrompt("生成一份说明"),
		WithModel("test-model"),
		WithMaxIterations(4),
		WithWorkspace(ws),
	)
	cfg.RegistryDefaultTools()
	if skillDir != "" {
		cfg.RegistrySkillsFrom(skillDir)
	}
	o := NewOrchestrator(comp, cfg)
	o.RetryBase = time.Millisecond
	o.MaxAttempts = 3
	tr := NewDefaultTracer(filepath.Join(t.TempDir(), "traces"))
	o.Tracer = tr
	o.SetTargetSkill("pdf")
	return o, ws
}

func writeSkill(t *testing.T, root, name, body string) {
	t.Helper()
	dir := filepath.Join(root, name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	content := "---\nname: " + name + "\ndescription: 当用户需要生成 PDF 时使用这个技能，不要用于图片。\n---\n# PDF\n1. 收集内容\n- 写成文件\n"
	if body != "" {
		content = body
	}
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestOrchestratorFinishAndSkillHit(t *testing.T) {
	skillRoot := t.TempDir()
	writeSkill(t, skillRoot, "pdf", "")
	comp := &scriptCompleter{responses: []*openai.ChatCompletion{
		mustCompletion(t, toolCompletion("use_skill", `{"name":"pdf"}`)),
		mustCompletion(t, toolCompletion("finish", `{"result":"done","artifacts":["out.txt"]}`)),
	}}
	o, _ := testOrchestrator(t, comp, skillRoot)
	if err := o.RunContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	trace := o.GetTrace()
	if trace == nil || !trace.Success {
		t.Fatalf("期望成功，trace=%+v", trace)
	}
	if trace.Iterations != 2 {
		t.Fatalf("iterations=%d", trace.Iterations)
	}
	if trace.Status != "succeeded" {
		t.Fatalf("status=%s", trace.Status)
	}
	if trace.ArtifactsAndResult == "" || trace.ArtifactsAndResult == "null" {
		t.Fatalf("缺少产物: %s", trace.ArtifactsAndResult)
	}
	seen := map[string]bool{}
	hit := false
	for _, span := range trace.Spans {
		if seen[span.SpanID] {
			t.Fatalf("span id 重复: %s", span.SpanID)
		}
		seen[span.SpanID] = true
		if span.Name == "use_skill" && span.IsTarget {
			hit = true
		}
	}
	if !hit {
		t.Fatalf("未记录目标 skill span: %#v", trace.Spans)
	}
	if len(trace.Steps) == 0 {
		t.Fatal("缺少 steps")
	}
}

func TestOrchestratorRetriesTransientError(t *testing.T) {
	comp := &scriptCompleter{
		errs: []error{errors.New("status 429 rate limit")},
		responses: []*openai.ChatCompletion{
			nil,
			mustCompletion(t, toolCompletion("finish", `{"result":"done"}`)),
		},
	}
	o, _ := testOrchestrator(t, comp, "")
	if err := o.RunContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	trace := o.GetTrace()
	if trace == nil || !trace.Success {
		t.Fatalf("重试后应成功: %+v", trace)
	}
	if len(trace.Notes) == 0 {
		t.Fatal("重试没有记入 notes")
	}
}

func TestOrchestratorStopsWhenNoToolCall(t *testing.T) {
	raw := `{
	  "id":"chatcmpl-text","object":"chat.completion","created":1,"model":"test-model",
	  "choices":[{"index":0,"finish_reason":"stop","message":{"role":"assistant","content":"还没做完"}}],
	  "usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2}
	}`
	comp := &scriptCompleter{responses: []*openai.ChatCompletion{
		mustCompletion(t, raw),
		mustCompletion(t, raw),
	}}
	o, _ := testOrchestrator(t, comp, "")
	err := o.RunContext(context.Background())
	if err == nil {
		t.Fatal("期望失败")
	}
	trace := o.GetTrace()
	if trace == nil || trace.Success || trace.FailureReason == "" {
		t.Fatalf("失败原因应写入 trace: %+v err=%v", trace, err)
	}
}

func TestOrchestratorUnknownSkill(t *testing.T) {
	comp := &scriptCompleter{responses: []*openai.ChatCompletion{
		mustCompletion(t, toolCompletion("use_skill", `{"name":"missing"}`)),
		mustCompletion(t, toolCompletion("finish", `{"result":"done"}`)),
	}}
	o, _ := testOrchestrator(t, comp, "")
	if err := o.RunContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, span := range o.GetTrace().Spans {
		if span.Name == "use_skill" && span.Error != "" {
			found = true
		}
	}
	if !found {
		t.Fatal("未知 skill 应记入 span error")
	}
}
