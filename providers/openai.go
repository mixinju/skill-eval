package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"skill-eval/tool"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
)

const (
	baseURL = ""
	apiKey  = "sk-"
	model   = "glm-5"

	maxLLMRetry      = 3
	maxToolRetry     = 3
	defaultMaxRounds = 8
)

type ToolCallRecord struct {
	Name    string         `json:"name"`
	Args    map[string]any `json:"args"`
	Result  string         `json:"result"`
	Success bool           `json:"success"`
	Error   string         `json:"error,omitempty"`
}

type RunStep struct {
	Round        int              `json:"round"`
	AssistantRaw string           `json:"assistant_raw"`
	ToolCalls    []ToolCallRecord `json:"tool_calls,omitempty"`
}

type RunResult struct {
	Success       bool      `json:"success"`
	Input         string    `json:"input"`
	FinalResponse string    `json:"final_response"`
	TotalRounds   int       `json:"total_rounds"`
	TotalTokens   int64     `json:"total_tokens"`
	Steps         []RunStep `json:"steps"`
	Error         string    `json:"error,omitempty"`
}

type RunHooks struct {
	OnRoundStart func(round int)
	OnRoundStep  func(step RunStep)
	OnToolCall   func(round int, call ToolCallRecord)
	OnMessage    func(round int, role string, content string, meta map[string]any)
}

var (
	registryOnce sync.Once
	defaultReg   *tool.Registry
	defaultTools []openai.ChatCompletionToolUnionParam
)

func Chat(messages []openai.ChatCompletionMessageParamUnion) {
	_ = messages
	ctx := context.Background()
	result, err := ExecutePrompt(ctx, "今天北京的天气怎么样", defaultMaxRounds)
	if err != nil {
		log.Printf("执行失败: %v", err)
		return
	}
	log.Printf("执行成功 rounds=%d tokens=%d", result.TotalRounds, result.TotalTokens)
	log.Printf("最终结果: %s", result.FinalResponse)
}

func ExecutePrompt(ctx context.Context, input string, maxRounds int) (RunResult, error) {
	return ExecutePromptWithHooks(ctx, input, maxRounds, nil)
}

func ExecutePromptWithHooks(ctx context.Context, input string, maxRounds int, hooks *RunHooks) (RunResult, error) {
	if maxRounds <= 0 {
		maxRounds = defaultMaxRounds
	}

	client := openai.NewClient(
		option.WithAPIKey(resolveAPIKey()),
		option.WithBaseURL(baseURL),
	)

	messages := []openai.ChatCompletionMessageParamUnion{openai.UserMessage(input)}
	registry, tools := buildTools()
	if len(tools) == 0 {
		return RunResult{Input: input, Success: false, Error: "no tools registered"}, fmt.Errorf("no tools registered")
	}

	result := RunResult{
		Success:     false,
		Input:       input,
		Steps:       make([]RunStep, 0, maxRounds),
		TotalRounds: 0,
	}
	if hooks != nil && hooks.OnMessage != nil {
		hooks.OnMessage(0, "user", input, map[string]any{})
	}

	for round := 1; round <= maxRounds; round++ {
		if hooks != nil && hooks.OnRoundStart != nil {
			hooks.OnRoundStart(round)
		}
		chatCompletion, err := callLLMWithRetry(ctx, client, messages, tools)
		if err != nil {
			result.Error = err.Error()
			return result, err
		}
		result.TotalTokens += chatCompletion.Usage.TotalTokens
		result.TotalRounds = round

		if len(chatCompletion.Choices) == 0 {
			result.Error = "empty completion choices"
			return result, fmt.Errorf("empty completion choices")
		}

		assistantMessage := chatCompletion.Choices[0].Message
		step := RunStep{
			Round:        round,
			AssistantRaw: assistantMessage.RawJSON(),
			ToolCalls:    []ToolCallRecord{},
		}
		messages = append(messages, assistantMessage.ToParam())
		if hooks != nil && hooks.OnMessage != nil {
			hooks.OnMessage(round, "assistant", assistantMessage.Content, map[string]any{
				"assistant_raw": assistantMessage.RawJSON(),
			})
		}

		// 模型不再调用工具时，认为任务结束。
		if len(assistantMessage.ToolCalls) == 0 {
			result.Success = true
			result.FinalResponse = assistantMessage.Content
			result.Steps = append(result.Steps, step)
			return result, nil
		}

		for _, tc := range assistantMessage.ToolCalls {
			var args map[string]any
			if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
				step.ToolCalls = append(step.ToolCalls, ToolCallRecord{
					Name:    tc.Function.Name,
					Args:    map[string]any{"raw_arguments": tc.Function.Arguments},
					Result:  "",
					Success: false,
					Error:   fmt.Sprintf("parse args failed: %v", err),
				})
				continue
			}

			toolResult, execErr := execToolWithRetry(ctx, registry, tc.Function.Name, args)
			record := ToolCallRecord{
				Name:    tc.Function.Name,
				Args:    args,
				Result:  toolResult,
				Success: execErr == nil,
			}
			if execErr != nil {
				record.Error = execErr.Error()
				toolResult = fmt.Sprintf("tool=%s failed: %v", tc.Function.Name, execErr)
			}
			step.ToolCalls = append(step.ToolCalls, record)
			if hooks != nil && hooks.OnToolCall != nil {
				hooks.OnToolCall(round, record)
			}
			messages = append(messages, openai.ToolMessage(toolResult, tc.ID))
			if hooks != nil && hooks.OnMessage != nil {
				hooks.OnMessage(round, "tool", toolResult, map[string]any{
					"name":    tc.Function.Name,
					"success": record.Success,
					"error":   record.Error,
				})
			}
		}
		if hooks != nil && hooks.OnRoundStep != nil {
			hooks.OnRoundStep(step)
		}
		result.Steps = append(result.Steps, step)
	}

	result.Error = fmt.Sprintf("exceeded max rounds: %d", maxRounds)
	return result, fmt.Errorf("exceeded max rounds: %d", maxRounds)
}

