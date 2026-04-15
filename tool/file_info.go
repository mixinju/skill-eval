package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
)

type FileInfo struct{}

func (*FileInfo) params() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "File or directory path",
			},
		},
		"required": []string{"path"},
	}
}

func (*FileInfo) Stat(ctx context.Context, params map[string]any) (string, error) {
	path, ok := params["path"].(string)
	if !ok || path == "" {
		return "", fmt.Errorf("param 'path' is required")
	}

	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("stat failed: %w", err)
	}

	result := map[string]any{
		"name":      info.Name(),
		"path":      path,
		"is_dir":    info.IsDir(),
		"size":      info.Size(),
		"mode":      info.Mode().String(),
		"mod_time":  info.ModTime(),
		"extension": "",
	}
	if !info.IsDir() {
		ext := ""
		for i := len(path) - 1; i >= 0; i-- {
			if path[i] == '.' {
				ext = path[i:]
				break
			}
			if path[i] == '/' {
				break
			}
		}
		result["extension"] = ext
	}

	b, _ := json.Marshal(result)
	return string(b), nil
}

func (f *FileInfo) GetTools() []Tool {
	return []Tool{
		NewBaseToolInfo("file_info", "获取文件或目录元信息", f.params(), f.Stat),
	}
}
