import type {
  ArtifactsResponse,
  EvalRecord,
  RunnerAvailability,
} from '@/types'

async function readJSON<T>(res: Response): Promise<T> {
  if (!res.ok) {
    let message = `请求失败 (${res.status})`
    try {
      const body = (await res.json()) as { error?: string }
      if (body.error) message = body.error
    } catch {
      // ignore
    }
    throw new Error(message)
  }
  return (await res.json()) as T
}

export async function fetchHealth(): Promise<boolean> {
  try {
    const res = await fetch('/api/health')
    return res.ok
  } catch {
    return false
  }
}

export async function fetchRunners(): Promise<RunnerAvailability[]> {
  const data = await readJSON<{ runners: RunnerAvailability[] }>(await fetch('/api/runners'))
  return data.runners ?? []
}

export async function fetchEvals(): Promise<EvalRecord[]> {
  const data = await readJSON<{ records: EvalRecord[] }>(await fetch('/api/evals'))
  return data.records ?? []
}

export async function fetchEval(id: string): Promise<EvalRecord> {
  return readJSON<EvalRecord>(await fetch(`/api/evals/${encodeURIComponent(id)}`))
}

export async function fetchLogs(id: string): Promise<string> {
  const res = await fetch(`/api/evals/${encodeURIComponent(id)}/logs`)
  if (res.status === 404) return ''
  if (!res.ok) throw new Error('读取日志失败')
  return res.text()
}

export async function fetchArtifacts(id: string): Promise<ArtifactsResponse> {
  return readJSON<ArtifactsResponse>(await fetch(`/api/evals/${encodeURIComponent(id)}/artifacts`))
}

export function artifactFileURL(id: string, path: string): string {
  const q = new URLSearchParams({ path })
  return `/api/evals/${encodeURIComponent(id)}/artifacts/file?${q.toString()}`
}

export interface CreateEvalInput {
  runner: string
  name?: string
  systemPrompt?: string
  userPrompt: string
  model?: string
  targetSkill?: string
  skillDir?: string
  maxIterations?: number
  timeoutSec?: number
  wait?: boolean
}

export async function createEval(input: CreateEvalInput): Promise<EvalRecord> {
  const res = await fetch('/api/evals', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(input),
  })
  return readJSON<EvalRecord>(res)
}

export async function compareEvals(evalIdA: string, evalIdB: string): Promise<EvalRecord> {
  const res = await fetch('/api/evals/compare', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ evalIdA, evalIdB }),
  })
  return readJSON<EvalRecord>(res)
}
