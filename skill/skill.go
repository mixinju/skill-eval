package skill

import (
    "bytes"
    "fmt"
    "os"
    "path/filepath"

    "gopkg.in/yaml.v3"
)

type Skill struct {
    Name        string `yaml:"name" json:"name"`
    Description string `yaml:"description" json:"description"`
    Version     string `yaml:"version,omitempty" json:"version,omitempty"`

    Content   string            `yaml:"-" json:"content"`
    FilePath  string            `yaml:"-" json:"filePath"`
    BaseDir   string            `yaml:"-" json:"baseDir"`
    Resources map[string]string `yaml:"-" json:"resources,omitempty"`
}

func (s *Skill) Load() error {
    if s.FilePath == "" {
        return fmt.Errorf("skill FilePath is empty")
    }

    data, err := os.ReadFile(s.FilePath)
    if err != nil {
        return fmt.Errorf("read skill file %s: %w", s.FilePath, err)
    }

    s.BaseDir = filepath.Dir(s.FilePath)

    frontMatter, content, err := parseFrontMatter(data)
    if err != nil {
        return fmt.Errorf("parse front matter from %s: %w", s.FilePath, err)
    }

    if err := yaml.Unmarshal(frontMatter, s); err != nil {
        return fmt.Errorf("unmarshal front matter: %w", err)
    }

    s.Content = string(content)

    if err := s.loadResources(); err != nil {
        return fmt.Errorf("load resources: %w", err)
    }

    return nil
}

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

func (s *Skill) loadResources() error {
    s.Resources = make(map[string]string)

    err := filepath.WalkDir(s.BaseDir, func(path string, d os.DirEntry, err error) error {
        if err != nil {
            return err
        }

        if d.IsDir() {
            return nil
        }

        if filepath.Base(path) == "SKILL.md" {
            return nil
        }

        relPath, err := filepath.Rel(s.BaseDir, path)
        if err != nil {
            return fmt.Errorf("get relative path for %s: %w", path, err)
        }

        s.Resources[relPath] = path
        return nil
    })

    return err
}

func parseFrontMatter(data []byte) (frontMatter []byte, content []byte, err error) {
    if !bytes.HasPrefix(data, []byte("---")) {
        return nil, nil, fmt.Errorf("skill file must start with YAML front matter (---)")
    }

    endIdx := bytes.Index(data[3:], []byte("---"))
    if endIdx == -1 {
        return nil, nil, fmt.Errorf("missing closing --- in YAML front matter")
    }

    frontMatter = data[3 : endIdx+3]
    content = bytes.TrimSpace(data[endIdx+6:])
    return frontMatter, content, nil
}
