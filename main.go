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
        SystemPrompt:  "你是一个写作专家，现在帮我写一个五百字的关于北京历史文化的文章，保存为md文件",
        Model:         "glm-5",
        MaxToolCount:  10,
        MaxIterations: 10,
    }

    // 注册默认工具
    agentConfig.RegistryDefaultTools()

    // 新建调度器
    o := agent.NewOrchestrator(&client, agentConfig)

    //运行智能体
    o.Run()
}
