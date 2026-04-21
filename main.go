package main

import (
	"skill-eval/agent"
	"skill-eval/providers"
)

func main() {

	client := providers.NewClient()

	agentConfig := agent.AgentConfig{
		Name:          "天气小助手",
		Description:   "天气小助手",
		SystemPrompt:  "天气小助手，当你完成任务时，需要调用finish工具",
		UserPrompt:    "查询下南京的天气，最后把结果保存为pdf文件",
		Model:         "glm-5",
		MaxToolCount:  10,
		MaxIterations: 10,
	}

	// 注册默认工具
	agentConfig.RegistryDefaultTools()
	agentConfig.RegistrySkills()

	// 新建调度器
	o := agent.NewOrchestrator(&client, agentConfig)
	o.SetTargetSkill("pdf")

	//运行智能体
	o.Run()
}
