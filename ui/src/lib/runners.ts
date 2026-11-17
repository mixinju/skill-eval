import type { RunnerId } from '@/types'

export interface RunnerMetaView {
  id: RunnerId
  label: string
  short: string
}

export const RUNNERS: RunnerMetaView[] = [
  { id: 'self', label: '自研内核', short: '内核' },
  { id: 'codex', label: 'Codex', short: 'Codex' },
  { id: 'claude-code', label: 'Claude Code', short: 'Claude' },
]

export function normalizeRunner(raw?: string): RunnerId {
  const value = (raw || '').toLowerCase().trim()
  if (value === 'codex') return 'codex'
  if (value === 'claude' || value === 'claude-code' || value === 'claudecode') return 'claude-code'
  return 'self'
}

export function runnerLabel(raw?: string): string {
  const id = normalizeRunner(raw)
  return RUNNERS.find((item) => item.id === id)?.label ?? '自研内核'
}
