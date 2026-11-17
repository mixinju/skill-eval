package runner

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/openai/openai-go/v3"
)

type singleFinish struct{}

func (singleFinish) Complete(ctx context.Context, params openai.ChatCompletionNewParams) (*openai.ChatCompletion, error) {
	raw := `{
	  "id":"chatcmpl-self","object":"chat.completion","created":1,"model":"test-model",
	  "choices":[{"index":0,"finish_reason":"tool_calls","message":{"role":"assistant","content":"done","tool_calls":[
	    {"id":"call_finish","type":"function","function":{"name":"finish","arguments":"{\"result\":\"ok\",\"artifacts\":[\"note.txt\"]}"}}
	  ]}}],
	  "usage":{"prompt_tokens":2,"completion_tokens":3,"total_tokens":5}
	}`
	var out openai.ChatCompletion
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func TestSelfRunnerProducesAlignedTrace(t *testing.T) {
	root := t.TempDir()
	skill := filepath.Join(root, "skills", "pdf")
	if err := os.MkdirAll(skill, 0o755); err != nil {
		t.Fatal(err)
	}
	body := "---\nname: pdf\ndescription: 当用户需要生成 PDF 时使用，不要用于别的格式。\n---\n# PDF\n1. 写文件\n- 保存到工作区\n"
	if err := os.WriteFile(filepath.Join(skill, "SKILL.md"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	self := NewSelfRunner()
	self.UseCompleter(singleFinish{})
	trace, err := self.Run(context.Background(), Task{
		ID: "self-test", Name: "self", UserPrompt: "写一个说明", Model: "test-model",
		TargetSkill: "pdf", SkillDir: filepath.Join(root, "skills"), DataDir: filepath.Join(root, "data"),
		Timeout: time.Minute, MaxIterations: 3,
	})
	if err != nil {
		t.Fatal(err)
	}
	if trace.Runner != string(TypeSelf) || !trace.Success || trace.Status != "succeeded" {
		t.Fatalf("self trace: success=%v status=%s reason=%s", trace.Success, trace.Status, trace.FailureReason)
	}
	if trace.RunnerMeta == nil || !trace.RunnerMeta.Isolated || trace.Workspace == "" || trace.SkillPath == "" {
		t.Fatalf("缺少可复核元数据: %+v", trace.RunnerMeta)
	}
	if _, err := os.Stat(trace.SkillPath); err != nil {
		t.Fatal(err)
	}
}
