package agent

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/openai/openai-go/v3"
	"github.com/sirupsen/logrus"
)

func (o *Orchestrator) maxAttempts() int {
	if o.MaxAttempts > 0 {
		return o.MaxAttempts
	}
	if raw := strings.TrimSpace(os.Getenv("EVAL_LLM_RETRIES")); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 {
			return n
		}
	}
	return 3
}

func (o *Orchestrator) complete(ctx context.Context, params openai.ChatCompletionNewParams) (*openai.ChatCompletion, error) {
	attempts := o.maxAttempts()
	delay := o.RetryBase
	if delay <= 0 {
		delay = 200 * time.Millisecond
	}

	var last error
	for attempt := 1; attempt <= attempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		resp, err := o.Chat.Complete(ctx, params)
		if err == nil {
			return resp, nil
		}
		last = err
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) || !isTransient(err) || attempt == attempts {
			break
		}
		msg := fmt.Sprintf("大模型调用失败，准备第 %d 次重试: %v", attempt, err)
		logrus.Warn(msg)
		o.emit(TraceEvent{Type: EventNote, Iteration: o.Context.CurrentIteration, Message: msg})
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, ctx.Err()
		case <-timer.C:
		}
		delay *= 2
		if delay > 5*time.Second {
			delay = 5 * time.Second
		}
	}
	if last == nil {
		last = fmt.Errorf("未知错误")
	}
	return nil, last
}

func isTransient(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	keys := []string{
		"429", "rate limit", "too many requests",
		"timeout", "temporarily", "unavailable",
		"502", "503", "504",
		"connection reset", "connection refused", "broken pipe", "eof",
	}
	for _, k := range keys {
		if strings.Contains(s, k) {
			return true
		}
	}
	return false
}
