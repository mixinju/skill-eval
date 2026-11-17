package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"skill-eval/agent"
	"skill-eval/eval"
	"skill-eval/providers"
	"skill-eval/runner"
	"skill-eval/server"

	"github.com/sirupsen/logrus"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	var err error
	switch os.Args[1] {
	case "serve":
		err = cmdServe(os.Args[2:])
	case "run":
		err = cmdRun(os.Args[2:])
	case "runners":
		err = cmdRunners()
	case "compare":
		err = cmdCompare(os.Args[2:])
	case "demo":
		err = cmdDemo()
	case "help", "-h", "--help":
		usage()
		return
	default:
		usage()
		os.Exit(2)
	}
	if err != nil {
		logrus.Error(err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, `skill-eval 是 skill 评测后端。

用法:
  skill-eval runners
  skill-eval run --runner self|codex|claude-code --prompt "..." --model <model> [--skill <name>] [--skill-dir <dir>]
  skill-eval serve [--addr :8080]
  skill-eval compare <evalIdA> <evalIdB>
  skill-eval demo

配置见 docs/runners.md，HTTP 契约见 docs/api.md。
`)
}

func cmdServe(args []string) error {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	addr := fs.String("addr", envOr("EVAL_HTTP_ADDR", ":8080"), "监听地址")
	dataDir := fs.String("data-dir", envOr("EVAL_DATA_DIR", "logs"), "评测数据目录")
	if err := fs.Parse(args); err != nil {
		return err
	}
	srv, err := server.New(*dataDir, runner.NewDefaultRegistry())
	if err != nil {
		return err
	}
	return srv.Listen(*addr)
}

func cmdRun(args []string) error {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	kind := fs.String("runner", "self", "执行后端: self、codex、claude-code")
	prompt := fs.String("prompt", "", "用户任务")
	promptFile := fs.String("prompt-file", "", "从文件读取用户任务")
	system := fs.String("system", "", "系统提示词")
	model := fs.String("model", "", "模型名。self 必填")
	skill := fs.String("skill", "", "目标 skill 名称")
	skillDir := fs.String("skill-dir", os.Getenv("EVAL_DEFAULT_SKILL_DIR"), "技能根目录")
	name := fs.String("name", "skill-eval", "本次评测名称")
	maxIter := fs.Int("max-iterations", 10, "自研内核最大迭代次数")
	timeout := fs.Duration("timeout", 15*time.Minute, "超时")
	dataDir := fs.String("data-dir", envOr("EVAL_DATA_DIR", "logs"), "评测数据目录")
	if err := fs.Parse(args); err != nil {
		return err
	}
	userPrompt := strings.TrimSpace(*prompt)
	if *promptFile != "" {
		data, err := os.ReadFile(*promptFile)
		if err != nil {
			return fmt.Errorf("读取 prompt 文件失败: %w", err)
		}
		userPrompt = string(data)
	}
	if strings.TrimSpace(userPrompt) == "" {
		return fmt.Errorf("请通过 --prompt 或 --prompt-file 提供任务")
	}
	runnerType, err := runner.Normalize(*kind)
	if err != nil {
		return err
	}
	if runnerType == runner.TypeSelf && strings.TrimSpace(*model) == "" {
		return fmt.Errorf("self runner 需要 --model")
	}
	srv, err := server.New(*dataDir, runner.NewDefaultRegistry())
	if err != nil {
		return err
	}
	task := runner.Task{
		Name:          *name,
		SystemPrompt:  *system,
		UserPrompt:    userPrompt,
		Model:         *model,
		TargetSkill:   *skill,
		SkillDir:      *skillDir,
		MaxIterations: *maxIter,
		Timeout:       *timeout,
		DataDir:       *dataDir,
	}
	rec, runErr := srv.Run(context.Background(), runnerType, task)
	if rec != nil && rec.Report != nil {
		rec.Report.Print()
		fmt.Printf("Eval ID:     %s\n", rec.ID)
		if rec.LogPath != "" {
			fmt.Printf("Logs:        %s\n", rec.LogPath)
		}
	}
	if runErr != nil && (rec == nil || rec.Trace == nil || len(rec.Trace.Spans) == 0) {
		return runErr
	}
	if rec != nil && rec.Report != nil && !rec.Report.Pass {
		os.Exit(1)
	}
	return nil
}

func cmdRunners() error {
	reg := runner.NewDefaultRegistry()
	for _, item := range reg.Probe(context.Background()) {
		state := "不可用"
		if item.Available {
			state = "可用"
		}
		fmt.Printf("%-12s %-8s %s\n", item.Type, state, item.Detail)
		if item.Binary != "" {
			fmt.Printf("             binary=%s", item.Binary)
			if item.Version != "" {
				fmt.Printf(" version=%s", item.Version)
			}
			fmt.Println()
		}
	}
	return nil
}

func cmdCompare(args []string) error {
	fs := flag.NewFlagSet("compare", flag.ContinueOnError)
	dataDir := fs.String("data-dir", envOr("EVAL_DATA_DIR", "logs"), "评测数据目录")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 2 {
		return fmt.Errorf("用法: skill-eval compare <evalIdA> <evalIdB>")
	}
	srv, err := server.New(*dataDir, runner.NewDefaultRegistry())
	if err != nil {
		return err
	}
	rec, err := srv.Compare(fs.Arg(0), fs.Arg(1))
	if err != nil {
		return err
	}
	fmt.Println(rec.CompareReport.Conclusion)
	fmt.Printf("Compare ID:  %s\n", rec.ID)
	return nil
}

func cmdDemo() error {
	client := providers.NewClient()
	agentConfig := agent.NewAgentConfig(
		agent.WithName("天气小助手"),
		agent.WithDescription("天气小助手"),
		agent.WithSystemPrompt("天气小助手，当你完成任务时，需要调用finish工具"),
		agent.WithUserPrompt("查询下南京的天气，最后把结果保存为pdf文件"),
		agent.WithModel("glm-5"),
		agent.WithMaxToolCount(10),
		agent.WithMaxIterations(10),
	)
	agentConfig.RegistryDefaultTools()
	agentConfig.RegistrySkills()

	o := agent.NewOrchestrator(agent.NewOpenAICompleter(client), agentConfig)
	o.Tracer = agent.NewDefaultTracer(filepath.Join("logs", "traces"))
	o.SetTargetSkill("pdf")
	if err := o.Run(); err != nil {
		logrus.Error(err)
	}
	trace := o.GetTrace()
	if trace == nil {
		return fmt.Errorf("demo 没有产生 trace")
	}
	trace.Runner = "self"
	r := eval.Exec(trace, eval.BuildDefaultScorer())
	r.Print()
	if !r.Pass {
		os.Exit(1)
	}
	return nil
}

func envOr(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func init() {
	logrus.SetFormatter(&logrus.TextFormatter{
		ForceColors:     true,
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
		CallerPrettyfier: func(f *runtime.Frame) (string, string) {
			return "", fmt.Sprintf(" %s:%d", filepath.Base(f.File), f.Line)
		},
	})
	logrus.SetLevel(logrus.InfoLevel)
	if strings.EqualFold(os.Getenv("EVAL_LOG_LEVEL"), "debug") {
		logrus.SetLevel(logrus.DebugLevel)
	}
	logrus.SetReportCaller(true)
}
