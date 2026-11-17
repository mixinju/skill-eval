package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"skill-eval/tool"
)

type artifactInfo struct {
	Path   string `json:"path"`
	Name   string `json:"name"`
	Exists bool   `json:"exists"`
	Size   int64  `json:"size,omitempty"`
}

type artifactsResponse struct {
	Result    string         `json:"result,omitempty"`
	Artifacts []artifactInfo `json:"artifacts"`
	Workspace string         `json:"workspace,omitempty"`
}

func (s *Server) getArtifacts(w http.ResponseWriter, r *http.Request) {
	rec, err := s.store.Get(r.PathValue("id"))
	if err != nil || rec.Trace == nil {
		writeError(w, http.StatusNotFound, "评测或 trace 不存在")
		return
	}
	writeJSON(w, http.StatusOK, collectArtifacts(rec))
}

func (s *Server) getArtifactFile(w http.ResponseWriter, r *http.Request) {
	rec, err := s.store.Get(r.PathValue("id"))
	if err != nil || rec.Trace == nil {
		writeError(w, http.StatusNotFound, "评测或 trace 不存在")
		return
	}
	rel := strings.TrimSpace(r.URL.Query().Get("path"))
	if rel == "" {
		writeError(w, http.StatusBadRequest, "缺少 path 参数")
		return
	}
	ws := strings.TrimSpace(rec.Trace.Workspace)
	if ws == "" {
		writeError(w, http.StatusNotFound, "这次评测没有工作区，无法读取产物文件")
		return
	}
	abs, err := resolveWorkspaceFile(ws, rel)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	data, err := os.ReadFile(abs)
	if err != nil {
		writeError(w, http.StatusNotFound, "产物文件不存在")
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filepath.Base(abs)))
	_, _ = w.Write(data)
}

func collectArtifacts(rec *Record) artifactsResponse {
	out := artifactsResponse{Artifacts: []artifactInfo{}}
	if rec == nil || rec.Trace == nil {
		return out
	}
	out.Workspace = rec.Trace.Workspace
	payload := parseFinishPayload(rec.Trace.ArtifactsAndResult)
	if payload.Result == "" && len(payload.Artifacts) == 0 {
		for i := len(rec.Trace.Spans) - 1; i >= 0; i-- {
			span := rec.Trace.Spans[i]
			if span == nil || string(span.Kind) != "tool_call" || span.Name != "finish" {
				continue
			}
			payload = parseFinishPayload(span.ToolOutput)
			if payload.Result == "" && len(payload.Artifacts) == 0 {
				payload = parseFinishPayload(span.ToolInput)
			}
			break
		}
	}
	out.Result = payload.Result
	seen := map[string]bool{}
	for _, path := range payload.Artifacts {
		path = strings.TrimSpace(path)
		if path == "" || seen[path] {
			continue
		}
		seen[path] = true
		info := artifactInfo{Path: path, Name: filepath.Base(path)}
		if abs, err := resolveWorkspaceFile(rec.Trace.Workspace, path); err == nil {
			if st, err := os.Stat(abs); err == nil && !st.IsDir() {
				info.Exists = true
				info.Size = st.Size()
			}
		}
		out.Artifacts = append(out.Artifacts, info)
	}
	return out
}

func parseFinishPayload(raw string) tool.FinishResult {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return tool.FinishResult{}
	}
	var fr tool.FinishResult
	if err := json.Unmarshal([]byte(raw), &fr); err != nil {
		return tool.FinishResult{}
	}
	return fr
}

func resolveWorkspaceFile(workspace, rel string) (string, error) {
	workspace = strings.TrimSpace(workspace)
	rel = strings.TrimSpace(rel)
	if workspace == "" {
		return "", fmt.Errorf("没有工作区")
	}
	if rel == "" {
		return "", fmt.Errorf("路径为空")
	}
	if filepath.IsAbs(rel) {
		return "", fmt.Errorf("只允许工作区内的相对路径")
	}
	clean := filepath.Clean(rel)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("非法产物路径")
	}
	root, err := filepath.Abs(workspace)
	if err != nil {
		return "", fmt.Errorf("工作区无效")
	}
	abs := filepath.Join(root, clean)
	abs, err = filepath.Abs(abs)
	if err != nil {
		return "", fmt.Errorf("产物路径无效")
	}
	relCheck, err := filepath.Rel(root, abs)
	if err != nil || strings.HasPrefix(relCheck, "..") {
		return "", fmt.Errorf("产物路径超出工作区")
	}
	return abs, nil
}
