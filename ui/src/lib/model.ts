import type {
  AssistantView,
  ChatMessage,
  DroppedCall,
  EvalRecord,
  Outcome,
  RawReport,
  RawSpan,
  RawTrace,
  ScoreSource,
  ScoreView,
  Span,
  SpanKind,
  Trace,
} from '../types'
import { isRunnerId } from './runners'

const SPAN_KINDS = new Set<SpanKind>(['llm_call', 'tool_call', 'llm_compress', 'unknown'])

function asKind(value: unknown): SpanKind {
  return typeof value === 'string' && SPAN_KINDS.has(value as SpanKind) ? (value as SpanKind) : 'unknown'
}

function asNumber(value: unknown): number | undefined {
  return typeof value === 'number' && Number.isFinite(value) ? value : undefined
}

function normalizeSpan(raw: RawSpan, index: number): Span {
  return {
    spanId: raw.spanId || `span-${index}`,
    parentId: raw.parentId || undefined,
    kind: asKind(raw.kind),
    name: raw.name || '未命名',
    iteration: asNumber(raw.iteration) ?? 0,
    startTime: raw.startTime || '',
    endTime: raw.endTime || undefined,
    duration: asNumber(raw.duration),
    inputMessages: asNumber(raw.inputMessages),
    totalTokens: asNumber(raw.totalTokens),
    finishReason: raw.finishReason || undefined,
    llmInput: raw.llmInput || undefined,
    llmOutput: raw.llmOutput || undefined,
    toolInput: raw.toolInput || undefined,
    toolOutput: raw.toolOutput || undefined,
    isTarget: raw.isTarget === true,
    error: raw.error || undefined,
  }
}

export function normalizeTrace(raw: RawTrace, file?: string): Trace {
  const spans = Array.isArray(raw.spans) ? raw.spans.map(normalizeSpan) : []
  const iterations = asNumber(raw.iterations) ?? spans.reduce((max, span) => Math.max(max, span.iteration), 0)
  return {
    id: raw.id || file || 'trace',
    agentName: raw.agentName || '未命名 Agent',
    model: raw.model || '未知模型',
    userPrompt: raw.userPrompt || '',
    targetSkill: raw.targetSkill || '',
    startTime: raw.startTime || '',
    endTime: raw.endTime || undefined,
    totalTokens: asNumber(raw.totalTokens) ?? 0,
    iterations,
    success: raw.success === true,
    runner: typeof raw.runner === 'string' ? raw.runner : undefined,
    artifactsAndResult: raw.artifactsAndResult || undefined,
    spans,
  }
}

export function parseAssistant(raw?: string): AssistantView | null {
  if (!raw) return null
  try {
    const value = JSON.parse(raw) as unknown
    if (!value || typeof value !== 'object' || Array.isArray(value)) {
      return { raw, toolCalls: [] }
    }
    const record = value as Record<string, unknown>
    const calls = Array.isArray(record.tool_calls) ? record.tool_calls : []
    const toolCalls = calls.flatMap((call) => {
      if (!call || typeof call !== 'object') return []
      const fn = (call as { function?: { name?: unknown; arguments?: unknown } }).function
      const name = typeof fn?.name === 'string' ? fn.name : ''
      if (!name) return []
      const args = fn?.arguments
      return [{ name, arguments: typeof args === 'string' ? args : JSON.stringify(args ?? {}) }]
    })
    return {
      content: typeof record.content === 'string' ? record.content : undefined,
      reasoning: typeof record.reasoning_content === 'string' ? record.reasoning_content : undefined,
      toolCalls,
      raw,
    }
  } catch {
    return { raw, toolCalls: [] }
  }
}

export function parseMessages(raw?: string): ChatMessage[] | null {
  if (!raw) return null
  try {
    const value = JSON.parse(raw) as unknown
    if (!Array.isArray(value)) return null
    return value.map((item) => {
      if (!item || typeof item !== 'object') return { role: 'unknown', content: String(item) }
      const record = item as Record<string, unknown>
      return {
        role: typeof record.role === 'string' ? record.role : 'unknown',
        content: typeof record.content === 'string' ? record.content : '',
      }
    })
  } catch {
    return null
  }
}

