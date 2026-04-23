package eval

import (
	"context"
	"encoding/json"
	"fmt"
	"skill-eval/agent"
	"skill-eval/tool"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/shared"
	"github.com/sirupsen/logrus"
)

type ArtifactScorer struct {
	Client *openai.Client
	Model  string
}

func NewArtifactScorer(client *openai.Client, model string) *ArtifactScorer {
	return &ArtifactScorer{
		Client: client,
		Model:  model,
	}
}

func (s *ArtifactScorer) Item() ScoreItem {
	return ScoreItem{
		Name: "产物评分",
		Desc: "针对输出",
	}
}

func (s *ArtifactScorer) Score(traces ...*agent.Trace) (Verdict, error) {
	if len(traces) != 1 && len(traces) != 2 {
		return Verdict{}, fmt.Errorf("仅支持传入1或2个Trace，实际：%d", len(traces))
	}

	if len(traces) == 1 {
		return s.single(traces[0]), nil
	}
	return s.compare(traces[0], traces[1]), nil
}

func (s *ArtifactScorer) single(trace *agent.Trace) Verdict {
	var a tool.FinishResult
	if err := json.Unmarshal([]byte(trace.ArtifactsAndResult), &a); err != nil {
		logrus.Warnf("反序列化Artifacts失败: %v", err)
	}

	if len(a.Artifacts) <= 0 {
		return Verdict{Info: s.Item(), Pass: false, Score: 0, Reason: "未产出任何文件"}
	}

	var message []openai.ChatCompletionMessageParamUnion

	message = append(message, openai.SystemMessage(
		`请你根据判断下当前的产物文件是否实现了用户的要求,返回格式为Json，返回 JSON 格式：{"score": 0.0到1.0的浮点数, "reason": "评分理由}`))
	message = append(message, openai.UserMessage(fmt.Sprintf("产物列表如下:[%v]\n", a.Artifacts)))
	message = append(message, openai.UserMessage(fmt.Sprintf("用户的输入如下：[%v] \n", trace.UserPrompt)))

	chat, err := s.Client.Chat.Completions.New(context.Background(), openai.ChatCompletionNewParams{
		Messages: message,
		Model:    s.Model,
		ResponseFormat: openai.ChatCompletionNewParamsResponseFormatUnion{
			OfJSONObject: &shared.ResponseFormatJSONObjectParam{},
		},
	})
	if err != nil {
		logrus.Warnf("调用大模型失败: %v", err)
		return Verdict{Info: s.Item(), Pass: false, Score: 0, Reason: "调用大模型失败: " + err.Error()}
	}

	if len(chat.Choices) == 0 {
		return Verdict{Info: s.Item(), Pass: false, Score: 0, Reason: "LLM 返回为空"}
	}

	raw := chat.Choices[0].Message.Content
	var result struct {
		Score  float64 `json:"score"`
		Reason string  `json:"reason"`
	}
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		logrus.Warnf("解析 LLM 返回失败: %v, raw: %s", err, raw)
		return Verdict{Info: s.Item(), Pass: false, Score: 0, Reason: "LLM 返回格式异常: " + raw}
	}

	return Verdict{
		Info:   s.Item(),
		Pass:   result.Score >= 0.6,
		Score:  result.Score,
		Reason: result.Reason,
	}
}

func (s *ArtifactScorer) compare(traceA, traceB *agent.Trace) Verdict {

	//TODO 暂时未实现
	panic("待实现")
}
