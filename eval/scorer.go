package eval

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"skill-eval/agent"
	"skill-eval/providers"

	"github.com/openai/openai-go/v3"
)

type Score struct {
	CaseID string `json:"case_id"`
	ScoreA int    `json:"score_a"`
	ScoreB int    `json:"score_b"`
	Reason string `json:"reason"`
}

type Scorer struct {
	provider *providers.OpenAIProvider
	model    string
}

func NewScorer(baseURL, apiKey, model string) *Scorer {
	return &Scorer{
		provider: providers.NewOpenAIProvider(baseURL, apiKey),
		model:    model,
	}
}

func (s *Scorer) Score(ctx context.Context, pairs []PairResult, cases []Case) ([]Score, error) {
	caseMap := make(map[string]Case)
	for _, c := range cases {
		caseMap[c.ID] = c
	}

	var scores []Score
	for _, pair := range pairs {
		c := caseMap[pair.CaseID]
		score, err := s.scoreOne(ctx, c, pair)
		if err != nil {
			fmt.Printf("Warning: scoring case %s failed: %v\n", pair.CaseID, err)
			scores = append(scores, Score{CaseID: pair.CaseID, ScoreA: -1, ScoreB: -1, Reason: err.Error()})
			continue
		}
		scores = append(scores, score)
	}
	return scores, nil
}

func (s *Scorer) scoreOne(ctx context.Context, c Case, pair PairResult) (Score, error) {
	artifactsA := readArtifacts(pair.ResultA)
	artifactsB := readArtifacts(pair.ResultB)

	prompt := fmt.Sprintf(`你是一个评测专家，请对比两个 Agent 的输出结果进行评分。

## 任务要求
%s

## 预期结果
%s

## Agent A 的输出
最终回复: %s
产物文件:
%s

## Agent B 的输出
最终回复: %s
产物文件:
%s

请对两个 Agent 的表现分别打分（1-10分），并给出评分理由。
请严格按照以下 JSON 格式返回:
{"score_a": 8, "score_b": 6, "reason": "评分理由"}`,
		c.Input,
		c.Expected,
		pair.ResultA.FinalOutput,
		artifactsA,
		pair.ResultB.FinalOutput,
		artifactsB,
	)

	messages := []openai.ChatCompletionMessageParamUnion{
		openai.UserMessage(prompt),
	}

	completion, err := s.provider.Chat(ctx, s.model, messages, nil)
	if err != nil {
		return Score{}, fmt.Errorf("llm scoring failed: %w", err)
	}

	if len(completion.Choices) == 0 {
		return Score{}, fmt.Errorf("llm returned empty choices")
	}

	var score Score
	score.CaseID = c.ID

	content := completion.Choices[0].Message.Content
	start := strings.Index(content, "{")
	end := strings.LastIndex(content, "}")
	if start >= 0 && end > start {
		content = content[start : end+1]
	}

	var result struct {
		ScoreA int    `json:"score_a"`
		ScoreB int    `json:"score_b"`
		Reason string `json:"reason"`
	}
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		return Score{}, fmt.Errorf("parse scoring result: %w", err)
	}

	score.ScoreA = result.ScoreA
	score.ScoreB = result.ScoreB
	score.Reason = result.Reason
	return score, nil
}

func readArtifacts(result *agent.RunResult) string {
	if result == nil || len(result.Artifacts) == 0 {
		return "(无产物文件)"
	}

	var parts []string
	for _, path := range result.Artifacts {
		content, err := os.ReadFile(path)
		if err != nil {
			parts = append(parts, fmt.Sprintf("--- %s ---\n[读取失败: %s]", filepath.Base(path), err))
			continue
		}
		if len(content) > 4096 {
			parts = append(parts, fmt.Sprintf("--- %s ---\n%s\n...(截断, 共 %d 字节)", filepath.Base(path), string(content[:4096]), len(content)))
		} else {
			parts = append(parts, fmt.Sprintf("--- %s ---\n%s", filepath.Base(path), string(content)))
		}
	}
	return strings.Join(parts, "\n\n")
}
