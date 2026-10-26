import { useMemo, useState } from 'react'
import type { EvalRecord, RunnerId } from '../types'
import { cx, formatMs, formatWhen } from '../lib/format'
import { traceStats } from '../lib/model'
import { RUNNERS } from '../lib/runners'
import { Mark, RunnerBadge } from './bits'

type RunnerFilter = 'all' | RunnerId
type OutcomeFilter = 'all' | 'pass' | 'fail'

interface SidebarProps {
  records: EvalRecord[]
  selectedId?: string
  tray: string[]
  loading: boolean
  note?: string
  theme: 'dark' | 'light'
  onSelect: (id: string) => void
  onToggleTray: (id: string) => void
  onHome: () => void
  onOpenCompare: () => void
  onClearTray: () => void
  onToggleTheme: () => void
  onOpenGaps: () => void
}

export default function Sidebar({
  records,
  selectedId,
  tray,
  loading,
  note,
  theme,
  onSelect,
  onToggleTray,
  onHome,
  onOpenCompare,
  onClearTray,
  onToggleTheme,
  onOpenGaps,
}: SidebarProps) {
  const [runner, setRunner] = useState<RunnerFilter>('all')
  const [outcome, setOutcome] = useState<OutcomeFilter>('all')
  const [skill, setSkill] = useState<string>('')
  const [query, setQuery] = useState('')
  const [showFixtures, setShowFixtures] = useState(true)

  const corpus = useMemo(
    () => (showFixtures ? records : records.filter((record) => record.source !== 'fixture')),
    [records, showFixtures],
  )

  const skills = useMemo(() => {
    return [...new Set(corpus.map((record) => record.trace.targetSkill).filter(Boolean))].sort()
  }, [corpus])

  const counts = useMemo(() => {
    const tally: Record<RunnerFilter, number> = { all: corpus.length, kernel: 0, codex: 0, 'claude-code': 0 }
    for (const record of corpus) tally[record.runner] += 1
    return tally
  }, [corpus])

  const filtered = corpus.filter((record) => {
    if (runner !== 'all' && record.runner !== runner) return false
    if (outcome === 'pass' && record.outcome !== 'pass') return false
    if (outcome === 'fail' && record.outcome !== 'fail') return false
    if (skill && record.trace.targetSkill !== skill) return false
    if (query.trim()) {
      const hay = `${record.trace.userPrompt} ${record.trace.model} ${record.trace.agentName} ${record.trace.targetSkill} ${record.id}`.toLowerCase()
      if (!hay.includes(query.trim().toLowerCase())) return false
    }
    return true
  })

  const runnerOptions: { id: RunnerFilter; label: string }[] = [
    { id: 'all', label: '全部' },
    ...RUNNERS.map((item) => ({ id: item.id, label: item.short })),
  ]

  return (
    <aside className="se-side">
      <div className="se-side-top">
        <div className="se-brand">
          <button type="button" className="se-brand-mark" onClick={onHome} title="回到总览">
            SE
          </button>
          <div>
            <h1>Skill Eval</h1>
            <p>{loading ? '正在读取 Trace' : `${corpus.length} 条记录`}</p>
          </div>
          <button type="button" className="se-icon-btn" onClick={onToggleTheme}>
            {theme === 'dark' ? '浅色' : '深色'}
          </button>
        </div>

        <div className="se-runners" role="group" aria-label="执行后端">
          {runnerOptions.map((option) => (
            <button
              key={option.id}
              type="button"
              aria-pressed={runner === option.id}
              onClick={() => setRunner(option.id)}
            >
              <b>{option.label}</b>
              <small>{counts[option.id]}</small>
            </button>
          ))}
        </div>

        <input
          className="se-search"
          value={query}
          placeholder="搜索指令、模型、skill"
          onChange={(event) => setQuery(event.target.value)}
          aria-label="搜索评测"
        />

        <div className="se-filters">
          {(['all', 'pass', 'fail'] as const).map((item) => (
            <button key={item} type="button" className="se-chip" aria-pressed={outcome === item} onClick={() => setOutcome(item)}>
              {item === 'all' ? '全部结果' : item === 'pass' ? '通过' : '未通过'}
            </button>
          ))}
          <button type="button" className="se-chip" aria-pressed={showFixtures} onClick={() => setShowFixtures((value) => !value)}>
            样例
          </button>
        </div>

        {skills.length > 1 && (
          <div className="se-filters">
            <button type="button" className="se-chip" aria-pressed={skill === ''} onClick={() => setSkill('')}>
              全部 skill
            </button>
            {skills.map((item) => (
              <button key={item} type="button" className="se-chip" aria-pressed={skill === item} onClick={() => setSkill(item)}>
                {item}
              </button>
            ))}
          </div>
        )}
      </div>

      <div className="se-list">
        {filtered.length === 0 && (
          <div className="se-empty">
            <p>没有符合条件的评测。</p>
            {runner !== 'all' && runner !== 'kernel' && !showFixtures && (
              <button type="button" className="se-textbtn" onClick={() => setShowFixtures(true)}>
                显示 {RUNNERS.find((item) => item.id === runner)?.label} 的界面样例
              </button>
            )}
            {runner !== 'all' && runner !== 'kernel' && showFixtures && (
              <p className="se-muted">这个执行后端还没有真实 Trace。</p>
            )}
          </div>
        )}
        {filtered.map((record) => {
          const stats = traceStats(record.trace)
          const inTray = tray.includes(record.id)
          return (
            <div key={record.id} className={cx('se-card', selectedId === record.id && 'selected', inTray && 'in-tray', record.source === 'fixture' && 'fixture')}>
              <button type="button" className="se-card-main" onClick={() => onSelect(record.id)}>
                <div className="se-card-top">
                  <Mark outcome={record.outcome} />
                  <RunnerBadge id={record.runner} inferred={record.runnerInferred} short />
                  {record.source === 'fixture' && <span className="se-badge fixture">样例</span>}
                </div>
                <div className="se-card-title">
                  {record.trace.targetSkill || '未指定 skill'}
                  <span> · {record.trace.model}</span>
                </div>
                <p className="se-card-prompt">{record.trace.userPrompt || '（没有用户指令）'}</p>
                <div className="se-card-meta">
                  <span>{record.trace.iterations} 轮</span>
                  <span>{stats.durationMs === null ? '—' : formatMs(stats.durationMs)}</span>
                  <span>{formatWhen(record.createdAt)}</span>
                </div>
              </button>
              <button
                type="button"
                className={cx('se-card-compare', inTray && 'on')}
                aria-pressed={inTray}
                onClick={() => onToggleTray(record.id)}
              >
                {inTray ? '已选' : '对比'}
              </button>
            </div>
          )
        })}
      </div>

      <div className="se-side-foot">
        {tray.length > 0 && (
          <div className="se-tray">
            <span>对比 {tray.length}/2</span>
            {tray.length === 2 && (
              <button type="button" className="se-textbtn" onClick={onOpenCompare}>
                查看
              </button>
            )}
            <button type="button" className="se-textbtn" onClick={onClearTray}>
              清除
            </button>
          </div>
        )}
        {note && <p className="se-foot-note">{note}</p>}
        <button type="button" className="se-textbtn" onClick={onOpenGaps}>
          数据从哪来
        </button>
      </div>
    </aside>
  )
}
