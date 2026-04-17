package main

import (
    "context"
    "encoding/json"
    "fmt"
    "log"
    "os"
    "path/filepath"
    "time"

    "skill-eval/agent"
    "skill-eval/eval"
    "skill-eval/skill"
)

func main() {
    if len(os.Args) < 2 {
        fmt.Println("Usage: skill-eval <cases.json> [--skill-a path] [--skill-b path]")
        fmt.Println("  --skill-a    Skill A 的 SKILL.md 路径")
        fmt.Println("  --skill-b    Skill B 的 SKILL.md 路径（不指定则为无 skill 对照组）")
        fmt.Println("  --model      模型名称 (默认: glm-5)")
        fmt.Println("  --base-url   API 地址 (默认: https://open.bigmodel.cn/api/paas/v4)")
        fmt.Println("  --api-key    API Key")
        fmt.Println("  --max-iters  最大迭代次数 (默认: 10)")
        fmt.Println("  --output     输出目录 (默认: ./eval-output)")
        os.Exit(1)
    }

    casesPath := os.Args[1]
    var skillAPath, skillBPath string
    model := "glm-5"
    baseURL := "https://open.bigmodel.cn/api/paas/v4"
    apiKey := ""
    maxIters := 10
    outputDir := "./eval-output"

    for i := 2; i < len(os.Args); i++ {
        switch os.Args[i] {
        case "--skill-a":
            i++
            skillAPath = os.Args[i]
        case "--skill-b":
            i++
            skillBPath = os.Args[i]
        case "--model":
            i++
            model = os.Args[i]
        case "--base-url":
            i++
            baseURL = os.Args[i]
        case "--api-key":
            i++
            apiKey = os.Args[i]
        case "--max-iters":
            i++
            fmt.Sscanf(os.Args[i], "%d", &maxIters)
        case "--output":
            i++
            outputDir = os.Args[i]
        }
    }

    cases, err := eval.LoadCases(casesPath)
    if err != nil {
        log.Fatalf("Failed to load cases: %v", err)
    }
    fmt.Printf("Loaded %d cases\n", len(cases))

    var skillA *skill.Skill
    if skillAPath != "" {
        skillA = &skill.Skill{FilePath: skillAPath}
        if err := skillA.Load(); err != nil {
            log.Fatalf("Failed to load skill A: %v", err)
        }
        fmt.Printf("Skill A: %s\n", skillA.Name)
    }

    var skillB *skill.Skill
    if skillBPath != "" {
        skillB = &skill.Skill{FilePath: skillBPath}
        if err := skillB.Load(); err != nil {
            log.Fatalf("Failed to load skill B: %v", err)
        }
        fmt.Printf("Skill B: %s\n", skillB.Name)
    }

    agentA := &agent.Agent{
        Name:         "agent-a",
        Model:        model,
        BaseURL:      baseURL,
        APIKey:       apiKey,
        Skill:        skillA,
        MaxTokens:    100000,
        MaxInters:    maxIters,
        SystemPrompt: "你是一个智能助手，请根据用户的要求完成任务。完成后使用 finish 工具提交结果。",
    }

    agentB := &agent.Agent{
        Name:         "agent-b",
        Model:        model,
        BaseURL:      baseURL,
        APIKey:       apiKey,
        Skill:        skillB,
        MaxTokens:    100000,
        MaxInters:    maxIters,
        SystemPrompt: "你是一个智能助手，请根据用户的要求完成任务。完成后使用 finish 工具提交结果。",
    }

    timestamp := time.Now().Format("20060102-150405")
    runDir := filepath.Join(outputDir, timestamp)
    os.MkdirAll(runDir, 0755)

    logFile, err := os.Create(filepath.Join(runDir, "events.jsonl"))
    if err != nil {
        log.Fatalf("Failed to create log file: %v", err)
    }
    defer logFile.Close()

    onEvent := func(e agent.Event) {
        entry := map[string]any{
            "type":      e.Type,
            "iteration": e.Iteration,
            "data":      e.Data,
            "timestamp": time.Now().Format(time.RFC3339),
        }
        data, _ := json.Marshal(entry)
        logFile.Write(data)
        logFile.WriteString("\n")
    }

    pair := eval.EvalPair{AgentA: agentA, AgentB: agentB}
    runner := eval.NewRunner(runDir, onEvent)

    fmt.Printf("Starting evaluation: %d cases, output: %s\n", len(cases), runDir)
    result := runner.Run(context.Background(), pair, cases)

    fmt.Println("\n=== Evaluation Complete ===")
    fmt.Printf("Total cases: %d\n", len(result.Pairs))
    for _, p := range result.Pairs {
        fmt.Printf("  Case %s: A=%s(%d iters, %d tokens), B=%s(%d iters, %d tokens)\n",
            p.CaseID,
            p.ResultA.StopReason, p.ResultA.Iterations, p.ResultA.TokensUsed,
            p.ResultB.StopReason, p.ResultB.Iterations, p.ResultB.TokensUsed)
    }

    scorer := eval.NewScorer(baseURL, apiKey, model)
    scores, err := scorer.Score(context.Background(), result.Pairs, cases)
    if err != nil {
        log.Printf("Scoring failed: %v", err)
    } else {
        result.Scores = scores
        fmt.Println("\n=== LLM Scores (pending human review) ===")
        for _, s := range scores {
            fmt.Printf("  Case %s: A=%d, B=%d — %s\n", s.CaseID, s.ScoreA, s.ScoreB, s.Reason)
        }
    }

    reportPath := filepath.Join(runDir, "report.json")
    reportData, _ := json.MarshalIndent(result, "", "  ")
    os.WriteFile(reportPath, reportData, 0644)
    fmt.Printf("\nReport saved to: %s\n", reportPath)
}
