package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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

func (t *DefaultTracer) OnEvent(event TraceEvent) {
	event.Timestamp = time.Now()

	switch event.Type {
	case EventRunStart:
		t.trace = &Trace{
			ID:          t.Id(),
			AgentName:   event.AgentName,
			Model:       event.Model,
			UserPrompt:  event.UserPrompt,
			TargetSkill: event.TargetSkill,
			StartTime:   event.Timestamp,
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
		logrus.Infof("[Tracer] Run 结束: success=%v, iterations=%d, tokens=%d",
			event.Success, event.Iteration, event.TotalTokens)
		t.save()

	case EventLLMStart:
		span := &Span{
			SpanID:        t.Id(),
			Kind:          SpanKindLLMCall,
			Name:          "chat_completion",
			Iteration:     event.Iteration,
			StartTime:     event.Timestamp,
			InputMessages: event.MessageCount,
			LLMInput:      event.LLMInput,
		}
		t.currentLLM = span
		t.trace.Spans = append(t.trace.Spans, span)

	case EventLLMEnd:
		if t.currentLLM == nil {
			return
		}
		t.currentLLM.EndTime = event.Timestamp
		t.currentLLM.Duration = event.Timestamp.Sub(t.currentLLM.StartTime)
		t.currentLLM.TotalTokens = event.TotalTokens
		t.currentLLM.FinishReason = event.FinishReason
		t.currentLLM.LLMOutput = event.LLMOutput
		t.currentLLM.Error = event.Error

	case EventLLMCompressStart:
		span := &Span{
			SpanID:        t.Id(),
			Kind:          SpanKindLLMCompress,
			Name:          "compress",
			Iteration:     event.Iteration,
			StartTime:     event.Timestamp,
			InputMessages: event.MessageCount,
		}
		t.currentLLM = span
		t.trace.Spans = append(t.trace.Spans, span)

	case EventLLMCompressEnd:
		if t.currentLLM == nil {
			return
		}
		t.currentLLM.EndTime = event.Timestamp
		t.currentLLM.Duration = event.Timestamp.Sub(t.currentLLM.StartTime)
		t.currentLLM.TotalTokens = event.TotalTokens
		t.currentLLM.Error = event.Error
		t.currentLLM = nil

	case EventToolStart:
		parentID := ""
		if t.currentLLM != nil {
			parentID = t.currentLLM.SpanID
		}
		span := &Span{
			SpanID:    t.Id(),
			ParentID:  parentID,
			Kind:      SpanKindToolCall,
			Name:      event.ToolName,
			Iteration: event.Iteration,
			StartTime: event.Timestamp,
			ToolInput: event.ToolInput,
			IsTarget:  event.IsTarget,
		}
		t.toolSpans[event.CallID] = span
		t.trace.Spans = append(t.trace.Spans, span)

	case EventToolEnd:
		span, ok := t.toolSpans[event.CallID]
		if !ok {
			return
		}
		span.EndTime = event.Timestamp
		span.Duration = event.Timestamp.Sub(span.StartTime)
		span.ToolOutput = event.ToolOutput
		span.Error = event.Error
		delete(t.toolSpans, event.CallID)

	}
}

func (t *DefaultTracer) save() {
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
