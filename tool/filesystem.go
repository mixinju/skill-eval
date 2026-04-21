package tool

import (
	"context"
	"fmt"
	"os"

	"path/filepath"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

type FileSystem struct {
	allowedPaths []string
	deniedPaths  []string
	workspace    string // 工作区路径，用于配置文件更新
	timeout      time.Duration
}

func NewFileSystem(allowedPaths []string, deniedPaths []string, timeout time.Duration) *FileSystem {

	now := time.Now()

	defaultWorkspace := filepath.Join("/Users/mixinju/Desktop/skill-eval-workplace/", now.Format("2006-01-02"))
	return &FileSystem{
		allowedPaths: allowedPaths,
		deniedPaths:  deniedPaths,
		timeout:      timeout,
		workspace:    defaultWorkspace,
	}
}

// ReadFile 读取文件
func (f *FileSystem) ReadFile(ctx context.Context, params map[string]any) (string, error) {
	path, ok := params["path"].(string)
	if !ok {
		return "", fmt.Errorf("path parameter is required")
	}

	ctx, cancelFunc := context.WithTimeout(ctx, f.timeout)
	defer cancelFunc()
	// 检查路径权限
	if !f.isAllowed(path) {
		return "", fmt.Errorf("access to path %s is not allowed", path)
	}

	path = filepath.Join(f.workspace, path)
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	return string(content), nil
}

// WriteFile 写入文件
func (f *FileSystem) WriteFile(ctx context.Context, params map[string]any) (string, error) {
	path, ok := params["path"].(string)
	if !ok {
		return "", fmt.Errorf("path parameter is required")
	}

	content, ok := params["content"].(string)
	if !ok {
		return "", fmt.Errorf("content parameter is required")
	}

	// 检查路径权限
	if !f.isAllowed(path) {
		return "", fmt.Errorf("access to path %s is not allowed", path)
	}

	// 确保目录存在
	localPath := filepath.Join(f.workspace, path)

	dir := filepath.Dir(localPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}

	// 写入文件
	if err := os.WriteFile(localPath, []byte(content), 0644); err != nil {
		return "", err
	}

	logrus.Infof("文件保存成功:%s", path)
	return fmt.Sprintf("Successfully wrote %d bytes to %s", len(content), path), nil
}

// EditFile 编辑文件（精确字符串替换）
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

	// 检查路径权限
	if !f.isAllowed(path) {
		return "", fmt.Errorf("access to path %s is not allowed", path)
	}

	// 读取文件内容
	localPath := filepath.Join(f.workspace, path)
	content, err := os.ReadFile(localPath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	fileContent := string(content)

	// 检查旧字符串是否存在
	if !strings.Contains(fileContent, oldStr) {
		return "", fmt.Errorf("old_string not found in file. Please verify the exact text to replace")
	}

	// 计算替换次数
	occurrences := strings.Count(fileContent, oldStr)

	// 执行替换
	newContent := strings.ReplaceAll(fileContent, oldStr, newStr)

	// 写入文件
	if err := os.WriteFile(localPath, []byte(newContent), 0644); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	return fmt.Sprintf("Successfully replaced %d occurrence(s) in %s", occurrences, path), nil
}

// ListDir 列出目录
func (f *FileSystem) ListDir(ctx context.Context, params map[string]any) (string, error) {
	path, ok := params["path"].(string)
	if !ok {
		return "", fmt.Errorf("path parameter is required")
	}

	// 检查路径权限
	if !f.isAllowed(path) {
		return "", fmt.Errorf("access to path %s is not allowed", path)
	}

	path = filepath.Join(f.workspace, path)
	entries, err := os.ReadDir(path)
	if err != nil {
		return "", err
	}

	var result []string
	for _, entry := range entries {
		info := ""
		if entry.IsDir() {
			info = "[DIR] "
		}
		result = append(result, info+entry.Name())
	}

	return strings.Join(result, "\n"), nil
}

// UpdateConfig 更新配置文件
func (f *FileSystem) UpdateConfig(ctx context.Context, params map[string]any) (string, error) {
	fileType, ok := params["file"].(string)
	if !ok {
		return "", fmt.Errorf("file parameter is required (identity, agents, soul, or user)")
	}

	content, ok := params["content"].(string)
	if !ok {
		return "", fmt.Errorf("content parameter is required")
	}

	// 验证文件类型
	validFiles := map[string]string{
		"identity": "IDENTITY.md",
		"agents":   "AGENTS.md",
		"soul":     "SOUL.md",
		"user":     "USER.md",
	}

	filename, valid := validFiles[fileType]
	if !valid {
		return "", fmt.Errorf("invalid file type: %s (must be one of: identity, agents, soul, user)", fileType)
	}

	// 构建完整路径
	if f.workspace == "" {
		return "", fmt.Errorf("workspace path is not configured")
	}
	path := filepath.Join(f.workspace, filename)

	// 确保目录存在
	if err := os.MkdirAll(f.workspace, 0755); err != nil {
		return "", fmt.Errorf("failed to create workspace directory: %w", err)
	}

	// 写入文件
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("failed to write config file: %w", err)
	}

	return fmt.Sprintf("Successfully updated %s\n\nThe changes will take effect in the next conversation.", filename), nil
}

