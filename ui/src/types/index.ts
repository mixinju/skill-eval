export type RunnerId = 'kernel' | 'codex' | 'claude-code'

export type SpanKind = 'llm_call' | 'tool_call' | 'llm_compress' | 'unknown'

export type ScoreAvailability = 'derived' | 'persisted' | 'fixture' | 'unavailable'

export type ScoreSource = 'derived' | 'persisted' | 'fixture'

export type Outcome = 'pass' | 'fail' | 'unknown'

export type RecordSource = 'trace-file' | 'fixture'

export interface Span {
  spanId: string
  parentId?: string
  kind: SpanKind
  name: string
  iteration: number
  startTime: string
  endTime?: string
  /** 原始 duration。旧文件里有的是纳秒，展示时改用起止时间。 */
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

export interface Trace {
  id: string
  agentName: string
  model: string
  userPrompt: string
  targetSkill: string
  startTime: string
  endTime?: string
  totalTokens: number
  iterations: number
  success: boolean
  runner?: string
  artifactsAndResult?: string
  spans: Span[]
}

export interface ScoreView {
  name: string
  desc: string
  availability: ScoreAvailability
  pass: boolean | null
  /** 与 eval.Verdict.score 一致，范围 0–1。没有分数时为 null。 */
  score: number | null
  reason: string
}

export interface DroppedCall {
  iteration: number
  name: string
  arguments: string
}

export interface EvalRecord {
  id: string
  createdAt: string
  source: RecordSource
  file?: string
  runner: RunnerId
  runnerRaw?: string
  runnerInferred: boolean
  trace: Trace
  scores: ScoreView[]
  scoreSource: ScoreSource
  outcome: Outcome
  summary: string
  artifacts: string[]
  finalResult?: string
  droppedCalls: DroppedCall[]
  gaps: string[]
}

export interface RawSpan {
  spanId?: string
  parentId?: string
  kind?: string
  name?: string
  iteration?: number
  startTime?: string
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

export interface RawTrace {
  id?: string
  agentName?: string
  model?: string
  userPrompt?: string
  targetSkill?: string
  startTime?: string
  endTime?: string
  totalTokens?: number
  iterations?: number
  success?: boolean
  runner?: string
  artifactsAndResult?: string
  spans?: RawSpan[]
}

export interface RawVerdict {
  info?: { name?: string; desc?: string }
  pass?: boolean
  score?: number
  reason?: string
}

/** 与 eval.Report 的 JSON 对齐。后端目前不会把报告写盘。 */
export interface RawReport {
  traceId?: string
  pass?: boolean
  scores?: RawVerdict[]
}

export interface ToolCallView {
  name: string
  arguments: string
}

export interface AssistantView {
  content?: string
  reasoning?: string
  toolCalls: ToolCallView[]
  raw: string
}

export interface ChatMessage {
  role: string
  content: string
}
