package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"strconv"

	"skill-eval/evaluator"
	"skill-eval/gateway"
	"skill-eval/providers"
)

func main() {
	if len(os.Args) < 2 {
		providers.Chat(nil)
		return
	}

	switch os.Args[1] {
	case "eval-single":
		if len(os.Args) < 5 {
			log.Fatalf("usage: go run . eval-single <suite.json> <skill_name> <report_path>")
		}
		runSingle(os.Args[2], os.Args[3], os.Args[4])
	case "eval-compare":
		if len(os.Args) < 6 {
			log.Fatalf("usage: go run . eval-compare <suite.json> <left_skill> <right_skill> <report_path>")
		}
		runCompare(os.Args[2], os.Args[3], os.Args[4], os.Args[5])
	case "eval-case":
		runCase()
	case "serve":
		runServer()
	default:
		providers.Chat(nil)
	}
}

func runSingle(suitePath string, skillName string, reportPath string) {
	suite, err := evaluator.LoadSuiteFromJSON(suitePath)
	if err != nil {
		log.Fatalf("load suite failed: %v", err)
	}
	runner := evaluator.NewRunner(evaluator.EvalConfig{
		MaxRounds:      8,
		UseDocker:      true,
		ProjectRootDir: mustGetwd(),
		KeepWorkDir:    os.Getenv("KEEP_WORKDIR") == "1",
	})
	report, err := runner.RunSingle(context.Background(), skillName, suite)
	if err != nil {
		log.Fatalf("run single eval failed: %v", err)
	}
	if err := evaluator.SaveReport(reportPath, report); err != nil {
		log.Fatalf("save report failed: %v", err)
	}
	log.Printf("single eval done, report=%s", reportPath)
}

func runCompare(suitePath string, leftSkill string, rightSkill string, reportPath string) {
	suite, err := evaluator.LoadSuiteFromJSON(suitePath)
	if err != nil {
		log.Fatalf("load suite failed: %v", err)
	}
	runner := evaluator.NewRunner(evaluator.EvalConfig{
		MaxRounds:      8,
		UseDocker:      true,
		ProjectRootDir: mustGetwd(),
		KeepWorkDir:    os.Getenv("KEEP_WORKDIR") == "1",
	})
	report, err := runner.RunCompare(context.Background(), leftSkill, rightSkill, suite)
	if err != nil {
		log.Fatalf("run compare eval failed: %v", err)
	}
	if err := evaluator.SaveReport(reportPath, report); err != nil {
		log.Fatalf("save report failed: %v", err)
	}
	log.Printf("compare eval done, report=%s", reportPath)
}

func runCase() {
	input := os.Getenv("EVAL_INPUT")
	if input == "" {
		log.Fatalf("EVAL_INPUT is required")
	}
	maxRounds := 8
	if v := os.Getenv("EVAL_MAX_ROUNDS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			maxRounds = n
		}
	}

	result, err := providers.ExecutePrompt(context.Background(), input, maxRounds)
	if err != nil && result.Error == "" {
		result.Error = err.Error()
	}
	b, marshalErr := json.Marshal(result)
	if marshalErr != nil {
		log.Fatalf("marshal eval-case result failed: %v", marshalErr)
	}
	_, _ = os.Stdout.Write(b)
}

func mustGetwd() string {
	wd, err := os.Getwd()
	if err != nil {
		return "."
	}
	return wd
}

func runServer() {
	addr := ":8080"
	if v := os.Getenv("EVAL_SERVER_ADDR"); v != "" {
		addr = v
	}
	s := gateway.NewServer()
	if err := s.Start(addr); err != nil {
		log.Fatalf("start server failed: %v", err)
	}
}