// ReadConfig 读取配置文件
func (f *FileSystem) ReadConfig(ctx context.Context, params map[string]any) (string, error) {
	fileType, ok := params["file"].(string)
	if !ok {
		return "", fmt.Errorf("file parameter is required (identity, agents, soul, or user)")
	}

	// 验证文件类型
	validFiles := map[string]string{
		"identity": "IDENTITY.md",
		"agents":   "AGENTS.md",
		"soul":     "SOUL.md",
		"user":     "USER.md",
	}

	filename, valid := validFiles[fileType]
	if !valid {
		return "", fmt.Errorf("invalid file type: %s (must be one of: identity, agents, soul, user)", fileType)
	}

	// 构建完整路径
	if f.workspace == "" {
		return "", fmt.Errorf("workspace path is not configured")
	}
	path := filepath.Join(f.workspace, filename)

	// 读取文件
	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Sprintf("Config file %s does not exist yet. Use update_config to create it.", filename), nil
		}
		return "", err
	}

	return string(content), nil
}

// GetTools 返回 FileSystem 暴露的所有工具定义
func (f *FileSystem) GetTools() []Tool {
	return []Tool{
		NewBaseToolInfo("read_file", "读取文件内容", map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{
					"type":        "string",
					"description": "要读取的文件路径",
				},
			},
			"required": []string{"path"},
		}, f.ReadFile),
		NewBaseToolInfo("write_file", "写入文件内容、保存文件到本地", map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{
					"type":        "string",
					"description": "要写入的文件路径",
				},
				"content": map[string]any{
					"type":        "string",
					"description": "要写入的文件内容",
				},
			},
			"required": []string{"path", "content"},
		}, f.WriteFile),
		NewBaseToolInfo("edit_file", "编辑文件（精确字符串替换）", map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{
					"type":        "string",
					"description": "要编辑的文件路径",
				},
				"old_string": map[string]any{
					"type":        "string",
					"description": "要被替换的原始字符串",
				},
				"new_string": map[string]any{
					"type":        "string",
					"description": "替换后的新字符串",
				},
			},
			"required": []string{"path", "old_string", "new_string"},
		}, f.EditFile),
		NewBaseToolInfo("list_dir", "列出目录内容", map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{
					"type":        "string",
					"description": "要列出的目录路径",
				},
			},
			"required": []string{"path"},
		}, f.ListDir),
		NewBaseToolInfo("update_config", "更新配置文件（identity/agents/soul/user）", map[string]any{
			"type": "object",
			"properties": map[string]any{
				"file": map[string]any{
					"type":        "string",
					"enum":        []string{"identity", "agents", "soul", "user"},
					"description": "配置文件类型，可选值：identity, agents, soul, user",
				},
				"content": map[string]any{
					"type":        "string",
					"description": "配置文件内容",
				},
			},
			"required": []string{"file", "content"},
		}, f.UpdateConfig),
		NewBaseToolInfo("read_config", "读取配置文件（identity/agents/soul/user）", map[string]any{
			"type": "object",
			"properties": map[string]any{
				"file": map[string]any{
					"type":        "string",
					"enum":        []string{"identity", "agents", "soul", "user"},
					"description": "配置文件类型，可选值：identity, agents, soul, user",
				},
			},
			"required": []string{"file"},
		}, f.ReadConfig),
	}
}

// isAllowed 检查路径是否允许访问
func (f *FileSystem) isAllowed(path string) bool {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return false
	}

	// 检查拒绝列表（转换为绝对路径）
	for _, denied := range f.deniedPaths {
		absDenied, err := filepath.Abs(denied)
		if err == nil && strings.HasPrefix(absPath, absDenied) {
			return false
		}
	}

	// 如果没有允许列表，允许所有路径
	if len(f.allowedPaths) == 0 {
		return true
	}

	// 检查允许列表（转换为绝对路径）
	for _, allowed := range f.allowedPaths {
		absAllowed, err := filepath.Abs(allowed)
		if err == nil && strings.HasPrefix(absPath, absAllowed) {
			return true
		}
	}

	return false
}
