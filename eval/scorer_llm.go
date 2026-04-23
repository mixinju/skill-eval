package eval

import (
	"context"
	"encoding/json"
	"fmt"
	"skill-eval/tool"
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

func (s *LLMJudgeScorer) Item() ScoreItem {
	return ScoreItem{
		Name: "大模型评估产物结果",
		Desc: "由大模型评估生成的产物文件",
	}
}

func (s *LLMJudgeScorer) Score(trace *agent.Trace) Verdict {

	var f tool.FinishResult
	if err := json.Unmarshal([]byte(trace.ArtifactsAndResult), &f); err != nil {
		logrus.Warnf("ArtifactsAndResult 反序列化失败 %v", err.Error())
	}
	if len(f.Artifacts) == 0 {
		return Verdict{Info: s.Item(), Pass: false, Score: 0, Reason: "无产物文件，跳过 LLM 评分"}
	}

	var contents []string
	for _, path := range f.Artifacts {
		text, err := ExtractContent(path)
		if err != nil {
			logrus.Warnf("[LLMJudge] 提取文件内容失败 %s: %v", path, err)
			continue
		}
		contents = append(contents, fmt.Sprintf("=== 文件: %s ===\n%s", path, text))
	}

	if len(contents) == 0 {
		return Verdict{Info: s.Item(), Pass: false, Score: 0, Reason: "所有产物文件均无法提取内容"}
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
			openai.SystemMessage(prompt),
		},
		Model: s.Model,
	})

	if err != nil {
		logrus.Errorf("[LLMJudge] LLM 调用失败: %v", err)
		return Verdict{Info: s.Item(), Pass: false, Score: 0, Reason: "LLM 调用失败: " + err.Error()}
	}

	if len(resp.Choices) == 0 {
		return Verdict{Info: s.Item(), Pass: false, Score: 0, Reason: "LLM 返回为空"}
	}

	raw := resp.Choices[0].Message.Content

	var result struct {
		Score  float64 `json:"score"`
		Reason string  `json:"reason"`
	}
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		logrus.Warnf("[LLMJudge] 解析 LLM 返回失败: %v, raw: %s", err, raw)
		return Verdict{Info: s.Item(), Pass: false, Score: 0, Reason: "LLM 返回格式异常: " + raw}
	}

	return Verdict{
		Info:   s.Item(),
		Pass:   result.Score >= 0.6,
		Score:  result.Score,
		Reason: result.Reason,
	}
}
