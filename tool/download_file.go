package tool

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

type DownloadFile struct{}

func (*DownloadFile) params() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"url": map[string]any{
				"type":        "string",
				"description": "File URL",
			},
			"save_path": map[string]any{
				"type":        "string",
				"description": "Local save path",
			},
		},
		"required": []string{"url", "save_path"},
	}
}

func (*DownloadFile) Download(ctx context.Context, params map[string]any) (string, error) {
	url, ok := params["url"].(string)
	if !ok || url == "" {
		return "", fmt.Errorf("param 'url' is required")
	}
	savePath, ok := params["save_path"].(string)
	if !ok || savePath == "" {
		return "", fmt.Errorf("param 'save_path' is required")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("build request failed: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	if err := os.MkdirAll(filepath.Dir(savePath), 0o755); err != nil {
		return "", fmt.Errorf("create save dir failed: %w", err)
	}

	f, err := os.Create(savePath)
	if err != nil {
		return "", fmt.Errorf("create file failed: %w", err)
	}
	defer f.Close()

	n, err := io.Copy(f, resp.Body)
	if err != nil {
		return "", fmt.Errorf("save file failed: %w", err)
	}

	return fmt.Sprintf("downloaded %d bytes to %s", n, savePath), nil
}

func (d *DownloadFile) GetTools() []Tool {
	return []Tool{
		NewBaseToolInfo("download_file", "下载文件到本地", d.params(), d.Download),
	}
}
