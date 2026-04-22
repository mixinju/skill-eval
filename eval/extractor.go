package eval

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var textExtensions = map[string]bool{
	".txt": true, ".md": true, ".csv": true, ".html": true,
	".py": true, ".json": true, ".yaml": true, ".yml": true,
	".xml": true, ".js": true, ".go": true, ".java": true,
	".sh": true, ".sql": true, ".log": true,
}

func ExtractContent(filePath string) (string, error) {
	ext := strings.ToLower(filepath.Ext(filePath))

	if textExtensions[ext] {
		data, err := os.ReadFile(filePath)
		if err != nil {
			return "", fmt.Errorf("读取文件失败: %w", err)
		}
		return string(data), nil
	}

	switch ext {
	case ".pdf":
		return runCommand("pdftotext", filePath, "-")
	case ".docx":
		return runCommand("pandoc", filePath, "-t", "plain")
	case ".xlsx":
		return runCommand("ssconvert", "--export-type=Gnumeric_stf:stf_csv", filePath, "fd://1")
	default:
		return "", fmt.Errorf("不支持的文件类型: %s", ext)
	}
}

func runCommand(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("执行 %s 失败: %w, stderr: %s", name, err, stderr.String())
	}
	return stdout.String(), nil
}