func buildTools() (*tool.Registry, []openai.ChatCompletionToolUnionParam) {
	registryOnce.Do(func() {
		registry := tool.NewRegistry()

		weather := (&tool.GetWeather{}).GetTools()
		fs := tool.NewFileSystem(".", nil, nil).GetTools()
		shell := (&tool.Shell{}).GetTools()
		python := (&tool.Python{}).GetTools()
		http := (&tool.HTTPRequest{}).GetTools()
		jsonExtract := (&tool.JSONExtract{}).GetTools()
		searchText := (&tool.SearchText{}).GetTools()
		runPythonFile := (&tool.RunPythonFile{}).GetTools()
		downloadFile := (&tool.DownloadFile{}).GetTools()
		globFiles := (&tool.GlobFiles{}).GetTools()
		fileInfo := (&tool.FileInfo{}).GetTools()
		jsonValidate := (&tool.JSONValidate{}).GetTools()
		readJSON := (&tool.ReadJSON{}).GetTools()
		writeJSON := (&tool.WriteJSON{}).GetTools()
		useSkill := (&tool.UseSkill{}).GetTools()

		if err := registry.RegisterGroup(
			weather,
			fs,
			shell,
			python,
			http,
			jsonExtract,
			searchText,
			runPythonFile,
			downloadFile,
			globFiles,
			fileInfo,
			jsonValidate,
			readJSON,
			writeJSON,
			useSkill,
		); err != nil {
			log.Printf("注册工具失败: %v", err)
		}

		defaultReg = registry
		defaultTools = registry.BuildToolParams()
	})

	return defaultReg, defaultTools
}

func resolveAPIKey() string {
	if key := os.Getenv("OPENAI_API_KEY"); key != "" {
		return key
	}
	return apiKey
}

func callLLMWithRetry(
	ctx context.Context,
	client openai.Client,
	messages []openai.ChatCompletionMessageParamUnion,
	tools []openai.ChatCompletionToolUnionParam,
) (*openai.ChatCompletion, error) {
	var lastErr error
	for attempt := 1; attempt <= maxLLMRetry; attempt++ {
		resp, err := client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
			Messages: messages,
			Model:    model,
			Tools:    tools,
		})
		if err == nil {
			return resp, nil
		}
		lastErr = err
		time.Sleep(time.Duration(attempt) * 300 * time.Millisecond)
	}
	return nil, fmt.Errorf("llm调用失败，已重试%d次: %w", maxLLMRetry, lastErr)
}

func execToolWithRetry(ctx context.Context, registry *tool.Registry, name string, args map[string]any) (string, error) {
	var lastErr error
	for attempt := 1; attempt <= maxToolRetry; attempt++ {
		result, err := registry.Exec(ctx, name, args)
		if err == nil {
			return result, nil
		}
		lastErr = err
		time.Sleep(time.Duration(attempt) * 200 * time.Millisecond)
	}
	return "", fmt.Errorf("工具 %s 执行失败，已重试%d次: %w", name, maxToolRetry, lastErr)
}
