package main

import (
    "skill-eval/agent"
    "skill-eval/providers"
)

func main() {

    client := providers.NewClient()

    agentConfig := agent.AgentConfig{
        Name:          "写作小能手",
        Description:   "写作专家",
        SystemPrompt:  "写作专家，当你完成任务时，需要调用finish工具",
        UserPrompt:    "帮我查询下北京的天气，然后推荐一些穿衣，最后把结果保存为pdf文件",
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
