import type { EvalRecord, RawReport, RawTrace } from '../types'
import { applyReport, isRawReport, recordFromTrace } from './model'

interface FileEntry {
  file?: string
  data?: unknown
  error?: string
}

interface Payload {
  traces?: FileEntry[]
  reports?: FileEntry[]
}

export interface LoadResult {
  records: EvalRecord[]
  origin: 'live' | 'bundle' | 'empty'
  note?: string
  skipped: string[]
}

function unwrap(mod: unknown): unknown {
  if (mod && typeof mod === 'object' && 'default' in mod) return (mod as { default: unknown }).default
  return mod
}

function bundledEntries(): FileEntry[] {
  const modules = import.meta.glob('../../../logs/traces/*.json', { eager: true }) as Record<string, unknown>
  return Object.entries(modules).map(([file, mod]) => ({
    file: file.split('/').pop() || file,
    data: unwrap(mod),
  }))
}

function isTrace(value: unknown): value is RawTrace {
  if (!value || typeof value !== 'object') return false
  const record = value as RawTrace
  return typeof record.id === 'string' || Array.isArray(record.spans)
}

export function recordsFromEntries(traces: FileEntry[], reports: FileEntry[]): { records: EvalRecord[]; skipped: string[] } {
  const skipped: string[] = []
  const seen = new Set<string>()
  const records: EvalRecord[] = []
  for (const entry of traces) {
    if (entry.error || !isTrace(entry.data)) {
      if (entry.file || entry.error) skipped.push(entry.file || '未知文件')
      continue
    }
    const record = recordFromTrace({ trace: entry.data, source: 'trace-file', file: entry.file })
    let id = record.id
    if (seen.has(id)) id = `${id}__${entry.file || seen.size}`
    seen.add(id)
    records.push(id === record.id ? record : { ...record, id })
  }

  const byTrace = new Map<string, RawReport>()
  for (const entry of reports) {
    if (isRawReport(entry.data) && entry.data.traceId) byTrace.set(entry.data.traceId, entry.data)
  }
  const merged = records.map((record) => {
    const report = byTrace.get(record.trace.id)
    return report ? applyReport(record, report) : record
  })
  merged.sort((a, b) => Date.parse(b.createdAt) - Date.parse(a.createdAt))
  return { records: merged, skipped }
}

export async function loadTraces(): Promise<LoadResult> {
  try {
    const response = await fetch('/__local/evals', { cache: 'no-store' })
    if (response.ok) {
      const payload = (await response.json()) as Payload
      const traces = payload.traces ?? []
      const { records, skipped } = recordsFromEntries(traces, payload.reports ?? [])
      if (records.length > 0 || traces.length === 0) {
        return {
          records,
          origin: 'live',
          skipped,
          note: records.length ? undefined : 'logs/traces 里还没有 Trace。',
        }
      }
    }
  } catch {
    // 落到构建时打进包里的文件。
  }

  const bundled = recordsFromEntries(bundledEntries(), [])
  if (bundled.records.length) {
    return {
      records: bundled.records,
      origin: 'bundle',
      skipped: bundled.skipped,
      note: '没有连上 /__local/evals，先展示构建时打包的 Trace。用 ui 目录的 pnpm dev 或 pnpm preview 可以读到新文件。',
    }
  }
  return {
    records: [],
    origin: 'empty',
    skipped: [],
    note: '没有读到 logs/traces。确认 DefaultTracer 写过 JSON，再用 ui 目录的 pnpm dev 打开。',
  }
}
