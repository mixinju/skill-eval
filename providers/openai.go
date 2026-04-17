package providers

import (
	"context"
	"fmt"

	"skill-eval/tool"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
)

type OpenAIProvider struct {
	client openai.Client
}

func NewOpenAIProvider(baseURL, apiKey string) *OpenAIProvider {
	client := openai.NewClient(
		option.WithAPIKey(apiKey),
		option.WithBaseURL(baseURL),
	)
	return &OpenAIProvider{client: client}
}

func (p *OpenAIProvider) Chat(ctx context.Context, model string, messages []openai.ChatCompletionMessageParamUnion, tools []tool.Tool) (*openai.ChatCompletion, error) {
	params := openai.ChatCompletionNewParams{
		Messages: messages,
		Model:    model,
	}

	if len(tools) > 0 {
		var toolParams []openai.ChatCompletionToolUnionParam
		for _, t := range tools {
			toolParams = append(toolParams, t.ChatCompletionToolUnionParam())
		}
		params.Tools = toolParams
	}

	completion, err := p.client.Chat.Completions.New(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("llm chat failed: %w", err)
	}

	return completion, nil
}