function parseFinishPayload(raw?: string): { result?: string; artifacts: string[] } | null {
  if (!raw) return null
  try {
    const value = JSON.parse(raw) as unknown
    if (!value || typeof value !== 'object' || Array.isArray(value)) return null
    const record = value as Record<string, unknown>
    const artifacts = Array.isArray(record.artifacts)
      ? record.artifacts.filter((item): item is string => typeof item === 'string')
      : []
    const result = typeof record.result === 'string' ? record.result : undefined
    if (!result && artifacts.length === 0) return null
    return { result, artifacts }
  } catch {
    return null
  }
}

export function readFinish(trace: Trace): { result?: string; artifacts: string[] } {
  const fromTrace = parseFinishPayload(trace.artifactsAndResult)
  if (fromTrace) return fromTrace
  const finish = [...trace.spans].reverse().find((span) => span.kind === 'tool_call' && span.name === 'finish')
  if (!finish) return { artifacts: [] }
  return parseFinishPayload(finish.toolOutput) ?? parseFinishPayload(finish.toolInput) ?? {
    result: finish.toolOutput || undefined,
    artifacts: [],
  }
}

export function droppedCalls(trace: Trace): DroppedCall[] {
  const groups = new Map<number, Span[]>()
  for (const span of trace.spans) {
    const list = groups.get(span.iteration) ?? []
    list.push(span)
    groups.set(span.iteration, list)
  }
  const dropped: DroppedCall[] = []
  for (const [iteration, spans] of groups) {
    const remaining = new Map<string, number>()
    for (const span of spans) {
      if (span.kind !== 'tool_call') continue
      remaining.set(span.name, (remaining.get(span.name) ?? 0) + 1)
    }
    for (const span of spans) {
      if (span.kind !== 'llm_call') continue
      const assistant = parseAssistant(span.llmOutput)
      if (!assistant) continue
      for (const call of assistant.toolCalls) {
        const left = remaining.get(call.name) ?? 0
        if (left > 0) {
          remaining.set(call.name, left - 1)
          continue
        }
        dropped.push({ iteration, name: call.name, arguments: call.arguments })
      }
    }
  }
  return dropped
}

function skillHit(trace: Trace): ScoreView {
  const name = 'Skill是否命中'
  const desc = '根据执行链路，评测是否命中目标Skill的展示'
  if (!trace.targetSkill) {
    return {
      name,
      desc,
      availability: 'derived',
      pass: true,
      score: 1,
      reason: '未设置目标 skill，跳过检查',
    }
  }
  for (const span of trace.spans) {
    if (span.kind !== 'tool_call' || span.name !== 'use_skill' || !span.toolInput) continue
    try {
      const params = JSON.parse(span.toolInput) as { name?: unknown }
      if (params.name === trace.targetSkill) {
        return {
          name,
          desc,
          availability: 'derived',
          pass: true,
          score: 1,
          reason: `命中目标 skill: ${trace.targetSkill}`,
        }
      }
    } catch {
      continue
    }
  }
  return {
    name,
    desc,
    availability: 'derived',
    pass: false,
    score: 0,
    reason: `未命中目标 skill: ${trace.targetSkill}`,
  }
}

function successScore(trace: Trace): ScoreView {
  return {
    name: '是否执行成功',
    desc: '整个智能体执行流程是否完成任务',
    availability: 'derived',
    pass: trace.success,
    score: trace.success ? 1 : 0,
    reason: trace.success ? 'Trace.success 为 true' : 'Trace.success 为 false',
  }
}

function artifactScore(artifacts: string[]): ScoreView {
  if (artifacts.length === 0) {
    return {
      name: '产物评分',
      desc: '针对输出',
      availability: 'derived',
      pass: false,
      score: 0,
      reason: '未产出任何文件',
    }
  }
  return {
    name: '产物评分',
    desc: '针对输出',
    availability: 'unavailable',
    pass: null,
    score: null,
    reason: `检测到 ${artifacts.length} 个产物。质量分要由 ArtifactScorer 写入报告，当前没有这份报告。`,
  }
}

function processScore(): ScoreView {
  return {
    name: '执行过程',
    desc: '评测执行过程',
    availability: 'unavailable',
    pass: null,
    score: null,
    reason: '执行过程分由 ExecProcessScorer 调用模型生成。报告没有落盘，这里不编造分数。',
  }
}

