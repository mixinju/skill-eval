package tool

import (
    "context"
    "fmt"

    "skill-eval/skill"
)

type UseSkill struct {
    skills map[string]*skill.Skill
}

func NewUseSkill(skills map[string]*skill.Skill) *UseSkill {
    return &UseSkill{skills: skills}
}

func (u *UseSkill) Exec(ctx context.Context, params map[string]any) (string, error) {
    skillName, ok := params["skill_name"].(string)
    if !ok {
        return "", fmt.Errorf("skill_name parameter is required")
    }

    s, ok := u.skills[skillName]
    if !ok {
        available := make([]string, 0, len(u.skills))
        for name := range u.skills {
            available = append(available, name)
        }
        return "", fmt.Errorf("skill %q not found, available skills: %v", skillName, available)
    }

    return fmt.Sprintf("# Skill: %s\n\n%s", s.Name, s.Content), nil
}

func (u *UseSkill) GetTools() []Tool {
    skillList := make([]string, 0, len(u.skills))
    for name := range u.skills {
        skillList = append(skillList, name)
    }

    description := fmt.Sprintf("选择一个技能加载到上下文中。可用技能: %v", skillList)

    params := map[string]any{
        "type": "object",
        "properties": map[string]any{
            "skill_name": map[string]any{
                "type":        "string",
                "description": "要使用的技能名称",
            },
        },
        "required": []string{"skill_name"},
    }
    return []Tool{
        NewBaseToolInfo("use_skill", description, params, u.Exec),
    }
}
