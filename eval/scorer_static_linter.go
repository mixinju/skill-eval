package eval

import (
	"fmt"
	"os"
	"strings"
	"unicode/utf8"

	"skill-eval/agent"
	"skill-eval/tool"
)

// StaticLintScorer 对目标 SKILL.md 做静态检查。
// 找不到技能文件时跳过，不把缺文件算成被测任务失败。
type StaticLintScorer struct{}

func NewStaticLintScorer() *StaticLintScorer { return &StaticLintScorer{} }

func (s StaticLintScorer) Item() ScoreItem {
	return ScoreItem{
		Name: "Skill静态检查",
		Desc: "检查 SKILL.md 的名称、描述、步骤和正文完整度",
	}
}

func (s StaticLintScorer) Score(traces ...*agent.Trace) (Verdict, error) {
	first, second, err := extraTrace(traces...)
	if err != nil {
		return Verdict{}, err
	}
	if second == nil {
		return s.single(first), nil
	}
	a := s.single(first)
	b := s.single(second)
	return Verdict{
		Info:   s.Item(),
		Pass:   a.Pass && b.Pass,
		Score:  (a.Score + b.Score) / 2,
		Reason: fmt.Sprintf("A %.2f (%s)；B %.2f (%s)", a.Score, a.Reason, b.Score, b.Reason),
		Source: "static",
	}, nil
}

func (s StaticLintScorer) single(trace *agent.Trace) Verdict {
	if trace == nil {
		return Verdict{Info: s.Item(), Pass: false, Score: 0, Reason: "trace 为空", Source: "static", Error: "trace 为空"}
	}
	path := trace.SkillPath
	if path == "" {
		return Verdict{Info: s.Item(), Pass: true, Score: 1, Reason: "未提供 SKILL.md 路径，跳过静态检查", Source: "static"}
	}
	if _, err := os.Stat(path); err != nil {
		return Verdict{Info: s.Item(), Pass: true, Score: 1, Reason: "未找到 SKILL.md，跳过静态检查: " + path, Source: "static"}
	}
	skill, err := tool.NewSkill(path)
	if err != nil {
		return Verdict{Info: s.Item(), Pass: false, Score: 0, Reason: "解析 SKILL.md 失败: " + err.Error(), Source: "static", Error: err.Error()}
	}

	var got []string
	var missed []string
	score := 0.0
	add := func(ok bool, weight float64, label string) {
		if ok {
			score += weight
			got = append(got, label)
			return
		}
		missed = append(missed, label)
	}
	add(strings.TrimSpace(skill.Name) != "", 0.2, "name")
	descLen := utf8.RuneCountInString(strings.TrimSpace(skill.Description))
	add(descLen >= 40, 0.2, "description长度")
	lower := strings.ToLower(skill.Description + "\n" + skill.Content)
	add(descLen >= 80 || strings.Contains(lower, "when") || strings.Contains(skill.Description, "触发") || strings.Contains(skill.Description, "使用"), 0.15, "触发条件")
	add(strings.Contains(skill.Content, "#"), 0.15, "标题")
	add(strings.Contains(skill.Content, "\n- ") || strings.Contains(skill.Content, "\n1.") || strings.HasPrefix(skill.Content, "- ") || strings.Contains(skill.Content, "1."), 0.15, "步骤")
	add(utf8.RuneCountInString(skill.Content) >= 80, 0.15, "正文长度")

	reason := fmt.Sprintf("通过 %s", strings.Join(got, "、"))
	if len(missed) > 0 {
		reason += fmt.Sprintf("；缺少 %s", strings.Join(missed, "、"))
	}
	return Verdict{
		Info:   s.Item(),
		Pass:   score >= 0.6,
		Score:  score,
		Reason: reason,
		Source: "static",
	}
}