function summarize(trace: Trace, dropped: DroppedCall[]): string {
  const tools = trace.spans.filter((span) => span.kind === 'tool_call').map((span) => span.name)
  const head = tools.length ? `实际执行 ${tools.join(' → ')}` : '没有工具 Span'
  if (!dropped.length) return `${head}。`
  const names = [...new Set(dropped.map((call) => call.name))].join('、')
  return `${head}。模型还请求了 ${names}，但同一轮没有对应的工具 Span。`
}

function outcomeFrom(scores: ScoreView[], override?: boolean): Outcome {
  if (typeof override === 'boolean') return override ? 'pass' : 'fail'
  const decided = scores.filter((score) => score.pass !== null)
  if (!decided.length) return 'unknown'
  return decided.every((score) => score.pass) ? 'pass' : 'fail'
}

function collectGaps(record: Omit<EvalRecord, 'gaps'>): string[] {
  const gaps: string[] = []
  if (record.source === 'fixture') {
    gaps.push('这是界面样例，不是 Codex 或 Claude Code 的真实运行。仓库里没有这两个执行器。')
  }
  if (record.runnerRaw && record.runnerRaw !== record.runner) {
    gaps.push(`Trace 上的 runner「${record.runnerRaw}」不在 kernel / codex / claude-code 里，暂时归入自研内核。`)
  } else if (record.runnerInferred) {
    gaps.push('这条 Trace 没有 runner 字段。目前只有自研内核会跑评测，所以归在自研内核。之后新跑的 Trace 会写上 runner: "kernel"。')
  }
  if (record.scoreSource === 'derived') {
    gaps.push('没有落盘的评测报告。通过与否只来自 Trace 上能确定的检查，不含模型打分。')
  }
  if (record.scoreSource === 'fixture') {
    gaps.push('标成「样例」的分数是前端写的，用来预览评分卡。标成「Trace」的分数按现有评分规则从这条样例链路算出。')
  }
  return gaps
}

function finalize(draft: Omit<EvalRecord, 'gaps' | 'outcome'>, outcome?: Outcome): EvalRecord {
  const next: EvalRecord = {
    ...draft,
    outcome: outcome ?? outcomeFrom(draft.scores),
    gaps: [],
  }
  next.gaps = collectGaps(next)
  return next
}

export function recordFromTrace(options: {
  trace: RawTrace
  source: EvalRecord['source']
  file?: string
}): EvalRecord {
  const trace = normalizeTrace(options.trace, options.file)
  const dropped = droppedCalls(trace)
  const finish = readFinish(trace)
  const runnerRaw = trace.runner
  const known = runnerRaw ? isRunnerId(runnerRaw) : false
  const scores = [skillHit(trace), successScore(trace), artifactScore(finish.artifacts), processScore()]
  return finalize({
    id: trace.id,
    createdAt: trace.startTime,
    source: options.source,
    file: options.file,
    runner: known && runnerRaw && isRunnerId(runnerRaw) ? runnerRaw : 'kernel',
    runnerRaw,
    runnerInferred: !known,
    trace,
    scores,
    scoreSource: 'derived',
    summary: summarize(trace, dropped),
    artifacts: finish.artifacts,
    finalResult: finish.result,
    droppedCalls: dropped,
  })
}

export function withFixtureScores(record: EvalRecord, extras: ScoreView[]): EvalRecord {
  const replaced = new Set(extras.map((score) => score.name))
  const scores = [...record.scores.filter((score) => !replaced.has(score.name)), ...extras]
  return finalize({ ...record, scores, scoreSource: 'fixture' satisfies ScoreSource })
}

export function applyReport(record: EvalRecord, report: RawReport): EvalRecord {
  const scores: ScoreView[] = (report.scores ?? []).map((item) => ({
    name: item.info?.name || '未命名',
    desc: item.info?.desc || '',
    availability: 'persisted',
    pass: typeof item.pass === 'boolean' ? item.pass : null,
    score: typeof item.score === 'number' && Number.isFinite(item.score) ? item.score : null,
    reason: item.reason || '',
  }))
  if (!scores.length) return record
  const outcome = typeof report.pass === 'boolean' ? outcomeFrom(scores, report.pass) : outcomeFrom(scores)
  return finalize({ ...record, scores, scoreSource: 'persisted' }, outcome)
}

