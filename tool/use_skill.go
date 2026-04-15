package tool

import "context"

type UseSkill struct {
}

type useSkillResult struct {
	Success bool
	Name    string
	Content string
	Message string
}

func (*UseSkill) Params() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"name": map[string]any{
				"type":        "string",
				"description": "Skill name to use",
			},
			"task": map[string]any{
				"type":        "string",
				"description": "Task for the selected skill",
			},
		},
		"required": []string{"name", "task"},
	}
}

func (u *UseSkill) Exec(ctx context.Context, params map[string]any) (string, error) {
	name, _ := params["name"].(string)
	task, _ := params["task"].(string)
	return "skill placeholder invoked: " + name + " task: " + task, nil
}

func (u *UseSkill) GetTools() []Tool {

	tools := []Tool{
		NewBaseToolInfo("use_skill", "Select a skill to use for the current task. This loads the full skill content into the context.", u.Params(), u.Exec),
	}

	return tools
}
