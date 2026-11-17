import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { PassBadge, RunnerBadge, StatusBadge } from '@/components/StatusBadge'
import { formatWhen } from '@/lib/format'
import { normalizeRunner, RUNNERS } from '@/lib/runners'
import type { EvalRecord, RunnerAvailability } from '@/types'

export function Overview({
  records,
  runners,
  onOpen,
  onFilterRunner,
}: {
  records: EvalRecord[]
  runners: RunnerAvailability[]
  onOpen: (id: string) => void
  onFilterRunner: (id: string) => void
}) {
  const singles = records.filter((item) => item.type !== 'compare')
  const recent = singles.slice(0, 8)

  return (
    <div className="mx-auto max-w-5xl space-y-8 p-6 md:p-10">
      <header className="space-y-3">
        <p className="text-sm font-medium tracking-[0.18em] text-primary uppercase">Skill Eval</p>
        <h2 className="max-w-2xl text-3xl font-semibold tracking-tight text-foreground md:text-4xl">
          评测控制台
        </h2>
        <p className="max-w-2xl text-sm leading-relaxed text-muted-foreground md:text-base">
          查看任务状态、评分、执行过程、日志与产物。自研内核、Codex 与 Claude Code 共用同一套 Trace / Report 契约。
        </p>
      </header>

      <section className="grid gap-3 md:grid-cols-3">
        {RUNNERS.map((runner) => {
          const probe = runners.find((item) => normalizeRunner(String(item.type)) === runner.id)
          const count = singles.filter((rec) => normalizeRunner(rec.runner || rec.trace?.runner) === runner.id).length
          return (
            <button
              key={runner.id}
              type="button"
              onClick={() => onFilterRunner(runner.id)}
              className="rounded-lg border bg-card/80 p-4 text-left shadow-sm transition hover:-translate-y-0.5 hover:border-primary/30 hover:shadow-md"
            >
              <div className="mb-3 flex items-center justify-between gap-2">
                <RunnerBadge runner={runner.id} />
                <StatusBadge status={probe?.available ? 'succeeded' : 'failed'} />
              </div>
              <div className="text-lg font-semibold">{runner.label}</div>
              <p className="mt-1 text-sm text-muted-foreground">{count} 条记录</p>
              <p className="mt-3 line-clamp-2 text-xs text-muted-foreground">
                {probe?.detail || '尚未探测到该执行后端的状态。'}
              </p>
            </button>
          )
        })}
      </section>

      <Card>
        <CardHeader>
          <CardTitle>最近运行</CardTitle>
          <CardDescription>来自评测 API；服务启动时也会导入历史 logs/traces。</CardDescription>
        </CardHeader>
        <CardContent>
          {recent.length === 0 ? (
            <p className="py-8 text-center text-sm text-muted-foreground">还没有评测记录。可点击「新建评测」提交任务。</p>
          ) : (
            <ul className="divide-y">
              {recent.map((rec) => (
                <li key={rec.id}>
                  <button
                    type="button"
                    className="flex w-full items-start gap-3 py-3 text-left transition hover:bg-muted/40"
                    onClick={() => onOpen(rec.id)}
                  >
                    <PassBadge pass={rec.report?.pass} />
                    <div className="min-w-0 flex-1">
                      <div className="flex flex-wrap items-center gap-2">
                        <span className="font-medium">
                          {rec.trace?.targetSkill || rec.report?.targetSkill || '未指定 skill'}
                        </span>
                        <RunnerBadge runner={rec.runner || rec.trace?.runner} />
                        <StatusBadge status={rec.status} />
                      </div>
                      <p className="mt-1 line-clamp-1 text-sm text-muted-foreground">
                        {rec.trace?.userPrompt || rec.report?.userPrompt || '（无提示词）'}
                      </p>
                    </div>
                    <time className="shrink-0 text-xs text-muted-foreground">{formatWhen(rec.createdAt)}</time>
                  </button>
                </li>
              ))}
            </ul>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
