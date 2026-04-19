package tool

import (
    "bytes"
    "fmt"
    "os"
    "path/filepath"

    "gopkg.in/yaml.v3"
)

// Skill 表示一个解析后的技能，包含 YAML Front Matter 元数据和 Markdown 正文
type Skill struct {
    Name        string `yaml:"name" json:"name"`
    Description string `yaml:"description" json:"description"`
    Version     string `yaml:"version,omitempty" json:"version,omitempty"`

    // 以下字段不从 YAML 解析，而是从文件内容和目录结构填充
    Content   string            `yaml:"-" json:"content"`             // Markdown 正文（不含 Front Matter）
    FilePath  string            `yaml:"-" json:"filePath"`            // SKILL.md 的绝对路径
    BaseDir   string            `yaml:"-" json:"baseDir"`             // 技能所在目录的绝对路径
    Resources map[string]string `yaml:"-" json:"resources,omitempty"` // 资源文件: 文件名 -> 绝对路径
}

// Load 从指定的路径加载技能，将内容赋值给s
// 要求 s.FilePath 已经设置为 SKILL.md 的绝对路径
func (s *Skill) Load() error {
    if s.FilePath == "" {
        return fmt.Errorf("skill FilePath is empty")
    }

    data, err := os.ReadFile(s.FilePath)
    if err != nil {
        return fmt.Errorf("read skill file %s: %w", s.FilePath, err)
    }

    // 设置 BaseDir 为 SKILL.md 所在目录
    s.BaseDir = filepath.Dir(s.FilePath)

    // 解析 YAML Front Matter 和 Markdown 正文
    frontMatter, content, err := parseFrontMatter(data)
    if err != nil {
        return fmt.Errorf("parse front matter from %s: %w", s.FilePath, err)
    }

    if err := yaml.Unmarshal(frontMatter, s); err != nil {
        return fmt.Errorf("unmarshal front matter: %w", err)
    }

    s.Content = string(content)

    // 扫描 BaseDir 下的资源文件（排除 SKILL.md 自身）
    if err := s.loadResources(); err != nil {
        return fmt.Errorf("load resources: %w", err)
    }

    return nil
}

// Resource 加载资源文件，返回文件内容
func (s *Skill) Resource(name string) (string, error) {
    if s.Resources == nil {
        return "", fmt.Errorf("no resources loaded")
    }

    path, ok := s.Resources[name]
    if !ok {
        return "", fmt.Errorf("resource %q not found", name)
    }

    data, err := os.ReadFile(path)
    if err != nil {
        return "", fmt.Errorf("read resource %s: %w", path, err)
    }

    return string(data), nil
}

// loadResources 扫描 BaseDir 下的所有文件（递归），将非 SKILL.md 的文件记录到 Resources
func (s *Skill) loadResources() error {
    s.Resources = make(map[string]string)

    err := filepath.WalkDir(s.BaseDir, func(path string, d os.DirEntry, err error) error {
        if err != nil {
            return err
        }

        // 跳过目录本身
        if d.IsDir() {
            return nil
        }

        // 跳过 SKILL.md 自身
        if filepath.Base(path) == "SKILL.md" {
            return nil
        }

        // 计算相对路径作为 key
        relPath, err := filepath.Rel(s.BaseDir, path)
        if err != nil {
            return fmt.Errorf("get relative path for %s: %w", path, err)
        }

        s.Resources[relPath] = path
        return nil
    })

    return err
}

// parseFrontMatter 从 SKILL.md 内容中分离 YAML Front Matter 和 Markdown 正文
// SKILL.md 格式:
//
//	---
//	name: xxx
//	description: xxx
//	---
//	Markdown 正文
func parseFrontMatter(data []byte) (frontMatter []byte, content []byte, err error) {
    // Front Matter 以 "---" 包裹
    if !bytes.HasPrefix(data, []byte("---")) {
        return nil, nil, fmt.Errorf("skill file must start with YAML front matter (---)")
    }

    // 找到第二个 "---"
    endIdx := bytes.Index(data[3:], []byte("---"))
    if endIdx == -1 {
        return nil, nil, fmt.Errorf("missing closing --- in YAML front matter")
    }

    frontMatter = data[3 : endIdx+3]
    content = bytes.TrimSpace(data[endIdx+6:])
    return frontMatter, content, nil
}
