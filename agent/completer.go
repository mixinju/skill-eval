package agent

import (
	"context"

	"github.com/openai/openai-go/v3"
)

// Completer 是自研 agent loop 使用的对话接口。
// 生产环境由 OpenAI 兼容客户端实现，测试可以注入脚本化实现。
type Completer interface {
	Complete(ctx context.Context, params openai.ChatCompletionNewParams) (*openai.ChatCompletion, error)
}

// OpenAICompleter 把 openai.Client 适配成 Completer。
type OpenAICompleter struct {
	Client openai.Client
}

func NewOpenAICompleter(client openai.Client) OpenAICompleter {
	return OpenAICompleter{Client: client}
}

func (c OpenAICompleter) Complete(ctx context.Context, params openai.ChatCompletionNewParams) (*openai.ChatCompletion, error) {
	return c.Client.Chat.Completions.New(ctx, params)
}
