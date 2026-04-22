package eval

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"skill-eval/agent"

	"github.com/openai/openai-go/v3"
	"github.com/sirupsen/logrus"
)

type LLMJudgeScorer struct {
	Client *openai.Client
	Model  string
}

func NewLLMJudgeScorer(client *openai.Client, model string) *LLMJudgeScorer {
	return &LLMJudgeScorer{Client: client, Model: model}
}

func (s *LLMJudgeScorer) Name() string { return "llm_judge" }

func (s *LLMJudgeScorer) Score(trace *agent.Trace) Verdict {
	if len(trace.Artifacts) == 0 {
		return Verdict{Name: s.Name(), Pass: false, Score: 0, Reason: "无产物文件，跳过 LLM 评分"}
	}

	var contents []string
	for _, path := range trace.Artifacts {
		text, err := ExtractContent(path)
		if err != nil {
			logrus.Warnf("[LLMJudge] 提取文件内容失败 %s: %v", path, err)
			continue
		}
		contents = append(contents, fmt.Sprintf("=== 文件: %s ===\n%s", path, text))
	}

	if len(contents) == 0 {
		return Verdict{Name: s.Name(), Pass: false, Score: 0, Reason: "所有产物文件均无法提取内容"}
	}

	prompt := fmt.Sprintf(`你是一个评测助手。用户给 agent 的指令是：
			"%s"

			agent 产出的文件内容如下：
			%s

			请评估产出内容是否满足用户的指令要求。返回 JSON 格式：
			{"score": 0.0到1.0的浮点数, "reason": "评分理由"}

			只返回 JSON，不要其他内容。`, trace.UserPrompt, strings.Join(contents, "\n\n"))

	resp, err := s.Client.Chat.Completions.New(context.Background(), openai.ChatCompletionNewParams{
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage(prompt),
		},
		Model: s.Model,
	})

	if err != nil {
		logrus.Errorf("[LLMJudge] LLM 调用失败: %v", err)
		return Verdict{Name: s.Name(), Pass: false, Score: 0, Reason: "LLM 调用失败: " + err.Error()}
	}

	if len(resp.Choices) == 0 {
		return Verdict{Name: s.Name(), Pass: false, Score: 0, Reason: "LLM 返回为空"}
	}

	raw := resp.Choices[0].Message.Content

	var result struct {
		Score  float64 `json:"score"`
		Reason string  `json:"reason"`
	}
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		logrus.Warnf("[LLMJudge] 解析 LLM 返回失败: %v, raw: %s", err, raw)
		return Verdict{Name: s.Name(), Pass: false, Score: 0, Reason: "LLM 返回格式异常: " + raw}
	}

	return Verdict{
		Name:   s.Name(),
		Pass:   result.Score >= 0.6,
		Score:  result.Score,
		Reason: result.Reason,
	}
}
