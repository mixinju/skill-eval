package eval

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"skill-eval/agent"
)

func sampleTrace(success bool) *agent.Trace {
	now := time.Now()
	return &agent.Trace{
		ID:            "t1",
		AgentName:     "tester",
		Model:         "m",
		UserPrompt:    "写一份 pdf",
		TargetSkill:   "pdf",
		StartTime:     now.Add(-time.Second),
		EndTime:       now,
		Iterations:    2,
		TotalTokens:   10,
		Success:       success,
		Status:        map[bool]string{true: "succeeded", false: "failed"}[success],
		FailureReason: map[bool]string{true: "", false: "连续2次无工具调用，任务未调用 finish"}[success],
		Runner:        "self",
		Spans: []*agent.Span{
			{SpanID: "s1", Kind: agent.SpanKindToolCall, Name: "use_skill", IsTarget: true, ToolInput: `{"name":"pdf"}`},
			{SpanID: "s2", Kind: agent.SpanKindToolCall, Name: "finish", ToolOutput: `{"result":"ok","artifacts":["a.pdf"]}`},
		},
	}
}

func TestSuccessAndSkillScorers(t *testing.T) {
	ok, err := NewSuccessScorer().Score(sampleTrace(true))
	if err != nil || !ok.Pass || ok.Source != "deterministic" {
		t.Fatalf("成功评分异常: %+v %v", ok, err)
	}
	bad, err := NewSuccessScorer().Score(sampleTrace(false))
	if err != nil || bad.Pass || bad.Reason == "" {
		t.Fatalf("失败评分应带原因: %+v %v", bad, err)
	}
	hit, err := NewSkillHitScorer().Score(sampleTrace(true))
	if err != nil || !hit.Pass {
		t.Fatalf("应命中 skill: %+v %v", hit, err)
	}
	if _, err := NewExecProcessScorer(nil, "m").Score(); err == nil {
		t.Fatal("空 trace 应返回错误")
	}
}

func TestStaticAndProcessAndCompare(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "SKILL.md")
	body := "---\nname: pdf\ndescription: 当用户需要生成 PDF 文件时使用。不要用于图片编辑。\n---\n# PDF\n1. 收集内容\n- 写成文件\n正文足够长，用来通过静态检查的长度门槛，并说明触发条件。\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	trace := sampleTrace(true)
	trace.SkillPath = path
	trace.ArtifactsAndResult = `{"result":"ok","artifacts":["a.pdf"]}`
	static, err := NewStaticLintScorer().Score(trace)
	if err != nil || !static.Pass || static.Source != "static" {
		t.Fatalf("静态检查: %+v %v", static, err)
	}
	proc, err := NewDeterministicProcessScorer().Score(trace)
	if err != nil || !proc.Pass || proc.Source != "deterministic" {
		t.Fatalf("过程评分: %+v %v", proc, err)
	}
	report := Exec(trace, DeterministicScorers())
	if !report.Pass || report.Runner != "self" || len(report.Scores) != 4 {
		t.Fatalf("报告不完整: %+v", report)
	}
	other := sampleTrace(true)
	other.ID = "t2"
	other.Runner = "codex"
	other.Iterations = 6
	otherReport := Exec(other, DeterministicScorers())
	cmp, err := Compare(trace, other, report, otherReport)
	if err != nil || cmp.Conclusion == "" || len(cmp.Scores) != 2 {
		t.Fatalf("对比失败: %+v %v", cmp, err)
	}
}
