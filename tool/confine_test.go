package tool

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestFileSystemRejectsEscape(t *testing.T) {
	ws := t.TempDir()
	fs := NewFileSystem([]string{ws}, nil, time.Second)
	fs.SetWorkspace(ws)
	outside := filepath.Join(filepath.Dir(ws), "secret.txt")
	if err := os.WriteFile(outside, []byte("nope"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := fs.ReadFile(context.Background(), map[string]any{"path": outside}); err == nil {
		t.Fatal("绝对路径越界应被拒绝")
	}
	if _, err := fs.ReadFile(context.Background(), map[string]any{"path": "../secret.txt"}); err == nil {
		t.Fatal("相对路径越界应被拒绝")
	}
	if _, err := fs.WriteFile(context.Background(), map[string]any{"path": "note.txt", "content": "hello"}); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(ws, "note.txt"))
	if err != nil || string(got) != "hello" {
		t.Fatalf("工作区内写入失败: %v %q", err, got)
	}
}

func TestBashUsesWorkspace(t *testing.T) {
	ws := t.TempDir()
	bash := NewBash(ws, time.Second)
	out, err := bash.Exec(context.Background(), map[string]any{"command": "pwd"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, filepath.Base(ws)) {
		t.Fatalf("bash 没有在工作区执行: %s", out)
	}
}
