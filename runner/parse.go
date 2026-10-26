package runner

import (
	"encoding/json"
	"regexp"
	"strings"
	"time"

	"skill-eval/agent"
	"skill-eval/tool"

	"github.com/google/uuid"
)

func asString(v any) string {
	s, _ := v.(string)
	return s
}

func asMap(v any) map[string]any {
	m, _ := v.(map[string]any)
	return m
}

func asInt64(v any) int64 {
	switch n := v.(type) {
	case float64:
		return int64(n)
	case int64:
		return n
	case int:
		return int64(n)
	case json.Number:
		i, _ := n.Int64()
		return i
	default:
		return 0
	}
}

func flattenContent(v any) string {
	switch c := v.(type) {
	case string:
		return c
	case []any:
		var parts []string
		for _, item := range c {
			m := asMap(item)
			if m == nil {
				if s, ok := item.(string); ok {
					parts = append(parts, s)
				}
				continue
			}
			if text := asString(m["text"]); text != "" {
				parts = append(parts, text)
			} else if content := flattenContent(m["content"]); content != "" {
				parts = append(parts, content)
			}
		}
		return strings.Join(parts, "\n")
	default:
		if m := asMap(v); m != nil {
			if text := asString(m["text"]); text != "" {
				return text
			}
		}
		raw, err := json.Marshal(v)
		if err != nil {
			return ""
		}
		if string(raw) == "null" {
			return ""
		}
		return string(raw)
	}
}

func skillReferenced(target, toolName, input, output string) bool {
	if target == "" {
		return false
	}
	switch strings.ToLower(toolName) {
	case "use_skill", "skill", "skills":
		if strings.Contains(input, target) || strings.Contains(output, target) {
			return true
		}
	}
	needles := []string{
		"/skills/" + target + "/",
		"/skills/" + target + "/SKILL.md",
		".agents/skills/" + target,
		".claude/skills/" + target,
	}
	blob := input + "\n" + output
	for _, needle := range needles {
		if strings.Contains(blob, needle) {
			return true
		}
	}
	return false
}

var artifactBlock = regexp.MustCompile(`(?s)<artifacts>(.*?)</artifacts>`)

func artifactsFromText(text string) []string {
	m := artifactBlock.FindStringSubmatch(text)
	if len(m) < 2 {
		return nil
	}
	var out []string
	for _, line := range strings.Split(m[1], "\n") {
		line = strings.TrimSpace(strings.TrimPrefix(line, "-"))
		line = strings.Trim(line, "`\"' ")
		if line == "" || strings.Contains(line, "..") {
			continue
		}
		out = append(out, line)
	}
	return out
}

func mergeArtifacts(groups ...[]string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, group := range groups {
		for _, item := range group {
			item = strings.TrimSpace(item)
			if item == "" {
				continue
			}
			if _, ok := seen[item]; ok {
				continue
			}
			seen[item] = struct{}{}
			out = append(out, item)
		}
	}
	return out
}

func finishPayload(result string, artifacts []string) string {
	if result == "" {
		result = ""
	}
	raw, err := json.Marshal(tool.FinishResult{Result: result, Artifacts: artifacts})
	if err != nil {
		return `{"result":"","artifacts":[]}`
	}
	return string(raw)
}

func newSpan(kind agent.SpanKind, name string, iteration int) *agent.Span {
	now := time.Now()
	return &agent.Span{
		SpanID:    uuid.NewString(),
		Kind:      kind,
		Name:      name,
		Iteration: iteration,
		StartTime: now,
		EndTime:   now,
	}
}

func closeSpan(span *agent.Span, output, errText string) {
	if span == nil {
		return
	}
	span.EndTime = time.Now()
	if span.EndTime.Before(span.StartTime) {
		span.EndTime = span.StartTime
	}
	span.Duration = span.EndTime.Sub(span.StartTime).Milliseconds()
	if output != "" {
		span.ToolOutput = output
	}
	span.Error = errText
}
