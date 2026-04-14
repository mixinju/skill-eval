package tool

import "github.com/openai/openai-go/v3"

type FileSystem struct {
}

func (f FileSystem) Prams() openai.FunctionParameters {
    //TODO implement me
    panic("implement me")
}

func (f FileSystem) Definition() openai.FunctionDefinitionParam {
    //TODO implement me
    panic("implement me")
}

func (f FileSystem) ToolParams() openai.ChatCompletionToolUnionParam {
    //TODO implement me
    panic("implement me")
}
