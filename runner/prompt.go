package runner

import (
	"fmt"
	"strings"
)

func externalPrompt(task Task, skillFile, layoutHint string) string {
	var b strings.Builder
	b.WriteString(task.SystemPrompt)
	b.WriteString("\n\n")
	b.WriteString("这是一次 skill 评测。请只在当前工作目录内读写文件，不要修改工作目录之外的内容。\n")
	if task.TargetSkill != "" {
		b.WriteString(fmt.Sprintf("目标 skill 名称：%s。\n", task.TargetSkill))
	}
	if skillFile != "" {
		b.WriteString("请先阅读并遵守该 skill 文件：\n")
		b.WriteString(skillFile)
		b.WriteString("\n")
	}
	if layoutHint != "" {
		b.WriteString(layoutHint)
		b.WriteString("\n")
	}
	b.WriteString("完成后在最终回复里用如下标记列出产物相对路径，没有产物就留空：\n")
	b.WriteString("<artifacts>\nrelative/path\n</artifacts>\n\n")
	b.WriteString("用户任务：\n")
	b.WriteString(task.UserPrompt)
	b.WriteString("\n")
	return b.String()
}
