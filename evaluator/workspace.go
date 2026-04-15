package evaluator

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func prepareIsolatedWorkspace(cfg EvalConfig, skillName string) (string, error) {
	projectRoot := cfg.ProjectRootDir
	if projectRoot == "" {
		wd, _ := os.Getwd()
		projectRoot = wd
	}
	base := cfg.WorkDirBase
	if base == "" {
		base = filepath.Join(projectRoot, ".runs")
	}

	runID := fmt.Sprintf("%s-%d", sanitizeName(skillName), time.Now().UnixNano())
	target := filepath.Join(base, runID)
	if err := os.MkdirAll(target, 0o755); err != nil {
		return "", fmt.Errorf("create workspace dir failed: %w", err)
	}

	if err := copyProjectTree(projectRoot, target); err != nil {
		return "", err
	}
	return target, nil
}

func cleanupWorkspace(path string, keep bool) error {
	if keep {
		return nil
	}
	if path == "" {
		return nil
	}
	return os.RemoveAll(path)
}

func copyProjectTree(srcRoot string, dstRoot string) error {
	return filepath.WalkDir(srcRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(srcRoot, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		if shouldSkip(rel) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		dstPath := filepath.Join(dstRoot, rel)
		if d.IsDir() {
			return os.MkdirAll(dstPath, 0o755)
		}
		return copyFile(path, dstPath)
	})
}

func shouldSkip(rel string) bool {
	p := filepath.ToSlash(rel)
	return strings.HasPrefix(p, ".git/") ||
		p == ".git" ||
		strings.HasPrefix(p, ".runs/") ||
		p == ".runs"
}

func copyFile(src string, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Chmod(0o644)
}

func sanitizeName(s string) string {
	if s == "" {
		return "skill"
	}
	s = strings.ToLower(s)
	replacer := strings.NewReplacer(" ", "-", "/", "-", "\\", "-", ":", "-", "@", "-", ".", "-")
	s = replacer.Replace(s)
	if s == "" {
		return "skill"
	}
	return s
}
