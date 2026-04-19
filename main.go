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

    "github.com/openai/openai-go/v3"
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
        //os.Exit(1)
    }

    casesPath := "/Users/mixinju/Desktop/skill-eval/data/case/case.json"
    var skillAPath, skillBPath string
    model := "glm-5"
    baseURL := "https://newapi.sankuai.com/v1"
    apiKey := "sk-89mOFvu7We1ctc76YcQk3QuWnrUbXQ9zp3svMFBExqtz2YP1"
    maxInters := 100
    outputDir := "./eval-output"

    fmt.Println(outputDir)

    skillAPath = "/Users/mixinju/.claude/skills/pdf/SKILL.md"
    skillBPath = "/Users/mixinju/.automan/skills/minimax-pdf/SKILL.md"

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
        MaxInters:    maxInters,
        SystemPrompt: "你是一个智能助手，请根据用户的要求完成任务。完成后使用 finish 工具提交结果。",
    }

}
