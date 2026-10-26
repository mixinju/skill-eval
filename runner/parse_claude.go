package runner

import (
	"encoding/json"
	"strings"

	"skill-eval/agent"
)

type claudeOutcome struct {
	ExternalID string
	Iterations int
	Tokens     int64
	Failed     bool
	Failure    string
	SawResult  bool
	OK         bool
	FinalText  string
	Artifacts  []string
	Notes      []string
	Model      string
}

// applyClaudeOutput 把 `claude -p --output-format stream-json` 收成统一 span。
func applyClaudeOutput(trace *agent.Trace, stdout, targetSkill string) claudeOutcome {
	out := claudeOutcome{}
	open := map[string]*agent.Span{}
	iteration := 0
	var assistantTokens int64
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
		case "system":
			if asString(ev["subtype"]) == "init" {
				out.ExternalID = asString(ev["session_id"])
				if model := asString(ev["model"]); model != "" {
					out.Model = model
				}
			}
		case "assistant":
			iteration++
			msg := asMap(ev["message"])
			if msg == nil {
				continue
			}
			if model := asString(msg["model"]); model != "" {
				out.Model = model
			}
			if usage := asMap(msg["usage"]); usage != nil {
				assistantTokens += asInt64(usage["input_tokens"]) + asInt64(usage["output_tokens"])
			}
			consumeClaudeAssistant(trace, &out, open, msg, iteration, targetSkill)
		case "user":
			msg := asMap(ev["message"])
			if msg == nil {
				continue
			}
			consumeClaudeUser(open, msg, targetSkill)
		case "result":
			out.SawResult = true
			out.FinalText = asString(ev["result"])
			out.Artifacts = mergeArtifacts(out.Artifacts, artifactsFromText(out.FinalText))
			if usage := asMap(ev["usage"]); usage != nil {
				total := asInt64(usage["input_tokens"]) + asInt64(usage["output_tokens"])
				if total == 0 {
					total = asInt64(usage["inputTokens"]) + asInt64(usage["outputTokens"])
				}
				if total > 0 {
					out.Tokens = total
				}
			}
			if out.Tokens == 0 {
				out.Tokens = asInt64(ev["total_input_tokens"]) + asInt64(ev["total_output_tokens"])
			}
			isErr, _ := ev["is_error"].(bool)
			subtype := asString(ev["subtype"])
			if isErr || strings.Contains(subtype, "error") {
				out.Failed = true
				out.OK = false
				if out.FinalText != "" {
					out.Failure = out.FinalText
				} else {
					out.Failure = "Claude Code 执行失败: " + subtype
				}
			} else {
				out.OK = true
			}
		}
	}
	if out.Tokens == 0 {
		out.Tokens = assistantTokens
	}
	if iteration == 0 && len(trace.Spans) > 0 {
		iteration = 1
	}
	out.Iterations = iteration
	for _, span := range open {
		closeSpan(span, "", "工具调用没有对应的 tool_result")
	}
	return out
}

func consumeClaudeAssistant(trace *agent.Trace, out *claudeOutcome, open map[string]*agent.Span, msg map[string]any, iteration int, target string) {
	content, _ := msg["content"].([]any)
	var text strings.Builder
	var tools []map[string]any
	for _, block := range content {
		m := asMap(block)
		if m == nil {
			continue
		}
		switch asString(m["type"]) {
		case "text":
			if t := asString(m["text"]); t != "" {
				if text.Len() > 0 {
					text.WriteByte('\n')
				}
				text.WriteString(t)
			}
		case "thinking":
			span := newSpan(agent.SpanKindLLMCall, "thinking", iteration)
			span.LLMOutput = asString(m["thinking"])
			if span.LLMOutput == "" {
				span.LLMOutput = asString(m["text"])
			}
			trace.Spans = append(trace.Spans, span)
		case "tool_use", "server_tool_use", "mcp_tool_use":
			tools = append(tools, m)
		}
	}
	if text.Len() > 0 || len(tools) == 0 {
		span := newSpan(agent.SpanKindLLMCall, "assistant", iteration)
		span.LLMOutput = text.String()
		span.FinishReason = asString(msg["stop_reason"])
		trace.Spans = append(trace.Spans, span)
		out.Artifacts = mergeArtifacts(out.Artifacts, artifactsFromText(text.String()))
	}
	for _, toolUse := range tools {
		id := asString(toolUse["id"])
		name := asString(toolUse["name"])
		inputRaw, _ := json.Marshal(toolUse["input"])
		span := newSpan(agent.SpanKindToolCall, name, iteration)
		span.ToolInput = string(inputRaw)
		span.IsTarget = skillReferenced(target, name, span.ToolInput, "")
		if id != "" {
			open[id] = span
		}
		trace.Spans = append(trace.Spans, span)
	}
}

func consumeClaudeUser(open map[string]*agent.Span, msg map[string]any, target string) {
	content, _ := msg["content"].([]any)
	if content == nil {
		if text := flattenContent(msg["content"]); text != "" {
			return
		}
	}
	for _, block := range content {
		m := asMap(block)
		if m == nil || asString(m["type"]) != "tool_result" {
			continue
		}
		id := asString(m["tool_use_id"])
		span := open[id]
		if span == nil {
			continue
		}
		output := flattenContent(m["content"])
		errText := ""
		if isErr, _ := m["is_error"].(bool); isErr {
			errText = output
			if errText == "" {
				errText = "tool_result is_error"
			}
		}
		span.IsTarget = span.IsTarget || skillReferenced(target, span.Name, span.ToolInput, output)
		closeSpan(span, output, errText)
		delete(open, id)
	}
}
