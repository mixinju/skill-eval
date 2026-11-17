package runner

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"skill-eval/tool"
)

type runPaths struct {
	ID         string
	Root       string
	Workspace  string
	LogDir     string
	StdoutLog  string
	StderrLog  string
	PromptPath string
}

func dataDir() string {
	if dir := os.Getenv("EVAL_DATA_DIR"); dir != "" {
		return dir
	}
	return "logs"
}

func newRunPaths(task Task) (runPaths, error) {
	if err := safeID(task.ID); err != nil {
		return runPaths{}, err
	}
	root := filepath.Join(task.DataDir, "runs", task.ID)
	ws := task.Workspace
	if ws == "" {
		ws = filepath.Join(root, "workspace")
	}
	logs := filepath.Join(root, "logs")
	if err := os.MkdirAll(ws, 0o755); err != nil {
		return runPaths{}, fmt.Errorf("创建工作区失败: %w", err)
	}
	if err := os.MkdirAll(logs, 0o755); err != nil {
		return runPaths{}, fmt.Errorf("创建日志目录失败: %w", err)
	}
	return runPaths{
		ID:         task.ID,
		Root:       root,
		Workspace:  ws,
		LogDir:     logs,
		StdoutLog:  filepath.Join(logs, "stdout.log"),
		StderrLog:  filepath.Join(logs, "stderr.log"),
		PromptPath: filepath.Join(ws, ".skill-eval", "prompt.txt"),
	}, nil
}

func safeID(id string) error {
	if id == "" || strings.Contains(id, "..") || strings.ContainsAny(id, `/\`) {
		return fmt.Errorf("非法任务 id")
	}
	return nil
}

func writePromptFile(path, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0o644)
}

func writeLogs(paths runPaths, stdout, stderr string) {
	_ = os.WriteFile(paths.StdoutLog, []byte(stdout), 0o644)
	_ = os.WriteFile(paths.StderrLog, []byte(stderr), 0o644)
}

// stageSkills 把技能复制到 runner 期望的目录，避免评测改写用户原来的技能库。
func stageSkills(src, dest string) error {
	info, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("读取技能目录失败: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("技能目录不是文件夹: %s", src)
	}
	if _, err := os.Stat(filepath.Join(src, "SKILL.md")); err == nil {
		return copyDir(src, filepath.Join(dest, filepath.Base(src)))
	}
	entries, err := os.ReadDir(src)
	if err != nil {
		return fmt.Errorf("读取技能目录失败: %w", err)
	}
	copied := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		from := filepath.Join(src, entry.Name())
		if _, err := os.Stat(filepath.Join(from, "SKILL.md")); err != nil {
			continue
		}
		if err := copyDir(from, filepath.Join(dest, entry.Name())); err != nil {
			return err
		}
		copied++
	}
	if copied == 0 {
		return fmt.Errorf("技能目录 %s 下没有包含 SKILL.md 的子目录", src)
	}
	return nil
}

func copyDir(src, dest string) error {
	return filepath.WalkDir(src, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dest, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		if d.Type()&os.ModeSymlink != 0 {
			return nil
		}
		return copyFile(path, target)
	})
}

func copyFile(src, dest string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	out, err := os.OpenFile(dest, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}

func findSkillFile(root, name string) string {
	if root == "" || name == "" {
		return ""
	}
	direct := filepath.Join(root, name, "SKILL.md")
	if _, err := os.Stat(direct); err == nil {
		return direct
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return ""
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		path := filepath.Join(root, entry.Name(), "SKILL.md")
		skill, err := tool.NewSkill(path)
		if err == nil && skill.Name == name {
			return path
		}
	}
	return ""
}

func gitInit(dir string) error {
	cmd := exec.Command("git", "init")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("初始化工作区 git 仓库失败: %w (%s)。Codex 要求在 git 仓库中执行", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func snapshot(root string) map[string]struct{} {
	files := map[string]struct{}{}
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil || ignoreRel(rel) {
			return nil
		}
		files[rel] = struct{}{}
		return nil
	})
	return files
}

func newFiles(root string, before map[string]struct{}) []string {
	var out []string
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil || ignoreRel(rel) {
			return nil
		}
		if _, ok := before[rel]; ok {
			return nil
		}
		out = append(out, rel)
		return nil
	})
	return out
}

func ignoreRel(rel string) bool {
	rel = filepath.ToSlash(rel)
	top := rel
	if i := strings.IndexByte(rel, '/'); i >= 0 {
		top = rel[:i]
	}
	switch top {
	case ".git", ".agents", ".claude", ".codex", ".skill-eval":
		return true
	default:
		return false
	}
}
