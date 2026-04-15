package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"skill-eval/evaluator"
)

type TaskStatus string

const (
	TaskPending TaskStatus = "pending"
	TaskRunning TaskStatus = "running"
	TaskSuccess TaskStatus = "success"
	TaskFailed  TaskStatus = "failed"
)

type Task struct {
	ID        string                `json:"id"`
	Status    TaskStatus            `json:"status"`
	Mode      string                `json:"mode"`
	CreatedAt string                `json:"created_at"`
	UpdatedAt string                `json:"updated_at"`
	Error     string                `json:"error,omitempty"`
	Result    any                   `json:"result,omitempty"`
	Events    []evaluator.EvalEvent `json:"events,omitempty"`
}

type TaskManager struct {
	mu    sync.RWMutex
	tasks map[string]*Task
	subs  map[string]map[chan evaluator.EvalEvent]struct{}
}

func NewTaskManager() *TaskManager {
	return &TaskManager{
		tasks: map[string]*Task{},
		subs:  map[string]map[chan evaluator.EvalEvent]struct{}{},
	}
}

func (m *TaskManager) Create(mode string) *Task {
	id := fmt.Sprintf("task-%d", time.Now().UnixNano())
	now := time.Now().Format(time.RFC3339)
	t := &Task{
		ID:        id,
		Status:    TaskPending,
		Mode:      mode,
		CreatedAt: now,
		UpdatedAt: now,
		Events:    []evaluator.EvalEvent{},
	}
	m.mu.Lock()
	m.tasks[id] = t
	m.mu.Unlock()
	return t
}

func (m *TaskManager) Get(id string) (*Task, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	t, ok := m.tasks[id]
	return t, ok
}

func (m *TaskManager) UpdateStatus(id string, status TaskStatus, errMsg string, result any) {
	m.mu.Lock()
	defer m.mu.Unlock()
	t, ok := m.tasks[id]
	if !ok {
		return
	}
	t.Status = status
	t.UpdatedAt = time.Now().Format(time.RFC3339)
	t.Error = errMsg
	if result != nil {
		t.Result = result
	}
}

func (m *TaskManager) Publish(event evaluator.EvalEvent) {
	m.mu.Lock()
	t, ok := m.tasks[event.TaskID]
	if ok {
		t.Events = append(t.Events, event)
		t.UpdatedAt = time.Now().Format(time.RFC3339)
	}
	subs := m.subs[event.TaskID]
	chs := make([]chan evaluator.EvalEvent, 0, len(subs))
	for ch := range subs {
		chs = append(chs, ch)
	}
	m.mu.Unlock()

	for _, ch := range chs {
		select {
		case ch <- event:
		default:
		}
	}
}

func (m *TaskManager) Subscribe(taskID string) (chan evaluator.EvalEvent, func()) {
	ch := make(chan evaluator.EvalEvent, 64)
	m.mu.Lock()
	if _, ok := m.subs[taskID]; !ok {
		m.subs[taskID] = map[chan evaluator.EvalEvent]struct{}{}
	}
	m.subs[taskID][ch] = struct{}{}
	m.mu.Unlock()

	cancel := func() {
		m.mu.Lock()
		defer m.mu.Unlock()
		if _, ok := m.subs[taskID]; ok {
			delete(m.subs[taskID], ch)
		}
		close(ch)
	}
	return ch, cancel
}

type Server struct {
	manager *TaskManager
}

func NewServer() *Server {
	return &Server{manager: NewTaskManager()}
}

func (s *Server) Start(addr string) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/eval/single", s.handleEvalSingle)
	mux.HandleFunc("/api/eval/compare", s.handleEvalCompare)
	mux.HandleFunc("/api/tasks/", s.handleTaskRoutes)
	log.Printf("SSE gateway listening on %s", addr)
	return http.ListenAndServe(addr, mux)
}

type singleRequest struct {
	SuitePath string `json:"suite_path"`
	SkillName string `json:"skill_name"`
}

