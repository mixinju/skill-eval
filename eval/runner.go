package eval

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"skill-eval/agent"
	"skill-eval/providers"
	"skill-eval/tool"

	"github.com/openai/openai-go/v3"
)

type EvalPair struct {
	AgentA *agent.Agent
	AgentB *agent.Agent
}

type PairResult struct {
	CaseID  string
	ResultA *agent.RunResult
	ResultB *agent.RunResult
}

type EvalResult struct {
	Pairs  []PairResult
	Scores []Score
}

type Runner struct {
	Orchestrator *agent.Orchestrator
	OnEvent      agent.EventHandler
	OutputDir    string
}

func NewRunner(outputDir string, onEvent agent.EventHandler) *Runner {
	return &Runner{
		Orchestrator: agent.NewOrchestrator(),
		OnEvent:      onEvent,
		OutputDir:    outputDir,
	}
}

func (r *Runner) Run(ctx context.Context, pair EvalPair, cases []Case) *EvalResult {
	var results []PairResult

	for _, c := range cases {
		fmt.Printf("Running case: %s (%s)\n", c.Name, c.ID)

		wsA := filepath.Join(r.OutputDir, c.ID, "a")
		wsB := filepath.Join(r.OutputDir, c.ID, "b")
		os.MkdirAll(wsA, 0755)
		os.MkdirAll(wsB, 0755)

		var wg sync.WaitGroup
		var resultA, resultB *agent.RunResult

		chatA := makeChatFunc(pair.AgentA)
		chatB := makeChatFunc(pair.AgentB)

		wg.Add(2)
		go func() {
			defer wg.Done()
			resultA = r.Orchestrator.Run(ctx, pair.AgentA, c.Input, wsA, r.wrapEvent("A", c.ID), chatA)
		}()
		go func() {
			defer wg.Done()
			resultB = r.Orchestrator.Run(ctx, pair.AgentB, c.Input, wsB, r.wrapEvent("B", c.ID), chatB)
		}()
		wg.Wait()

		results = append(results, PairResult{CaseID: c.ID, ResultA: resultA, ResultB: resultB})

		fmt.Printf("Case %s done: A=%s(%d iters), B=%s(%d iters)\n",
			c.ID, resultA.StopReason, resultA.Iterations, resultB.StopReason, resultB.Iterations)
	}

	return &EvalResult{Pairs: results}
}

func (r *Runner) wrapEvent(label string, caseID string) agent.EventHandler {
	return func(e agent.Event) {
		if r.OnEvent != nil {
			e.Data = map[string]any{
				"label":   label,
				"case_id": caseID,
				"data":    e.Data,
			}
			r.OnEvent(e)
		}
	}
}

func makeChatFunc(ag *agent.Agent) agent.ChatFunc {
	provider := providers.NewOpenAIProvider(ag.BaseURL, ag.APIKey)
	return func(ctx context.Context, messages []openai.ChatCompletionMessageParamUnion, tools []tool.Tool) (*openai.ChatCompletion, error) {
		return provider.Chat(ctx, ag.Model, messages, tools)
	}
}
