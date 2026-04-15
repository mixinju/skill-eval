package evaluator

import "skill-eval/providers"

type TestCase struct {
	ID          string   `json:"id"`
	Input       string   `json:"input"`
	Expectation string   `json:"expectation"`
	Tags        []string `json:"tags,omitempty"`
}

type CaseGroup struct {
	GroupID string     `json:"group_id"`
	Cases   []TestCase `json:"cases"`
}

type EvalSuite struct {
	SuiteID string      `json:"suite_id"`
	Groups  []CaseGroup `json:"groups"`
}

type EvalConfig struct {
	MaxRounds      int            `json:"max_rounds"`
	UseDocker      bool           `json:"use_docker"`
	DockerImage    string         `json:"docker_image"`
	ProjectRootDir string         `json:"project_root_dir"`
	WorkDirBase    string         `json:"work_dir_base"`
	KeepWorkDir    bool           `json:"keep_work_dir"`
	TaskID         string         `json:"task_id,omitempty"`
	Publisher      EventPublisher `json:"-"`
}

type EvalEvent struct {
	TaskID string         `json:"task_id"`
	Type   string         `json:"type"`
	Time   string         `json:"time"`
	Data   map[string]any `json:"data,omitempty"`
}

type EventPublisher interface {
	Publish(event EvalEvent)
}

type DockerHealth struct {
	Enabled         bool   `json:"enabled"`
	Available       bool   `json:"available"`
	APIVersion      string `json:"api_version,omitempty"`
	ServerVersion   string `json:"server_version,omitempty"`
	OperatingSystem string `json:"operating_system,omitempty"`
	Error           string `json:"error,omitempty"`
	CheckedAt       string `json:"checked_at"`
}

type CaseResult struct {
	Case       TestCase            `json:"case"`
	Run        providers.RunResult `json:"run"`
	AIEval     string              `json:"ai_eval"`
	HumanCheck string              `json:"human_check"`
	Score      float64             `json:"score"`
	Runner     string              `json:"runner"`
	Container  string              `json:"container,omitempty"`
	Image      string              `json:"image,omitempty"`
	DebugLog   string              `json:"debug_log,omitempty"`
}

type GroupResult struct {
	GroupID string       `json:"group_id"`
	Results []CaseResult `json:"results"`
}

type EvalReport struct {
	Mode             string         `json:"mode"`
	SuiteID          string         `json:"suite_id"`
	SkillName        string         `json:"skill_name"`
	GroupResults     []GroupResult  `json:"group_results"`
	SuccessRate      float64        `json:"success_rate"`
	AverageScore     float64        `json:"average_score"`
	TotalCases       int            `json:"total_cases"`
	SuccessCases     int            `json:"success_cases"`
	FailedCases      int            `json:"failed_cases"`
	TotalTokens      int64          `json:"total_tokens"`
	TotalRounds      int            `json:"total_rounds"`
	GeneratedAt      string         `json:"generated_at"`
	IsolationMode    string         `json:"isolation_mode"`
	WorkspaceDir     string         `json:"workspace_dir,omitempty"`
	WorkDirKept      bool           `json:"work_dir_kept"`
	DockerHealth     DockerHealth   `json:"docker_health"`
	FallbackCount    int            `json:"fallback_count"`
	FallbackByReason map[string]int `json:"fallback_by_reason,omitempty"`
}

type CompareReport struct {
	Mode               string         `json:"mode"`
	SuiteID            string         `json:"suite_id"`
	LeftSkill          string         `json:"left_skill"`
	RightSkill         string         `json:"right_skill"`
	Left               EvalReport     `json:"left"`
	Right              EvalReport     `json:"right"`
	Winner             string         `json:"winner"`
	GeneratedAt        string         `json:"generated_at"`
	IsolationMode      string         `json:"isolation_mode"`
	Notes              []string       `json:"notes,omitempty"`
	WorkspaceDirs      []string       `json:"workspace_dirs,omitempty"`
	DockerHealth       DockerHealth   `json:"docker_health"`
	TotalFallbackCount int            `json:"total_fallback_count"`
	FallbackByReason   map[string]int `json:"fallback_by_reason,omitempty"`
}
