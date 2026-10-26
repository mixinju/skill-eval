import { useState } from 'react'
import type { Outcome, RunnerId, ScoreAvailability } from '../types'
import { cx } from '../lib/format'
import { runnerClass, runnerMeta } from '../lib/runners'

export function Mark({ outcome, large = false }: { outcome: Outcome; large?: boolean }) {
  const label = outcome === 'pass' ? '通过' : outcome === 'fail' ? '未通过' : '待定'
  return <span className={cx('se-mark', outcome, large && 'lg')}>{label}</span>
}

export function RunnerBadge({ id, inferred = false, short = false }: { id: RunnerId; inferred?: boolean; short?: boolean }) {
  const meta = runnerMeta(id)
  return (
    <span className={cx('se-badge', runnerClass(id))} title={meta.summary}>
      {short ? meta.short : meta.label}
      {inferred ? ' · 未标注' : ''}
    </span>
  )
}

export function Availability({ value }: { value: ScoreAvailability }) {
  const label = value === 'derived' ? 'Trace' : value === 'persisted' ? '报告' : value === 'fixture' ? '样例' : '未落盘'
  return <span className={cx('se-badge', 'muted', value === 'fixture' && 'fixture')}>{label}</span>
}

export function Reveal({ text, limit = 700 }: { text: string; limit?: number }) {
  const [open, setOpen] = useState(false)
  if (!text) return <p className="se-muted">（空）</p>
  const clipped = !open && text.length > limit
  return (
    <div>
      <pre className="se-pre">{clipped ? `${text.slice(0, limit)}…` : text}</pre>
      {text.length > limit && (
        <button type="button" className="se-textbtn" onClick={() => setOpen((value) => !value)}>
          {open ? '收起' : `展开全文（${text.length.toLocaleString('zh-CN')} 字）`}
        </button>
      )}
    </div>
  )
}
