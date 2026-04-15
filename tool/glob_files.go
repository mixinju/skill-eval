package tool

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
)

type GlobFiles struct{}

func (*GlobFiles) params() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"pattern": map[string]any{
				"type":        "string",
				"description": "Glob pattern, e.g. ./tool/*.go",
			},
		},
		"required": []string{"pattern"},
	}
}

func (*GlobFiles) Match(ctx context.Context, params map[string]any) (string, error) {
	pattern, ok := params["pattern"].(string)
	if !ok || pattern == "" {
		return "", fmt.Errorf("param 'pattern' is required")
	}

	matches, err := filepath.Glob(pattern)
	if err != nil {
		return "", fmt.Errorf("glob failed: %w", err)
	}
	if len(matches) == 0 {
		return "no matches", nil
	}
	return strings.Join(matches, "\n"), nil
}

func (g *GlobFiles) GetTools() []Tool {
	return []Tool{
		NewBaseToolInfo("glob_files", "按 glob pattern 搜索文件", g.params(), g.Match),
	}
}
