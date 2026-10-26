package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type DefaultTracer struct {
	id         string
	trace      *Trace
	currentLLM *Span
	toolSpans  map[string]*Span
	outputDir  string
}

func NewDefaultTracer(outputDir string) *DefaultTracer {
	return &DefaultTracer{
		id:        uuid.New().String(),
		outputDir: outputDir,
		toolSpans: make(map[string]*Span),
	}
}

func (t *DefaultTracer) Id() string {
	return t.id
}

// SetID 在 Run 开始前指定 trace id，使评测记录与 trace 对齐。
func (t *DefaultTracer) SetID(id string) {
	if id != "" {
		t.id = id
	}
}

func (t *DefaultTracer) spanID() string {
	return uuid.NewString()
}

func (t *DefaultTracer) GetTrace() *Trace {
	return t.trace
}

func (t *DefaultTracer) OnEvent(event TraceEvent) {
	event.Timestamp = time.Now()

	switch event.Type {
	case EventRunStart:
		t.trace = &Trace{
			ID:            t.Id(),
			AgentName:     event.AgentName,
			Model:         event.Model,
			UserPrompt:    event.UserPrompt,
			TargetSkill:   event.TargetSkill,
			StartTime:     event.Timestamp,
			SchemaVersion: 1,
			Status:        "running",
			Spans:         []*Span{},
			Steps:         []Step{},
		}
		logrus.Infof("[Tracer] Run 开始: %s", t.trace.ID)

	case EventRunEnd:
		if t.trace == nil {
			return
		}
		t.trace.EndTime = event.Timestamp
		t.trace.Success = event.Success
		t.trace.Iterations = event.Iteration
		t.trace.TotalTokens = event.TotalTokens
		t.trace.FailureReason = event.Error
		t.trace.Error = event.Error
		switch {
		case event.Success:
			t.trace.Status = "succeeded"
		case strings.Contains(event.Error, "超时"):
			t.trace.Status = "timeout"
		case strings.Contains(event.Error, "取消"):
			t.trace.Status = "cancelled"
		default:
			t.trace.Status = "failed"
		}
		if t.trace.Spans == nil {
			t.trace.Spans = []*Span{}
		}
		t.trace.Steps = BuildSteps(t.trace.Spans)
		logrus.Infof("[Tracer] Run 结束: success=%v, status=%s, iterations=%d, tokens=%d",
			event.Success, t.trace.Status, event.Iteration, event.TotalTokens)
		t.Save()

	case EventLLMStart:
		span := &Span{
			SpanID:        t.spanID(),
			Kind:          SpanKindLLMCall,
			Name:          "chat_completion",
			Iteration:     event.Iteration,
			StartTime:     event.Timestamp,
			InputMessages: event.MessageCount,
			LLMInput:      clipText(event.LLMInput, 32*1024),
		}
		t.currentLLM = span
		t.appendSpan(span)

	case EventLLMEnd:
		if t.currentLLM == nil {
			return
		}
		t.currentLLM.EndTime = event.Timestamp
		t.currentLLM.Duration = event.Timestamp.Sub(t.currentLLM.StartTime).Milliseconds()
		t.currentLLM.TotalTokens = event.TotalTokens
		t.currentLLM.FinishReason = event.FinishReason
		t.currentLLM.LLMOutput = clipText(event.LLMOutput, 32*1024)
		t.currentLLM.Error = event.Error

	case EventLLMCompressStart:
		span := &Span{
			SpanID:        t.spanID(),
			Kind:          SpanKindLLMCompress,
			Name:          "compress",
			Iteration:     event.Iteration,
			StartTime:     event.Timestamp,
			InputMessages: event.MessageCount,
		}
		t.toolSpans["__compress__"] = span
		t.appendSpan(span)

	case EventLLMCompressEnd:
		span := t.toolSpans["__compress__"]
		if span == nil {
			return
		}
		span.EndTime = event.Timestamp
		span.Duration = event.Timestamp.Sub(span.StartTime).Milliseconds()
		span.TotalTokens = event.TotalTokens
		span.Error = event.Error
		delete(t.toolSpans, "__compress__")

	case EventToolStart, EventSkillStart:
		if t.trace == nil {
			return
		}
		parentID := ""
		if t.currentLLM != nil {
			parentID = t.currentLLM.SpanID
		}
		name := event.ToolName
		if event.Type == EventSkillStart && name == "" {
			name = "use_skill"
		}
		span := &Span{
			SpanID:    t.spanID(),
			ParentID:  parentID,
			Kind:      SpanKindToolCall,
			Name:      name,
			Iteration: event.Iteration,
			StartTime: event.Timestamp,
			ToolInput: event.ToolInput,
			IsTarget:  event.IsTarget,
		}
		if event.CallID != "" {
			t.toolSpans[event.CallID] = span
		}
		t.appendSpan(span)

	case EventSkillEnd:
		event.Type = EventToolEnd
		t.OnEvent(event)
		return

	case EventToolEnd:
		span, ok := t.toolSpans[event.CallID]
		if !ok {
			return
		}
		span.EndTime = event.Timestamp
		span.Duration = event.Timestamp.Sub(span.StartTime).Milliseconds()
		span.ToolOutput = clipText(event.ToolOutput, 16*1024)
		span.Error = event.Error

		// 结束时保留产物信息到最外层的Trace上。
		// EventToolEnd 不一定回带工具名，以 span 名称为准。
		if span.Name == "finish" && event.ToolOutput != "" {
			t.trace.ArtifactsAndResult = event.ToolOutput
		}
		delete(t.toolSpans, event.CallID)

	case EventNote:
		if t.trace != nil && event.Message != "" {
			t.trace.Notes = append(t.trace.Notes, event.Message)
		}
	}
}

func (t *DefaultTracer) appendSpan(span *Span) {
	if t.trace == nil {
		return
	}
	if t.trace.Spans == nil {
		t.trace.Spans = []*Span{}
	}
	t.trace.Spans = append(t.trace.Spans, span)
}

// Save 将当前 trace 写入 outputDir。Run 结束时会自动调用。
func (t *DefaultTracer) Save() {
	if t.trace == nil {
		return
	}

	if err := os.MkdirAll(t.outputDir, 0755); err != nil {
		logrus.Errorf("[Tracer] 创建目录失败: %v", err)
		return
	}

	filename := fmt.Sprintf("%s_%s.json",
		t.trace.ID,
		t.trace.StartTime.Format("20060102_150405"))
	path := filepath.Join(t.outputDir, filename)

	data, err := json.MarshalIndent(t.trace, "", "  ")
	if err != nil {
		logrus.Errorf("[Tracer] 序列化失败: %v", err)
		return
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		logrus.Errorf("[Tracer] 写入文件失败: %v", err)
		return
	}

	logrus.Infof("[Tracer] Trace 已保存: %s", path)
}
