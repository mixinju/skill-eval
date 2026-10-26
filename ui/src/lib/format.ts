import type { Span, SpanKind } from '../types'

export function cx(...parts: Array<string | false | null | undefined>): string {
  return parts.filter(Boolean).join(' ')
}

export function formatTokens(value: number): string {
  if (!Number.isFinite(value)) return '—'
  const abs = Math.abs(value)
  if (abs >= 1000) {
    const compact = value / 1000
    const digits = Math.abs(compact) >= 10 ? 0 : 1
    return `${compact.toFixed(digits)}k`
  }
  return String(Math.round(value))
}

export function formatMs(ms: number): string {
  if (!Number.isFinite(ms)) return '—'
  if (ms < 1) return '<1ms'
  if (ms < 1000) return `${Math.round(ms)}ms`
  if (ms < 60_000) {
    const seconds = ms / 1000
    return `${seconds >= 10 ? seconds.toFixed(0) : seconds.toFixed(1)}s`
  }
  const minutes = Math.floor(ms / 60_000)
  const seconds = Math.round((ms % 60_000) / 1000)
  return `${minutes}m ${seconds}s`
}

export function formatSignedMs(ms: number): string {
  if (!Number.isFinite(ms)) return '—'
  if (ms === 0) return '0'
  const sign = ms > 0 ? '+' : '−'
  return `${sign}${formatMs(Math.abs(ms))}`
}

export function formatWhen(iso?: string): string {
  if (!iso) return '—'
  const date = new Date(iso)
  if (Number.isNaN(date.getTime())) return '—'
  return date.toLocaleString('zh-CN', {
    month: 'numeric',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  })
}

export function formatClock(iso?: string): string {
  if (!iso) return '—'
  const date = new Date(iso)
  if (Number.isNaN(date.getTime())) return '—'
  return date.toLocaleTimeString('zh-CN', {
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
  })
}

export function pretty(raw?: string): string {
  if (!raw) return ''
  try {
    return JSON.stringify(JSON.parse(raw), null, 2)
  } catch {
    return raw
  }
}

export function kindLabel(kind: SpanKind | 'run'): string {
  switch (kind) {
    case 'llm_call':
      return 'LLM'
    case 'tool_call':
      return '工具'
    case 'llm_compress':
      return '压缩'
    case 'run':
      return '运行'
    default:
      return '未知'
  }
}

/** 优先用起止时间。旧 Trace 的 duration 有时是纳秒。 */
export function durationMs(span: Span): number | null {
  if (span.startTime && span.endTime) {
    const start = Date.parse(span.startTime)
    const end = Date.parse(span.endTime)
    if (Number.isFinite(start) && Number.isFinite(end) && end >= start) return end - start
  }
  if (typeof span.duration !== 'number' || !Number.isFinite(span.duration) || span.duration < 0) return null
  if (span.duration >= 1_000_000) return span.duration / 1e6
  return span.duration
}

export function spanDurationLabel(span: Span): string {
  const ms = durationMs(span)
  if (ms === null) return '—'
  return formatMs(ms)
}

export function traceDurationMs(start?: string, end?: string): number | null {
  if (!start || !end) return null
  const a = Date.parse(start)
  const b = Date.parse(end)
  if (!Number.isFinite(a) || !Number.isFinite(b) || b < a) return null
  return b - a
}

export function clampScore(score: number | null): number | null {
  if (score === null || !Number.isFinite(score)) return null
  return Math.min(1, Math.max(0, score))
}

export function formatScore(score: number | null): string {
  if (score === null || !Number.isFinite(score)) return '—'
  return score.toFixed(2)
}

export function shortId(id: string): string {
  if (id.length <= 14) return id
  return `${id.slice(0, 8)}…${id.slice(-4)}`
}
