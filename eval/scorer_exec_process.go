package eval

import (
	"context"
	"encoding/json"
	"fmt"
	"skill-eval/agent"
	"strings"

	"github.com/openai/openai-go/v3"
	"github.com/sirupsen/logrus"
)

// ExecProcessScorer 执行过程评分器
// 直接解析 Trace 中的 Span 信息，传递给大模型进行评分
type ExecProcessScorer struct {
	Client *openai.Client
	Model  string
}

func NewExecProcessScorer(client *openai.Client, model string) *ExecProcessScorer {
	return &ExecProcessScorer{Client: client, Model: model}
}

func (s *ExecProcessScorer) Item() ScoreItem {
	return ScoreItem{
		Name: "执行过程",
		Desc: "评测执行过程",
	}
}

func (s *ExecProcessScorer) Score(trace ...*agent.Trace) Verdict {
	// 构建提示词
	prompt := s.buildPrompt(trace[0])

	// 调用 LLM 评分
	resp, err := s.Client.Chat.Completions.New(context.Background(), openai.ChatCompletionNewParams{
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(prompt),
		},
		Model: s.Model,
	})
	if err != nil {
		logrus.Errorf("[ExecProcessScorer] LLM 调用失败: %v", err)
		return Verdict{Info: s.Item(), Pass: false, Score: 0, Reason: "LLM 调用失败: " + err.Error()}
	}

	if len(resp.Choices) == 0 {
		return Verdict{Info: s.Item(), Pass: false, Score: 0, Reason: "LLM 返回为空"}
	}

	raw := resp.Choices[0].Message.Content

	// 解析返回结果
	var result struct {
		Score  float64 `json:"score"`
		Reason string  `json:"reason"`
	}
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		logrus.Warnf("[ExecProcessScorer] 解析 LLM 返回失败: %v, raw: %s", err, raw)
		return Verdict{Info: s.Item(), Pass: false, Score: 0, Reason: "LLM 返回格式异常: " + raw}
	}

	return Verdict{
		Info:   s.Item(),
		Pass:   result.Score >= 0.6,
		Score:  result.Score,
		Reason: result.Reason,
	}
}

func (s *ExecProcessScorer) buildPrompt(trace *agent.Trace) string {
	var sb strings.Builder

	sb.WriteString("你是一个智能体执行过程评估专家。请评估以下执行过程的质量。\n\n")

	sb.WriteString("## 任务信息\n")
	sb.WriteString(fmt.Sprintf("- 用户指令: %s\n", trace.UserPrompt))
	sb.WriteString(fmt.Sprintf("- 目标技能: %s\n", trace.TargetSkill))
	sb.WriteString(fmt.Sprintf("- 是否成功: %v\n", trace.Success))
	sb.WriteString(fmt.Sprintf("- 总迭代次数: %d\n", trace.Iterations))
	sb.WriteString(fmt.Sprintf("- 总 Token 消耗: %d\n", trace.TotalTokens))
	sb.WriteString(fmt.Sprintf("- 总执行时长: %d ms\n", trace.EndTime.Sub(trace.StartTime).Milliseconds()))
	sb.WriteString("\n")

	sb.WriteString("## Spans 执行链路\n")
	sb.WriteString("以下是完整的执行链路信息（JSON 数组）：\n\n")

	spansJSON, err := json.MarshalIndent(trace.Spans, "", "  ")
	if err != nil {
		logrus.Warnf("[ExecProcessScorer] 序列化 Spans 失败: %v", err)
		sb.WriteString("[]\n")
	} else {
		sb.WriteString("```json\n")
		sb.WriteString(string(spansJSON))
		sb.WriteString("\n```\n")
	}
	sb.WriteString("\n")

	sb.WriteString("## Span 类型说明\n")
	sb.WriteString("- `llm_call`: LLM 调用，包含 tokens、输入输出、finish_reason\n")
	sb.WriteString("- `llm_compress`: 消息压缩，用于长对话压缩\n")
	sb.WriteString("- `tool_call`: 工具调用，包含工具名、输入输出、是否命中目标技能、是否有错误\n")
	sb.WriteString("\n")

	sb.WriteString("## 评估标准\n")
	sb.WriteString("请从以下维度评估执行过程质量（每项 0-1 分）：\n\n")

	sb.WriteString("1. **效率**: 迭代次数是否合理，是否有过多的无效循环？\n")
	sb.WriteString("   - 0.0-0.3: 迭代次数过多，存在明显无效循环\n")
	sb.WriteString("   - 0.4-0.6: 迭代次数合理，但存在一些可优化的环节\n")
	sb.WriteString("   - 0.7-1.0: 迭代次数合理，执行高效\n\n")

	sb.WriteString("2. **资源消耗**: Token 消耗是否合理？\n")
	sb.WriteString("   - 0.0-0.3: Token 消耗过高，存在明显浪费\n")
	sb.WriteString("   - 0.4-0.6: Token 消耗适中，有优化空间\n")
	sb.WriteString("   - 0.7-1.0: Token 消耗合理，资源利用高效\n\n")

	sb.WriteString("3. **工具使用**: 工具调用是否有效，失败率如何？\n")
	sb.WriteString("   - 0.0-0.3: 工具调用失败率高，或工具选择不当\n")
	sb.WriteString("   - 0.4-0.6: 工具调用基本正确，存在少量失败\n")
	sb.WriteString("   - 0.7-1.0: 工具调用准确，失败率低\n\n")

	sb.WriteString("4. **技能命中**: 目标技能是否被正确调用？\n")
	sb.WriteString("   - 0.0: 目标技能未被调用\n")
	sb.WriteString("   - 0.5: 目标技能被调用但次数不足\n")
	sb.WriteString("   - 1.0: 目标技能被正确调用\n\n")

	sb.WriteString("## 输出格式\n")
	sb.WriteString("请返回 JSON 格式：\n")
	sb.WriteString("```json\n")
	sb.WriteString("{\n")
	sb.WriteString("  \"score\": 0.0到1.0的浮点数（四个维度的平均分），\n")
	sb.WriteString("  \"reason\": \"评分理由，简要说明各维度的表现\"\n")
	sb.WriteString("}\n")
	sb.WriteString("```\n")
	sb.WriteString("\n只返回 JSON，不要其他内容。")

	return sb.String()
}
