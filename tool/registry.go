package tool

import (
	"context"
	"fmt"

	"github.com/openai/openai-go/v3"
)

type Registry struct {
	tools map[string]Tool
}

func NewRegistry() *Registry {
	return &Registry{tools: make(map[string]Tool)}
}

func (r *Registry) Register(tools ...Tool) error {
	for _, t := range tools {
		if _, exists := r.tools[t.Name()]; exists {
			return fmt.Errorf("tool %q already registered", t.Name())
		}
		r.tools[t.Name()] = t
	}
	return nil
}

func (r *Registry) RegisterGroup(groups ...[]Tool) error {
	for _, g := range groups {
		if err := r.Register(g...); err != nil {
			return err
		}
	}
	return nil
}

func (r *Registry) Get(name string) (Tool, bool) {
	t, ok := r.tools[name]
	return t, ok
}

func (r *Registry) BuildToolParams() []openai.ChatCompletionToolUnionParam {
	toolParams := make([]openai.ChatCompletionToolUnionParam, 0, len(r.tools))
	for _, t := range r.tools {
		toolParams = append(toolParams, t.ChatCompletionToolUnionParam())
	}
	return toolParams
}

func (r *Registry) Exec(ctx context.Context, name string, params map[string]any) (string, error) {
	t, ok := r.Get(name)
	if !ok {
		return "", fmt.Errorf("tool %q not found", name)
	}
	return t.Exec(ctx, params)
}
