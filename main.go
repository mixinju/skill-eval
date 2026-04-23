package main

import (
	"fmt"
	"path/filepath"
	"runtime"
	"skill-eval/agent"
	"skill-eval/providers"

	"github.com/sirupsen/logrus"
)

func main() {

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

	// 注册默认工具
	agentConfig.RegistryDefaultTools()
	agentConfig.RegistrySkills()

	// 新建调度器
	o := agent.NewOrchestrator(&client, agentConfig)
	o.Tracer = agent.NewDefaultTracer("./traces")
	o.SetTargetSkill("pdf")

	//运行智能体
	o.Run()

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
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetReportCaller(true)
}
