package server

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"skill-eval/agent"
	"skill-eval/eval"
)

// Record 是前端可直接消费的评测记录。
// 单次评测使用 report + trace；对比评测使用 compareReport。
// status / runner / error 是在原有 EvalRecord 上追加的字段。
type Record struct {
	ID            string              `json:"id"`
	Type          string              `json:"type"`
	CreatedAt     time.Time           `json:"createdAt"`
	Status        string              `json:"status"`
	Runner        string              `json:"runner,omitempty"`
	Error         string              `json:"error,omitempty"`
	LogPath       string              `json:"logPath,omitempty"`
	Report        *eval.Report        `json:"report,omitempty"`
	Trace         *agent.Trace        `json:"trace,omitempty"`
	CompareReport *eval.CompareReport `json:"compareReport,omitempty"`
}

type Store struct {
	dir string
	mu  sync.Mutex
}

func NewStore(dataDir string) (*Store, error) {
	if dataDir == "" {
		dataDir = "logs"
	}
	dir := filepath.Join(dataDir, "evals")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("创建评测目录失败: %w", err)
	}
	return &Store{dir: dir}, nil
}

func (s *Store) path(id string) (string, error) {
	if !validID(id) {
		return "", fmt.Errorf("非法评测 id")
	}
	return filepath.Join(s.dir, id+".json"), nil
}

func validID(id string) bool {
	return id != "" && !strings.Contains(id, "..") && !strings.ContainsAny(id, `/\`)
}

func (s *Store) Save(rec *Record) error {
	if rec == nil {
		return fmt.Errorf("记录为空")
	}
	path, err := s.path(rec.ID)
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(rec, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化评测记录失败: %w", err)
	}
	tmp := path + ".tmp"
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("写入评测记录失败: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("保存评测记录失败: %w", err)
	}
	return nil
}

func (s *Store) Get(id string) (*Record, error) {
	path, err := s.path(id)
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("评测不存在: %s", id)
		}
		return nil, err
	}
	var rec Record
	if err := json.Unmarshal(data, &rec); err != nil {
		return nil, fmt.Errorf("解析评测记录失败: %w", err)
	}
	return &rec, nil
}

func (s *Store) List() ([]*Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return nil, err
	}
	var out []*Record
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") || strings.HasSuffix(entry.Name(), ".tmp") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(s.dir, entry.Name()))
		if err != nil {
			continue
		}
		var rec Record
		if err := json.Unmarshal(data, &rec); err != nil {
			continue
		}
		out = append(out, &rec)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].CreatedAt.After(out[j].CreatedAt)
	})
	return out, nil
}
