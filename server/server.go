package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"skill-eval/agent"
	"skill-eval/eval"
	"skill-eval/runner"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// Server 提供前端消费的评测 API。
type Server struct {
	reg     *runner.Registry
	store   *Store
	dataDir string
}

func New(dataDir string, reg *runner.Registry) (*Server, error) {
	if reg == nil {
		reg = runner.NewDefaultRegistry()
	}
	store, err := NewStore(dataDir)
	if err != nil {
		return nil, err
	}
	if dataDir == "" {
		dataDir = "logs"
	}
	return &Server{reg: reg, store: store, dataDir: dataDir}, nil
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", s.health)
	mux.HandleFunc("GET /api/runners", s.runners)
	mux.HandleFunc("POST /api/evals", s.createEval)
	mux.HandleFunc("GET /api/evals", s.listEvals)
	mux.HandleFunc("POST /api/evals/compare", s.compare)
	mux.HandleFunc("GET /api/evals/{id}", s.getEval)
	mux.HandleFunc("GET /api/evals/{id}/logs", s.getLogs)
	mux.HandleFunc("GET /api/evals/{id}/trace", s.getTrace)
	mux.HandleFunc("GET /api/evals/{id}/report", s.getReport)
	return withCORS(mux)
}

func (s *Server) Listen(addr string) error {
	if addr == "" {
		addr = ":8080"
	}
	logrus.Infof("评测 API 监听 %s", addr)
	return http.ListenAndServe(addr, s.Handler())
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) runners(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"runners": s.reg.Probe(r.Context())})
}

type evalRequest struct {
	Runner        string `json:"runner"`
	Name          string `json:"name"`
	Description   string `json:"description"`
	SystemPrompt  string `json:"systemPrompt"`
	UserPrompt    string `json:"userPrompt"`
	Model         string `json:"model"`
	TargetSkill   string `json:"targetSkill"`
	SkillDir      string `json:"skillDir"`
	MaxIterations int    `json:"maxIterations"`
	TimeoutSec    int    `json:"timeoutSec"`
	Wait          bool   `json:"wait"`
}

