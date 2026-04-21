package agent

import (
	"log"
	"os"
	"path/filepath"
	"skill-eval/tool"
)

type AgentConfig struct {
	Name          string
	Description   string
	SystemPrompt  string
	UserPrompt    string
	Model         string
	Tools         []tool.Tool
	Skills        []tool.Skill
	MaxToolCount  int
	MaxIterations int
}

type ConfigOpt func(*AgentConfig)

func NewAgentConfig(opts ...ConfigOpt) *AgentConfig {

	a := &AgentConfig{}
	for _, opt := range opts {
		opt(a)
	}
	return a
}

func WithName(name string) ConfigOpt {
	return func(c *AgentConfig) {
		c.Name = name
	}
}

func WithDescription(description string) ConfigOpt {
	return func(c *AgentConfig) {
		c.Description = description
	}
}

func WithSystemPrompt(prompt string) ConfigOpt {
	return func(c *AgentConfig) {
		c.SystemPrompt = prompt
	}
}

func WithModel(model string) ConfigOpt {
	return func(c *AgentConfig) {
		c.Model = model
	}
}

func WithTools(tools ...tool.Tool) ConfigOpt {
	return func(c *AgentConfig) {
		c.Tools = append(c.Tools, tools...)
	}
}

func WithMaxToolCount(max int) ConfigOpt {
	return func(c *AgentConfig) {
		c.MaxToolCount = max
	}
}

func WithMaxIterations(max int) ConfigOpt {
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

	fs := tool.NewFileSystem([]string{"./workplace"}, nil, 4)
	tools = append(tools, fs.GetTools()...)

	bash := tool.NewBash("./workplace", 5)
	tools = append(tools, bash.GetTools()...)

	finish := tool.NewFinish()
	tools = append(tools, finish.GetTools()...)

	weather := tool.NewGetWeather()
	tools = append(tools, weather.GetTools()...)

	useSkill := tool.NewUseSkill()
	tools = append(tools, useSkill.GetTools()...)

	a.Tools = tools
}

// RegistrySkills 加载SKILL
// 默认从.claude/skills目录下加载所有的目录
func (a *AgentConfig) RegistrySkills() {

	claudeSkillDir := "~/.claude/skills"

	entries, err := os.ReadDir(claudeSkillDir)
	if err != nil {
		log.Printf("[WARN] 加载Claude 技能文件夹失败")
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		s, err := tool.NewSkill(filepath.Join(claudeSkillDir, entry.Name()))
		if err != nil {
			log.Printf("[WARN] 加载SKILL失败: %v", err)
			continue
		}
		a.Skills = append(a.Skills, s)
	}

}
