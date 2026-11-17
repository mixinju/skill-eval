package runner

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"skill-eval/agent"
)

func TestCodexParseAndRun(t *testing.T) {
	stdout := strings.Join([]string{
		`{"type":"thread.started","thread_id":"th_1"}`,
		`{"type":"turn.started"}`,
		`{"type":"item.completed","item":{"id":"item_0","type":"agent_message","text":"先读 skill"}}`,
		`{"type":"item.started","item":{"id":"item_1","type":"command_execution","command":"cat .agents/skills/pdf/SKILL.md","status":"in_progress"}}`,
		`{"type":"item.completed","item":{"id":"item_1","type":"command_execution","command":"cat .agents/skills/pdf/SKILL.md","aggregated_output":"# PDF","exit_code":0,"status":"completed"}}`,
		`{"type":"item.completed","item":{"id":"item_2","type":"file_change","changes":[{"path":"out.txt","kind":"add"}],"status":"completed"}}`,
		`{"type":"item.completed","item":{"id":"item_3","type":"agent_message","phase":"final_answer","text":"完成\n<artifacts>\nout.txt\n</artifacts>"}}`,
		`{"type":"turn.completed","usage":{"input_tokens":10,"output_tokens":5}}`,
	}, "\n")

	trace := &agent.Trace{Spans: []*agent.Span{}, TargetSkill: "pdf"}
	out := applyCodexOutput(trace, stdout, "pdf")
	if !out.SawCompleted || out.Failed || out.Tokens != 15 || out.ExternalID != "th_1" {
		t.Fatalf("codex outcome: %+v", out)
	}
	if !strings.Contains(out.FinalText, "完成") {
		t.Fatalf("final=%s", out.FinalText)
	}
	hit := false
	for _, span := range trace.Spans {
		if span.Name == "command_execution" && span.IsTarget {
			hit = true
		}
	}
	if !hit {
		t.Fatal("读取目标 skill 的命令应标记 isTarget")
	}

	codex := NewCodexRunner()
	codex.run = func(ctx context.Context, spec commandSpec) (string, string, int, error) {
		joined := strings.Join(spec.Args, " ")
		if !strings.Contains(joined, "--json") || !strings.Contains(joined, "--skip-git-repo-check") || !strings.Contains(joined, "workspace-write") {
			t.Fatalf("codex 参数不完整: %s", joined)
		}
		if strings.Contains(joined, "OPENAI_API_KEY") || strings.Contains(joined, "sk-test-secret") {
			t.Fatal("参数中不应出现密钥")
		}
		return stdout, "", 0, nil
	}
	skill := t.TempDir()
	if err := os.MkdirAll(filepath.Join(skill, "pdf"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skill, "pdf", "SKILL.md"), []byte("---\nname: pdf\ndescription: 生成 PDF\n---\n# PDF\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := codex.Run(context.Background(), Task{
		ID: "codex-test", UserPrompt: "做成 pdf", Model: "gpt-test", TargetSkill: "pdf",
		SkillDir: skill, DataDir: t.TempDir(), Timeout: time.Minute, MaxIterations: 3,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Runner != "codex" || !got.Success || got.RunnerMeta == nil || !got.RunnerMeta.Isolated {
		t.Fatalf("trace 未对齐: %+v meta=%+v", got, got.RunnerMeta)
	}
	if !strings.Contains(got.ArtifactsAndResult, "out.txt") {
		t.Fatalf("产物结构未对齐: %s", got.ArtifactsAndResult)
	}
	if _, err := os.Stat(filepath.Join(got.Workspace, ".agents", "skills", "pdf", "SKILL.md")); err != nil {
		t.Fatal(err)
	}
}

func TestClaudeParseAndMissingBinary(t *testing.T) {
	stdout := strings.Join([]string{
		`{"type":"system","subtype":"init","session_id":"sess","model":"claude-test"}`,
		`{"type":"assistant","message":{"model":"claude-test","stop_reason":"tool_use","usage":{"input_tokens":4,"output_tokens":3},"content":[{"type":"text","text":"使用 skill"},{"type":"tool_use","id":"toolu_1","name":"Skill","input":{"skill":"pdf"}}]}}`,
		`{"type":"user","message":{"content":[{"type":"tool_result","tool_use_id":"toolu_1","content":"已加载 .claude/skills/pdf/SKILL.md"}]}}`,
		`{"type":"assistant","message":{"stop_reason":"end_turn","content":[{"type":"text","text":"好了\n<artifacts>\nresult.pdf\n</artifacts>"}]}}`,
		`{"type":"result","subtype":"success","result":"好了\n<artifacts>\nresult.pdf\n</artifacts>","usage":{"input_tokens":8,"output_tokens":6}}`,
	}, "\n")
	trace := &agent.Trace{Spans: []*agent.Span{}, TargetSkill: "pdf"}
	out := applyClaudeOutput(trace, stdout, "pdf")
	if !out.OK || out.Failed || out.Tokens != 14 || out.Model != "claude-test" {
		t.Fatalf("claude outcome: %+v", out)
	}
	hit := false
	for _, span := range trace.Spans {
		if span.Name == "Skill" && span.IsTarget && span.Error == "" {
			hit = true
		}
	}
	if !hit {
		t.Fatalf("claude skill span 未对齐: %#v", trace.Spans)
	}

	t.Setenv("CLAUDE_BIN", filepath.Join(t.TempDir(), "missing-claude"))
	_, err := NewClaudeCodeRunner().Run(context.Background(), Task{UserPrompt: "hi", DataDir: t.TempDir()})
	if err == nil || !strings.Contains(err.Error(), "CLAUDE_BIN") {
		t.Fatalf("缺少 CLI 时应给出明确错误: %v", err)
	}
}

func TestNormalizeRunner(t *testing.T) {
	kind, err := Normalize("claude_code")
	if err != nil || kind != TypeClaudeCode {
		t.Fatalf("%s %v", kind, err)
	}
	if _, err := Normalize("nope"); err == nil {
		t.Fatal("未知 runner 应失败")
	}
}
