package tool

import "github.com/openai/openai-go/v3"

type Tool interface {
    Prams() openai.FunctionParameters
    Definition() openai.FunctionDefinitionParam
    ToolParams() openai.ChatCompletionToolUnionParam
}
