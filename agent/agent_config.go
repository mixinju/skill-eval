package agent

import "skill-eval/tool"

type AgentConfig struct {
    Name          string
    Description   string
    SystemPrompt  string
    Model         string
    Tools         []tool.Tool
    MaxToolCount  int
    MaxIterations int
}

type AgentConfigOpt func(*AgentConfig)

func NewAgentConfig(opts ...AgentConfigOpt) *AgentConfig {

    a := &AgentConfig{}
    for _, opt := range opts {
        opt(a)
    }
    return a
}

func WithName(name string) AgentConfigOpt {
    return func(c *AgentConfig) {
        c.Name = name
    }
}

func WithDescription(description string) AgentConfigOpt {
    return func(c *AgentConfig) {
        c.Description = description
    }
}

func WithSystemPrompt(prompt string) AgentConfigOpt {
    return func(c *AgentConfig) {
        c.SystemPrompt = prompt
    }
}

func WithModel(model string) AgentConfigOpt {
    return func(c *AgentConfig) {
        c.Model = model
    }
}

func WithTools(tools ...tool.Tool) AgentConfigOpt {
    return func(c *AgentConfig) {
        c.Tools = append(c.Tools, tools...)
    }
}

func WithMaxToolCount(max int) AgentConfigOpt {
    return func(c *AgentConfig) {
        c.MaxToolCount = max
    }
}

func WithMaxIterations(max int) AgentConfigOpt {
    return func(c *AgentConfig) {
        c.MaxIterations = max
    }
}

func (a *AgentConfig) RegistryTool(tool ...tool.Tool) bool {
    if len(tool) >= a.MaxToolCount {
        return false
    }
    a.Tools = append(a.Tools, tool...)

    return true
}

func (a *AgentConfig) RegistryDefaultTools() {

    var tools []tool.Tool

    fs := tool.NewFileSystem(nil, nil, 4)
    tools = append(tools, fs.GetTools()...)

    bash := tool.NewBash("./workplace", 5)
    tools = append(tools, bash.GetTools()...)

    finish := tool.NewFinish()
    tools = append(tools, finish.GetTools()...)

    weather := tool.NewGetWeather()
    tools = append(tools, weather.GetTools()...)

    a.Tools = tools
}
