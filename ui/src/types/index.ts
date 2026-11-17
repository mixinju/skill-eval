export type RunnerId = 'self' | 'codex' | 'claude-code'

export type SpanKind = 'llm_call' | 'tool_call' | 'llm_compress'

export interface Span {
  spanId: string
  parentId?: string
  kind: SpanKind
  name: string
  iteration: number
  startTime: string
  endTime?: string
  duration?: number
  inputMessages?: number
  totalTokens?: number
  finishReason?: string
  llmInput?: string
  llmOutput?: string
  toolInput?: string
  toolOutput?: string
  isTarget?: boolean
  error?: string
}

export interface RunnerMeta {
  type: string
  displayName?: string
  binary?: string
  version?: string
  model?: string
  isolated?: boolean
  skillDir?: string
  command?: string
  logPath?: string
}

export interface Step {
  index: number
  kind: string
  name: string
  iteration: number
  ok: boolean
  summary?: string
  error?: string
}

export interface Trace {
  id: string
  agentName: string
  model: string
  userPrompt: string
  targetSkill?: string
  startTime: string
  endTime?: string
  totalTokens: number
  iterations: number
  success: boolean
  spans: Span[]
  artifactsAndResult?: string
  schemaVersion?: number
  runner?: string
  runnerMeta?: RunnerMeta
  status?: string
  failureReason?: string
  error?: string
  workspace?: string
  skillPath?: string
  externalId?: string
  notes?: string[]
  steps?: Step[]
}

export interface ScoreItem {
  name: string
  desc: string
}

export interface Verdict {
  info: ScoreItem
  pass: boolean
  score: number
  reason: string
  source?: string
  error?: string
}

export interface Report {
  traceId: string
  agentName: string
  model: string
  userPrompt: string
  targetSkill: string
  totalTokens: number
  iterations: number
  duration: number
  scores: Verdict[]
  pass: boolean
  runner?: string
  runnerMeta?: RunnerMeta
  status?: string
  failureReason?: string
  workspace?: string
  skillPath?: string
}

export interface CompareReport {
  traceA: Trace
  traceB: Trace
  reportA: Report
  reportB: Report
  scores: Verdict[]
  conclusion: string
}

export interface EvalRecord {
  id: string
  type: 'single' | 'compare'
  createdAt: string
  status?: string
  runner?: string
  error?: string
  logPath?: string
  report?: Report
  trace?: Trace
  compareReport?: CompareReport
}

export interface RunnerAvailability {
  type: RunnerId | string
  name: string
  available: boolean
  detail: string
  binary?: string
  version?: string
}

export interface ArtifactInfo {
  path: string
  name: string
  exists: boolean
  size?: number
}

export interface ArtifactsResponse {
  result?: string
  artifacts: ArtifactInfo[]
  workspace?: string
}