export function outcomeCaption(record: EvalRecord): string {
  if (record.scoreSource === 'persisted') {
    return record.outcome === 'pass' ? '评测报告为通过。' : '评测报告为未通过。'
  }
  if (record.scoreSource === 'fixture') {
    return record.outcome === 'pass'
      ? '样例。链路检查和样例分数都通过。'
      : '样例。有检查未通过。'
  }
  const failed = record.scores.find((score) => score.pass === false)
  if (failed) return failed.reason
  const pending = record.scores.some((score) => score.availability === 'unavailable')
  if (record.outcome === 'pass' && pending) return '确定性检查通过。模型分数还没有报告。'
  if (record.outcome === 'pass') return '确定性检查通过。'
  return '还没有足够的评分来判断通过或失败。'
}

export function isNestedSpan(span: Span): boolean {
  return Boolean(span.parentId && span.parentId !== span.spanId)
}

export interface IterationGroup {
  iteration: number
  spans: { span: Span; index: number }[]
}

export function groupIterations(trace: Trace): IterationGroup[] {
  const map = new Map<number, IterationGroup>()
  trace.spans.forEach((span, index) => {
    const group = map.get(span.iteration) ?? { iteration: span.iteration, spans: [] }
    group.spans.push({ span, index })
    map.set(span.iteration, group)
  })
  return [...map.values()].sort((a, b) => a.iteration - b.iteration)
}

export interface TraceStats {
  durationMs: number | null
  llm: number
  tool: number
  compress: number
  errors: number
  iterations: number
  tokens: number
}

export function traceStats(trace: Trace): TraceStats {
  let llm = 0
  let tool = 0
  let compress = 0
  let errors = 0
  for (const span of trace.spans) {
    if (span.kind === 'llm_call') llm += 1
    else if (span.kind === 'tool_call') tool += 1
    else if (span.kind === 'llm_compress') compress += 1
    if (span.error) errors += 1
  }
  const durationMs = (() => {
    if (!trace.startTime || !trace.endTime) return null
    const start = Date.parse(trace.startTime)
    const end = Date.parse(trace.endTime)
    if (!Number.isFinite(start) || !Number.isFinite(end) || end < start) return null
    return end - start
  })()
  return { durationMs, llm, tool, compress, errors, iterations: trace.iterations, tokens: trace.totalTokens }
}

export function timelineScale(trace: Trace): { start: number; end: number } {
  let start = Date.parse(trace.startTime)
  let end = trace.endTime ? Date.parse(trace.endTime) : start
  if (!Number.isFinite(start)) start = 0
  if (!Number.isFinite(end) || end < start) end = start
  for (const span of trace.spans) {
    const spanStart = Date.parse(span.startTime)
    const spanEnd = span.endTime ? Date.parse(span.endTime) : spanStart
    if (Number.isFinite(spanStart)) start = Math.min(start, spanStart)
    if (Number.isFinite(spanEnd)) end = Math.max(end, spanEnd)
  }
  if (end <= start) end = start + 1
  return { start, end }
}

export function barPlacement(span: Span, scale: { start: number; end: number }): { left: string; width: string } {
  const spanStart = Date.parse(span.startTime)
  const spanEnd = span.endTime ? Date.parse(span.endTime) : spanStart
  const total = scale.end - scale.start
  const left = Number.isFinite(spanStart) ? ((spanStart - scale.start) / total) * 100 : 0
  const rawWidth = Number.isFinite(spanStart) && Number.isFinite(spanEnd) ? ((spanEnd - spanStart) / total) * 100 : 0
  const clampedLeft = Math.min(Math.max(left, 0), 99)
  const width = Math.min(Math.max(rawWidth, 0.8), 100 - clampedLeft)
  return { left: `${clampedLeft}%`, width: `${width}%` }
}

export function isRawReport(value: unknown): value is RawReport {
  return Boolean(value) && typeof value === 'object' && typeof (value as RawReport).traceId === 'string'
}