func (s *Server) createEval(w http.ResponseWriter, r *http.Request) {
	var req evalRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "请求体不是合法 JSON")
		return
	}
	if r.URL.Query().Get("wait") == "true" {
		req.Wait = true
	}
	task, kind, err := s.taskFromRequest(req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	rec := &Record{
		ID:        task.ID,
		Type:      "single",
		CreatedAt: time.Now().UTC(),
		Status:    "running",
		Runner:    string(kind),
	}
	if err := s.store.Save(rec); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if req.Wait {
		done, err := s.execute(r.Context(), kind, task)
		if done == nil {
			writeError(w, http.StatusBadGateway, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, done)
		return
	}
	go func() {
		if _, err := s.execute(context.Background(), kind, task); err != nil {
			logrus.Warnf("评测 %s 执行结束: %v", task.ID, err)
		}
	}()
	writeJSON(w, http.StatusAccepted, rec)
}

func (s *Server) taskFromRequest(req evalRequest) (runner.Task, runner.Type, error) {
	if strings.TrimSpace(req.UserPrompt) == "" {
		return runner.Task{}, "", fmt.Errorf("userPrompt 不能为空")
	}
	kind, err := runner.Normalize(req.Runner)
	if err != nil {
		return runner.Task{}, "", err
	}
	task := runner.Task{
		ID:            uuid.NewString(),
		Name:          req.Name,
		Description:   req.Description,
		SystemPrompt:  req.SystemPrompt,
		UserPrompt:    req.UserPrompt,
		Model:         req.Model,
		TargetSkill:   req.TargetSkill,
		SkillDir:      req.SkillDir,
		MaxIterations: req.MaxIterations,
		DataDir:       s.dataDir,
	}
	if req.TimeoutSec > 0 {
		task.Timeout = time.Duration(req.TimeoutSec) * time.Second
	}
	if kind == runner.TypeSelf && strings.TrimSpace(task.Model) == "" {
		return runner.Task{}, "", fmt.Errorf("self runner 需要 model")
	}
	return task, kind, nil
}

// Run 执行一次评测并保存记录。供 CLI 和 HTTP 共用。
func (s *Server) Run(ctx context.Context, kind runner.Type, task runner.Task) (*Record, error) {
	if task.ID == "" {
		task.ID = uuid.NewString()
	}
	if task.DataDir == "" {
		task.DataDir = s.dataDir
	}
	rec := &Record{
		ID:        task.ID,
		Type:      "single",
		CreatedAt: time.Now().UTC(),
		Status:    "running",
		Runner:    string(kind),
	}
	if err := s.store.Save(rec); err != nil {
		return nil, err
	}
	return s.execute(ctx, kind, task)
}

// Compare 对比两份已经完成的评测。
func (s *Server) Compare(idA, idB string) (*Record, error) {
	a, err := s.store.Get(idA)
	if err != nil {
		return nil, fmt.Errorf("评测 A 不存在")
	}
	b, err := s.store.Get(idB)
	if err != nil {
		return nil, fmt.Errorf("评测 B 不存在")
	}
	if a.Status == "running" || b.Status == "running" || a.Report == nil || b.Report == nil || a.Trace == nil || b.Trace == nil {
		return nil, fmt.Errorf("两边评测都完成后才能对比")
	}
	report, err := eval.Compare(a.Trace, b.Trace, a.Report, b.Report)
	if err != nil {
		return nil, err
	}
	rec := &Record{
		ID:            uuid.NewString(),
		Type:          "compare",
		CreatedAt:     time.Now().UTC(),
		Status:        "succeeded",
		CompareReport: report,
	}
	if err := s.store.Save(rec); err != nil {
		return nil, err
	}
	return rec, nil
}

func (s *Server) execute(ctx context.Context, kind runner.Type, task runner.Task) (*Record, error) {
	rn, ok := s.reg.Get(kind)
	if !ok {
		rec := &Record{ID: task.ID, Type: "single", CreatedAt: time.Now().UTC(), Status: "failed", Runner: string(kind), Error: "runner 未注册"}
		_ = s.store.Save(rec)
		return rec, fmt.Errorf("runner 未注册")
	}
	trace, runErr := rn.Run(ctx, task)
	return s.finish(task.ID, string(kind), trace, runErr)
}

func (s *Server) finish(id, kind string, trace *agent.Trace, runErr error) (*Record, error) {
	existing, _ := s.store.Get(id)
	created := time.Now().UTC()
	if existing != nil && !existing.CreatedAt.IsZero() {
		created = existing.CreatedAt
	}
	if trace == nil {
		reason := "runner 没有返回 trace"
		if runErr != nil {
			reason = runErr.Error()
		}
		trace = &agent.Trace{
			ID:            id,
			Success:       false,
			Status:        "failed",
			FailureReason: reason,
			Error:         reason,
			Runner:        kind,
			SchemaVersion: 1,
			Spans:         []*agent.Span{},
			StartTime:     created,
			EndTime:       time.Now().UTC(),
		}
	}
	if trace.Spans == nil {
		trace.Spans = []*agent.Span{}
	}
	report := eval.Exec(trace, eval.BuildDefaultScorer())
	status := trace.Status
	if status == "" || status == "running" {
		if trace.Success {
			status = "succeeded"
		} else {
			status = "failed"
		}
	}
	rec := &Record{
		ID:        id,
		Type:      "single",
		CreatedAt: created,
		Status:    status,
		Runner:    kind,
		Report:    report,
		Trace:     trace,
	}
	if trace.RunnerMeta != nil {
		rec.LogPath = trace.RunnerMeta.LogPath
	}
	if runErr != nil {
		rec.Error = runErr.Error()
	}
	if err := s.store.Save(rec); err != nil {
		return rec, err
	}
	return rec, runErr
}

func (s *Server) listEvals(w http.ResponseWriter, r *http.Request) {
	recs, err := s.store.List()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	out := make([]Record, 0, len(recs))
	for _, rec := range recs {
		out = append(out, summary(*rec))
	}
	writeJSON(w, http.StatusOK, map[string]any{"records": out})
}

func (s *Server) getEval(w http.ResponseWriter, r *http.Request) {
	rec, err := s.store.Get(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, rec)
}

func (s *Server) getTrace(w http.ResponseWriter, r *http.Request) {
	rec, err := s.store.Get(r.PathValue("id"))
	if err != nil || rec.Trace == nil {
		writeError(w, http.StatusNotFound, "trace 不存在")
		return
	}
	writeJSON(w, http.StatusOK, rec.Trace)
}

func (s *Server) getReport(w http.ResponseWriter, r *http.Request) {
	rec, err := s.store.Get(r.PathValue("id"))
	if err != nil || rec.Report == nil {
		writeError(w, http.StatusNotFound, "report 不存在")
		return
	}
	writeJSON(w, http.StatusOK, rec.Report)
}

func (s *Server) getLogs(w http.ResponseWriter, r *http.Request) {
	rec, err := s.store.Get(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	path := rec.LogPath
	if path == "" && rec.Trace != nil && rec.Trace.RunnerMeta != nil {
		path = rec.Trace.RunnerMeta.LogPath
	}
	if path == "" {
		writeError(w, http.StatusNotFound, "这次评测没有日志文件")
		return
	}
	data, err := os.ReadFile(path)
	if err != nil {
		writeError(w, http.StatusNotFound, "日志文件不存在: "+path)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write(data)
}

type compareRequest struct {
	EvalIDA string `json:"evalIdA"`
	EvalIDB string `json:"evalIdB"`
}

func (s *Server) compare(w http.ResponseWriter, r *http.Request) {
	var req compareRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "请求体不是合法 JSON")
		return
	}
	rec, err := s.Compare(req.EvalIDA, req.EvalIDB)
	if err != nil {
		status := http.StatusBadRequest
		if strings.Contains(err.Error(), "不存在") {
			status = http.StatusNotFound
		}
		if strings.Contains(err.Error(), "完成后") {
			status = http.StatusConflict
		}
		writeError(w, status, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, rec)
}

func summary(rec Record) Record {
	if rec.Trace != nil {
		cp := *rec.Trace
		cp.Spans = []*agent.Span{}
		cp.Steps = []agent.Step{}
		rec.Trace = &cp
	}
	if rec.CompareReport != nil {
		cp := *rec.CompareReport
		if cp.TraceA != nil {
			a := *cp.TraceA
			a.Spans = []*agent.Span{}
			a.Steps = []agent.Step{}
			cp.TraceA = &a
		}
		if cp.TraceB != nil {
			b := *cp.TraceB
			b.Spans = []*agent.Span{}
			b.Steps = []agent.Step{}
			cp.TraceB = &b
		}
		rec.CompareReport = &cp
	}
	return rec
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(body)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
