import type { RunnerId } from '../types'

export interface RunnerMeta {
  id: RunnerId
  label: string
  short: string
  summary: string
  implemented: boolean
}

export const RUNNERS: RunnerMeta[] = [
  {
    id: 'kernel',
    label: '自研内核',
    short: '内核',
    summary: 'agent.Orchestrator。这是仓库里唯一会执行评测并把 Trace 写到 logs/traces 的后端。',
    implemented: true,
  },
  {
    id: 'codex',
    label: 'Codex',
    short: 'Codex',
    summary: '契约预留了 runner: "codex"。没有执行器，也没有真实 Trace。列表中的 Codex 记录是界面样例。',
    implemented: false,
  },
  {
    id: 'claude-code',
    label: 'Claude Code',
    short: 'Claude',
    summary: '契约预留了 runner: "claude-code"。没有执行器，也没有真实 Trace。列表中的 Claude Code 记录是界面样例。',
    implemented: false,
  },
]

const RUNNER_SET = new Set<string>(RUNNERS.map((runner) => runner.id))

export function isRunnerId(value: string): value is RunnerId {
  return RUNNER_SET.has(value)
}

export function runnerMeta(id: RunnerId): RunnerMeta {
  return RUNNERS.find((runner) => runner.id === id) ?? RUNNERS[0]
}

export function runnerClass(id: RunnerId): string {
  if (id === 'claude-code') return 'claude'
  return id
}
