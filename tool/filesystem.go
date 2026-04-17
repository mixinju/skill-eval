package tool

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type FileSystem struct {
	workspace string
}

func NewFileSystem(workspace string) *FileSystem {
	return &FileSystem{workspace: workspace}
}

func (f *FileSystem) resolvePath(path string) (string, error) {
	if !filepath.IsAbs(path) {
		path = filepath.Join(f.workspace, path)
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	absWorkspace, err := filepath.Abs(f.workspace)
	if err != nil {
		return "", err
	}
	if !strings.HasPrefix(absPath, absWorkspace) {
		return "", fmt.Errorf("access to path %s is not allowed, must be within workspace %s", absPath, absWorkspace)
	}
	return absPath, nil
}

func (f *FileSystem) ReadFile(ctx context.Context, params map[string]any) (string, error) {
	path, ok := params["path"].(string)
	if !ok {
		return "", fmt.Errorf("path parameter is required")
	}
	resolved, err := f.resolvePath(path)
	if err != nil {
		return "", err
	}
	content, err := os.ReadFile(resolved)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

func (f *FileSystem) WriteFile(ctx context.Context, params map[string]any) (string, error) {
	path, ok := params["path"].(string)
	if !ok {
		return "", fmt.Errorf("path parameter is required")
	}
	content, ok := params["content"].(string)
	if !ok {
		return "", fmt.Errorf("content parameter is required")
	}
	resolved, err := f.resolvePath(path)
	if err != nil {
		return "", err
	}
	dir := filepath.Dir(resolved)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	if err := os.WriteFile(resolved, []byte(content), 0644); err != nil {
		return "", err
	}
	return fmt.Sprintf("Successfully wrote %d bytes to %s", len(content), resolved), nil
}

func (f *FileSystem) EditFile(ctx context.Context, params map[string]any) (string, error) {
	path, ok := params["path"].(string)
	if !ok {
		return "", fmt.Errorf("path parameter is required")
	}
	oldStr, ok := params["old_string"].(string)
	if !ok {
		return "", fmt.Errorf("old_string parameter is required")
	}
	newStr, ok := params["new_string"].(string)
	if !ok {
		return "", fmt.Errorf("new_string parameter is required")
	}
	resolved, err := f.resolvePath(path)
	if err != nil {
		return "", err
	}
	content, err := os.ReadFile(resolved)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}
	fileContent := string(content)
	if !strings.Contains(fileContent, oldStr) {
		return "", fmt.Errorf("old_string not found in file")
	}
	occurrences := strings.Count(fileContent, oldStr)
	newContent := strings.ReplaceAll(fileContent, oldStr, newStr)
	if err := os.WriteFile(resolved, []byte(newContent), 0644); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}
	return fmt.Sprintf("Successfully replaced %d occurrence(s) in %s", occurrences, resolved), nil
}

func (f *FileSystem) ListDir(ctx context.Context, params map[string]any) (string, error) {
	path, ok := params["path"].(string)
	if !ok {
		path = "."
	}
	resolved, err := f.resolvePath(path)
	if err != nil {
		return "", err
	}
	entries, err := os.ReadDir(resolved)
	if err != nil {
		return "", err
	}
	var result []string
	for _, entry := range entries {
		prefix := ""
		if entry.IsDir() {
			prefix = "[DIR] "
		}
		result = append(result, prefix+entry.Name())
	}
	return strings.Join(result, "\n"), nil
}

func (f *FileSystem) GetTools() []Tool {
	readParams := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "文件路径（相对于工作目录或绝对路径）",
			},
		},
		"required": []string{"path"},
	}
	writeParams := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "文件路径",
			},
			"content": map[string]any{
				"type":        "string",
				"description": "文件内容",
			},
		},
		"required": []string{"path", "content"},
	}
	editParams := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "文件路径",
			},
			"old_string": map[string]any{
				"type":        "string",
				"description": "要替换的原始字符串",
			},
			"new_string": map[string]any{
				"type":        "string",
				"description": "替换后的新字符串",
			},
		},
		"required": []string{"path", "old_string", "new_string"},
	}
	listParams := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "目录路径，默认为工作目录",
			},
		},
	}
	return []Tool{
		NewBaseToolInfo("read_file", "读取文件内容", readParams, f.ReadFile),
		NewBaseToolInfo("write_file", "写入文件内容", writeParams, f.WriteFile),
		NewBaseToolInfo("edit_file", "精确字符串替换编辑文件", editParams, f.EditFile),
		NewBaseToolInfo("list_dir", "列出目录内容", listParams, f.ListDir),
	}
}
