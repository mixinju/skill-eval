import { useMemo, useState } from 'react'
import { GitCompareArrows, RefreshCw } from 'lucide-react'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Separator } from '@/components/ui/separator'
import { PassBadge, RunnerBadge, StatusBadge } from '@/components/StatusBadge'
import { NewRunDialog } from '@/components/NewRunDialog'
import { formatWhen } from '@/lib/format'
import { normalizeRunner, RUNNERS } from '@/lib/runners'
import { cn } from '@/lib/utils'
import type { EvalRecord, RunnerAvailability } from '@/types'

type RunnerFilter = 'all' | string
type OutcomeFilter = 'all' | 'pass' | 'fail'

export function Sidebar({
  records,
  selectedId,
  tray,
  runners,
  loading,
  apiOnline,
  note,
  onSelect,
  onHome,
  onRefresh,
  onToggleTray,
  onOpenCompare,
  onCreated,
}: {
  records: EvalRecord[]
  selectedId?: string
  tray: string[]
  runners: RunnerAvailability[]
  loading: boolean
  apiOnline: boolean
  note?: string
  onSelect: (id: string) => void
  onHome: () => void
  onRefresh: () => void
  onToggleTray: (id: string) => void
  onOpenCompare: () => void
  onCreated: (rec: EvalRecord) => void
}) {
  const [runner, setRunner] = useState<RunnerFilter>('all')
  const [outcome, setOutcome] = useState<OutcomeFilter>('all')
  const [query, setQuery] = useState('')

  const singles = useMemo(() => records.filter((item) => item.type !== 'compare'), [records])

  const filtered = singles.filter((rec) => {
    if (runner !== 'all' && normalizeRunner(rec.runner || rec.trace?.runner) !== runner) return false
    if (outcome === 'pass' && rec.report?.pass !== true) return false
    if (outcome === 'fail' && rec.report?.pass !== false) return false
    if (query.trim()) {
      const hay = [
        rec.id,
        rec.trace?.userPrompt,
        rec.trace?.model,
        rec.trace?.targetSkill,
        rec.report?.targetSkill,
        rec.runner,
      ]
        .filter(Boolean)
        .join(' ')
        .toLowerCase()
      if (!hay.includes(query.trim().toLowerCase())) return false
    }
    return true
  })

  return (
    <aside className="flex h-full min-h-0 flex-col border-r bg-card/70 backdrop-blur-sm">
      <div className="space-y-3 p-4">
        <div className="flex items-center gap-3">
          <button
            type="button"
            onClick={onHome}
            className="flex h-10 w-10 items-center justify-center rounded-lg bg-primary text-sm font-semibold text-primary-foreground shadow-sm transition hover:bg-primary/90"
            title="总览"
          >
            SE
          </button>
          <div className="min-w-0 flex-1">
            <h1 className="truncate text-base font-semibold tracking-tight">Skill Eval</h1>
            <p className="text-xs text-muted-foreground">
              {loading ? '加载中…' : `${singles.length} 条运行`}
              {' · '}
              <span className={apiOnline ? 'text-success' : 'text-destructive'}>
                {apiOnline ? 'API 在线' : 'API 离线'}
              </span>
            </p>
          </div>
          <Button variant="ghost" size="icon" onClick={onRefresh} title="刷新">
            <RefreshCw className={cn('h-4 w-4', loading && 'animate-spin')} />
          </Button>
        </div>

        <div className="flex items-center gap-2">
          <NewRunDialog runners={runners} onCreated={onCreated} />
          <Button
            variant="outline"
            size="sm"
            disabled={tray.length !== 2}
            onClick={onOpenCompare}
            className="flex-1"
          >
            <GitCompareArrows className="h-4 w-4" />
            对比 {tray.length}/2
          </Button>
        </div>

        <div className="grid grid-cols-4 gap-1 rounded-md bg-muted p-1">
          {([{ id: 'all', label: '全部' }, ...RUNNERS.map((r) => ({ id: r.id, label: r.short }))] as const).map((item) => (
            <button
              key={item.id}
              type="button"
              onClick={() => setRunner(item.id)}
              className={cn(
                'rounded-sm px-1 py-1.5 text-center text-[11px] font-medium transition',
                runner === item.id ? 'bg-card text-foreground shadow-sm' : 'text-muted-foreground hover:text-foreground',
              )}
            >
              {item.label}
            </button>
          ))}
        </div>

        <div className="flex gap-1">
          {(
            [
              { id: 'all', label: '全部结果' },
              { id: 'pass', label: '通过' },
              { id: 'fail', label: '未通过' },
            ] as const
          ).map((item) => (
            <button
              key={item.id}
              type="button"
              onClick={() => setOutcome(item.id)}
              className={cn(
                'rounded-md px-2 py-1 text-xs transition',
                outcome === item.id ? 'bg-accent text-accent-foreground' : 'text-muted-foreground hover:bg-muted',
              )}
            >
              {item.label}
            </button>
          ))}
        </div>

        <Input
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          placeholder="搜索指令、模型、skill"
          aria-label="搜索评测"
        />
        {note && <p className="text-xs text-muted-foreground">{note}</p>}
      </div>

      <Separator />

      <ScrollArea className="flex-1">
        <div className="space-y-1 p-2">
          {filtered.length === 0 ? (
            <p className="px-2 py-8 text-center text-sm text-muted-foreground">还没有匹配的评测记录</p>
          ) : (
            filtered.map((rec) => {
              const skill = rec.trace?.targetSkill || rec.report?.targetSkill || '未指定 skill'
              const prompt = rec.trace?.userPrompt || rec.report?.userPrompt || ''
              const selected = selectedId === rec.id
              const inTray = tray.includes(rec.id)
              return (
                <div
                  key={rec.id}
                  className={cn(
                    'group rounded-md border border-transparent p-2.5 transition hover:bg-accent/60',
                    selected && 'border-border bg-accent/80',
                  )}
                >
                  <button type="button" className="w-full text-left" onClick={() => onSelect(rec.id)}>
                    <div className="mb-1.5 flex flex-wrap items-center gap-1.5">
                      <PassBadge pass={rec.report?.pass} />
                      <StatusBadge status={rec.status} />
                      <RunnerBadge runner={rec.runner || rec.trace?.runner} />
                    </div>
                    <div className="truncate text-sm font-medium">{skill}</div>
                    <div className="mt-0.5 line-clamp-2 text-xs text-muted-foreground">{prompt || '（无提示词）'}</div>
                    <div className="mt-2 flex items-center justify-between text-[11px] text-muted-foreground">
                      <span>{rec.trace?.model || rec.report?.model || '—'}</span>
                      <time>{formatWhen(rec.createdAt)}</time>
                    </div>
                  </button>
                  <div className="mt-2 flex justify-end">
                    <Button
                      variant={inTray ? 'secondary' : 'ghost'}
                      size="sm"
                      className="h-7 px-2 text-xs"
                      onClick={() => onToggleTray(rec.id)}
                    >
                      {inTray ? '移出对比' : '加入对比'}
                    </Button>
                  </div>
                </div>
              )
            })
          )}
        </div>
      </ScrollArea>

      {tray.length > 0 && (
        <div className="border-t p-3">
          <div className="mb-1 text-xs font-medium text-muted-foreground">对比托盘</div>
          <div className="flex flex-wrap gap-1">
            {tray.map((id) => (
              <Badge key={id} variant="outline" className="font-mono">
                {id.slice(0, 8)}
              </Badge>
            ))}
          </div>
        </div>
      )}
    </aside>
  )
}
