package tool

import (
    "context"

    "github.com/openai/openai-go/v3"
)

// Tool 定义工具的动作
type Tool interface {
    Exec(ctx context.Context, prams map[string]any) (string, error)
    Name() string
    Description() string
    FunctionParameters() openai.FunctionParameters
    ChatCompletionToolUnionParam() openai.ChatCompletionToolUnionParam
}

// BaseToolInfo 承载tool的实体
type BaseToolInfo struct {
    name        string
    description string
    parameters  map[string]any
    execFunc    func(ctx context.Context, prams map[string]any) (string, error)
}

func (b *BaseToolInfo) Exec(ctx context.Context, prams map[string]any) (string, error) {
    return b.execFunc(ctx, prams)
}

func (b *BaseToolInfo) Name() string {
    return b.name
}

func (b *BaseToolInfo) Description() string {
    return b.description
}

func (b *BaseToolInfo) FunctionParameters() openai.FunctionParameters {
    return b.parameters
}

func (b *BaseToolInfo) ChatCompletionToolUnionParam() openai.ChatCompletionToolUnionParam {
    return openai.ChatCompletionToolUnionParam{
        OfFunction: &openai.ChatCompletionFunctionToolParam{Function: openai.FunctionDefinitionParam{
            Name:        b.Name(),
            Description: openai.String(b.Description()),
            Parameters:  b.FunctionParameters(),
        }},
    }
}

func NewBaseToolInfo(name string, description string, params map[string]any, f func(ctx context.Context, prams map[string]any) (string, error)) *BaseToolInfo {
    return &BaseToolInfo{
        name:        name,
        description: description,
        parameters:  params,
        execFunc:    f,
    }
}
