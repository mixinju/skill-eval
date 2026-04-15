package tool

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type SearchText struct{}

func (*SearchText) params() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "Root path to search",
			},
			"keyword": map[string]any{
				"type":        "string",
				"description": "Text keyword",
			},
			"max_results": map[string]any{
				"type":        "number",
				"description": "Max matched files, default 20",
			},
		},
		"required": []string{"path", "keyword"},
	}
}

func (*SearchText) Search(ctx context.Context, params map[string]any) (string, error) {
	root, ok := params["path"].(string)
	if !ok || root == "" {
		return "", fmt.Errorf("param 'path' is required")
	}
	keyword, ok := params["keyword"].(string)
	if !ok || keyword == "" {
		return "", fmt.Errorf("param 'keyword' is required")
	}

	maxResults := 20
	if v, ok := params["max_results"].(float64); ok && v > 0 {
		maxResults = int(v)
	}

	matches := make([]string, 0, maxResults)
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if len(matches) >= maxResults {
			return filepath.SkipAll
		}
		if d.IsDir() {
			name := d.Name()
			if name == ".git" || name == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}

		content, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}
		if strings.Contains(string(content), keyword) {
			matches = append(matches, path)
		}
		return nil
	})
	if err != nil && err != filepath.SkipAll {
		return "", fmt.Errorf("search failed: %w", err)
	}

	if len(matches) == 0 {
		return "no matches", nil
	}
	return strings.Join(matches, "\n"), nil
}

func (s *SearchText) GetTools() []Tool {
	return []Tool{
		NewBaseToolInfo("search_text", "在目录中搜索文本关键字", s.params(), s.Search),
	}
}
