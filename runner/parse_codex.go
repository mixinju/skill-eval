package runner

import (
	"encoding/json"
	"strings"

	"skill-eval/agent"
)

type codexOutcome struct {
	ExternalID   string
	Iterations   int
	Tokens       int64
	Failed       bool
	Failure      string
	SawCompleted bool
	FinalText    string
	Artifacts    []string
	Notes        []string
	Model        string
}

// applyCodexOutput 把 `codex exec --json` 的 JSONL 收成与自研内核对齐的 span。
func applyCodexOutput(trace *agent.Trace, stdout, targetSkill string) codexOutcome {
	out := codexOutcome{}
	open := map[string]*agent.Span{}
	iteration := 0
	for _, line := range strings.Split(stdout, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var ev map[string]any
		if err := json.Unmarshal([]byte(line), &ev); err != nil {
			out.Notes = append(out.Notes, "非 JSON 输出: "+clipLine(line))
			continue
		}
		switch asString(ev["type"]) {
		case "thread.started":
			out.ExternalID = asString(ev["thread_id"])
		case "turn.started":
			iteration++
		case "turn.completed":
			out.SawCompleted = true
			if usage := asMap(ev["usage"]); usage != nil {
				total := asInt64(usage["input_tokens"]) + asInt64(usage["output_tokens"])
				if total > 0 {
					out.Tokens = total
				}
			}
		case "turn.failed":
			out.Failed = true
			out.Failure = codexError(ev)
		case "error":
			msg := codexError(ev)
			if strings.Contains(msg, "Reconnecting") {
				out.Notes = append(out.Notes, msg)
				continue
			}
			out.Failed = true
			if msg != "" {
				out.Failure = msg
			}
		case "item.started", "item.updated", "item.completed":
			item := asMap(ev["item"])
			if item == nil {
				continue
			}
			consumeCodexItem(trace, &out, open, asString(ev["type"]), item, iteration, targetSkill)
		}
	}
	if iteration == 0 && len(trace.Spans) > 0 {
		iteration = 1
	}
	out.Iterations = iteration
	for _, span := range open {
		closeSpan(span, span.ToolOutput, "工具调用未结束")
	}
	return out
}

func consumeCodexItem(trace *agent.Trace, out *codexOutcome, open map[string]*agent.Span, event string, item map[string]any, iteration int, target string) {
	if iteration == 0 {
		iteration = 1
	}
	id := asString(item["id"])
	kind := asString(item["type"])
	if kind == "" {
		kind = asString(item["item_type"])
	}
	switch kind {
	case "agent_message", "assistant_message":
		if event != "item.completed" {
			return
		}
		text := asString(item["text"])
		phase := asString(item["phase"])
		span := newSpan(agent.SpanKindLLMCall, "agent_message", iteration)
		span.LLMOutput = text
		span.FinishReason = phase
		trace.Spans = append(trace.Spans, span)
		if phase == "" || phase == "final_answer" {
			out.FinalText = text
		}
		out.Artifacts = mergeArtifacts(out.Artifacts, artifactsFromText(text))
	case "reasoning":
		if event != "item.completed" {
			return
		}
		span := newSpan(agent.SpanKindLLMCall, "reasoning", iteration)
		span.LLMOutput = asString(item["text"])
		trace.Spans = append(trace.Spans, span)
	case "command_execution":
		span := open[id]
		if event != "item.completed" {
			if span == nil {
				span = newSpan(agent.SpanKindToolCall, "command_execution", iteration)
				span.ToolInput = asString(item["command"])
				open[id] = span
				trace.Spans = append(trace.Spans, span)
			}
			return
		}
		if span == nil {
			span = newSpan(agent.SpanKindToolCall, "command_execution", iteration)
			span.ToolInput = asString(item["command"])
			trace.Spans = append(trace.Spans, span)
		}
		output := asString(item["aggregated_output"])
		status := asString(item["status"])
		errText := ""
		if status == "failed" || status == "declined" || (item["exit_code"] != nil && asInt64(item["exit_code"]) != 0) {
			errText = "命令失败: " + status
		}
		span.IsTarget = skillReferenced(target, span.Name, span.ToolInput, output)
		closeSpan(span, output, errText)
		delete(open, id)
	default:
		if event != "item.completed" {
			return
		}
		raw, _ := json.Marshal(item)
		name := kind
		if name == "" {
			name = "item"
		}
		span := newSpan(agent.SpanKindToolCall, name, iteration)
		span.ToolInput = string(raw)
		paths := codexPaths(item)
		output := strings.Join(paths, "\n")
		status := asString(item["status"])
		errText := ""
		if status == "failed" || status == "declined" {
			errText = name + " 失败: " + status
		}
		span.IsTarget = skillReferenced(target, name, span.ToolInput, output)
		closeSpan(span, output, errText)
		trace.Spans = append(trace.Spans, span)
		out.Artifacts = mergeArtifacts(out.Artifacts, paths)
	}
}

func codexPaths(item map[string]any) []string {
	var paths []string
	if changes, ok := item["changes"].([]any); ok {
		for _, change := range changes {
			m := asMap(change)
			if m == nil {
				continue
			}
			if p := asString(m["path"]); p != "" {
				paths = append(paths, p)
			}
		}
	}
	if p := asString(item["path"]); p != "" {
		paths = append(paths, p)
	}
	return paths
}

func codexError(ev map[string]any) string {
	if msg := asString(ev["message"]); msg != "" {
		return msg
	}
	if errMap := asMap(ev["error"]); errMap != nil {
		if msg := asString(errMap["message"]); msg != "" {
			return msg
		}
	}
	raw, _ := json.Marshal(ev["error"])
	if string(raw) != "" && string(raw) != "null" {
		return string(raw)
	}
	return "codex 执行失败"
}

func clipLine(s string) string {
	if len(s) > 180 {
		return s[:180]
	}
	return s
}
