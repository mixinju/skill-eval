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
    return map[string]any{}
}

func (u *UseSkill) Exec(ctx context.Context, params map[string]any) (string, error) {
    return "", nil
}

func (u *UseSkill) GetTools() []Tool {

    tools := []Tool{
        NewBaseToolInfo("use_skill", "Select a skill to use for the current task. This loads the full skill content into the context.", u.Params(), u.Exec),
    }

    return tools
}
