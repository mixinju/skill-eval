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
        NewBaseToolInfo("use_skill", "选择使用一个SKILL，并加载SKILL具体的内容", u.Params(), u.Exec),
    }

    return tools
}