type compareRequest struct {
	SuitePath  string `json:"suite_path"`
	LeftSkill  string `json:"left_skill"`
	RightSkill string `json:"right_skill"`
}

func (s *Server) handleEvalSingle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req singleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	task := s.manager.Create("single")
	s.manager.UpdateStatus(task.ID, TaskRunning, "", nil)

	go func(taskID string) {
		suite, err := evaluator.LoadSuiteFromJSON(req.SuitePath)
		if err != nil {
			s.manager.UpdateStatus(taskID, TaskFailed, err.Error(), nil)
			return
		}
		wd, _ := os.Getwd()
		runner := evaluator.NewRunner(evaluator.EvalConfig{
			MaxRounds:      8,
			UseDocker:      true,
			ProjectRootDir: wd,
			TaskID:         taskID,
			Publisher:      s.manager,
		})
		report, err := runner.RunSingle(context.Background(), req.SkillName, suite)
		if err != nil {
			s.manager.UpdateStatus(taskID, TaskFailed, err.Error(), report)
			return
		}
		s.manager.UpdateStatus(taskID, TaskSuccess, "", report)
	}(task.ID)

	writeJSON(w, map[string]any{"task_id": task.ID})
}

func (s *Server) handleEvalCompare(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req compareRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	task := s.manager.Create("compare")
	s.manager.UpdateStatus(task.ID, TaskRunning, "", nil)

	go func(taskID string) {
		suite, err := evaluator.LoadSuiteFromJSON(req.SuitePath)
		if err != nil {
			s.manager.UpdateStatus(taskID, TaskFailed, err.Error(), nil)
			return
		}
		wd, _ := os.Getwd()
		runner := evaluator.NewRunner(evaluator.EvalConfig{
			MaxRounds:      8,
			UseDocker:      true,
			ProjectRootDir: wd,
			TaskID:         taskID,
			Publisher:      s.manager,
		})
		report, err := runner.RunCompare(context.Background(), req.LeftSkill, req.RightSkill, suite)
		if err != nil {
			s.manager.UpdateStatus(taskID, TaskFailed, err.Error(), report)
			return
		}
		s.manager.UpdateStatus(taskID, TaskSuccess, "", report)
	}(task.ID)

	writeJSON(w, map[string]any{"task_id": task.ID})
}

func (s *Server) handleTaskRoutes(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/tasks/")
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		http.NotFound(w, r)
		return
	}
	taskID := parts[0]
	if len(parts) == 1 && r.Method == http.MethodGet {
		s.handleTaskGet(w, r, taskID)
		return
	}
	if len(parts) == 2 && parts[1] == "stream" && r.Method == http.MethodGet {
		s.handleTaskStream(w, r, taskID)
		return
	}
	http.NotFound(w, r)
}

func (s *Server) handleTaskGet(w http.ResponseWriter, r *http.Request, taskID string) {
	task, ok := s.manager.Get(taskID)
	if !ok {
		http.Error(w, "task not found", http.StatusNotFound)
		return
	}
	writeJSON(w, task)
}

func (s *Server) handleTaskStream(w http.ResponseWriter, r *http.Request, taskID string) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "stream unsupported", http.StatusInternalServerError)
		return
	}
	task, exists := s.manager.Get(taskID)
	if !exists {
		http.Error(w, "task not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	for _, evt := range task.Events {
		writeSSE(w, evt)
	}
	flusher.Flush()

	ch, cancel := s.manager.Subscribe(taskID)
	defer cancel()
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case evt := <-ch:
			writeSSE(w, evt)
			flusher.Flush()
		case <-ticker.C:
			_, _ = w.Write([]byte(": ping\n\n"))
			flusher.Flush()
		}
	}
}

func writeSSE(w http.ResponseWriter, evt evaluator.EvalEvent) {
	b, _ := json.Marshal(evt)
	_, _ = w.Write([]byte("data: "))
	_, _ = w.Write(b)
	_, _ = w.Write([]byte("\n\n"))
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
