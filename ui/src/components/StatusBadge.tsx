import { Badge } from '@/components/ui/badge'
import { runnerLabel, normalizeRunner } from '@/lib/runners'

export function StatusBadge({ status }: { status?: string }) {
  const value = (status || '').toLowerCase()
  if (value === 'succeeded' || value === 'success') return <Badge variant="success">已完成</Badge>
  if (value === 'running') return <Badge variant="warn">运行中</Badge>
  if (value === 'timeout') return <Badge variant="warn">超时</Badge>
  if (value === 'cancelled') return <Badge variant="muted">已取消</Badge>
  if (value === 'failed' || value === 'fail') return <Badge variant="fail">失败</Badge>
  return <Badge variant="muted">{status || '未知'}</Badge>
}

export function PassBadge({ pass }: { pass?: boolean | null }) {
  if (pass === true) return <Badge variant="success">通过</Badge>
  if (pass === false) return <Badge variant="fail">未通过</Badge>
  return <Badge variant="muted">待定</Badge>
}

export function RunnerBadge({ runner }: { runner?: string }) {
  const id = normalizeRunner(runner)
  const variant = id === 'self' ? 'secondary' : id === 'codex' ? 'outline' : 'muted'
  return <Badge variant={variant}>{runnerLabel(runner)}</Badge>
}
