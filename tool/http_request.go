package tool

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type HTTPRequest struct{}

func (*HTTPRequest) params() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"method": map[string]any{
				"type":        "string",
				"description": "HTTP method: GET|POST|PUT|PATCH|DELETE",
			},
			"url": map[string]any{
				"type":        "string",
				"description": "Request URL",
			},
			"headers": map[string]any{
				"type":        "object",
				"description": "Request headers key-value",
			},
			"body": map[string]any{
				"type":        "string",
				"description": "Request body string",
			},
			"timeout_sec": map[string]any{
				"type":        "number",
				"description": "Timeout seconds, default 30",
			},
		},
		"required": []string{"method", "url"},
	}
}

func (*HTTPRequest) Do(ctx context.Context, params map[string]any) (string, error) {
	method, ok := params["method"].(string)
	if !ok || method == "" {
		return "", fmt.Errorf("param 'method' is required")
	}
	url, ok := params["url"].(string)
	if !ok || url == "" {
		return "", fmt.Errorf("param 'url' is required")
	}

	method = strings.ToUpper(method)
	body, _ := params["body"].(string)
	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewBufferString(body))
	if err != nil {
		return "", fmt.Errorf("build request failed: %w", err)
	}

	if headers, ok := params["headers"].(map[string]any); ok {
		for k, v := range headers {
			req.Header.Set(k, fmt.Sprintf("%v", v))
		}
	}

	timeout := 30 * time.Second
	if v, ok := params["timeout_sec"].(float64); ok && v > 0 {
		timeout = time.Duration(v) * time.Second
	}

	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response failed: %w", err)
	}

	result := map[string]any{
		"status_code": resp.StatusCode,
		"headers":     resp.Header,
		"body":        string(respBody),
	}
	b, _ := json.Marshal(result)
	return string(b), nil
}

func (h *HTTPRequest) GetTools() []Tool {
	return []Tool{
		NewBaseToolInfo("http_request", "发送 HTTP 请求", h.params(), h.Do),
	}
}
