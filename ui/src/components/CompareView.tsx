import { Badge } from '@/components/ui/badge'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { PassBadge, RunnerBadge } from '@/components/StatusBadge'
import { formatMs, formatScore, formatTokens, shortId } from '@/lib/format'
import type { CompareReport, EvalRecord, Report, Trace } from '@/types'

export function CompareView({
  left,
  right,
  compare,
}: {
  left?: EvalRecord
  right?: EvalRecord
  compare?: CompareReport
}) {
  const reportA = compare?.reportA || left?.report
  const reportB = compare?.reportB || right?.report
  const traceA = compare?.traceA || left?.trace
  const traceB = compare?.traceB || right?.trace

  if (!reportA || !reportB || !traceA || !traceB) {
    return <div className="p-8 text-sm text-muted-foreground">请选择两条已完成的评测进行对比。</div>
  }

  return (
    <div className="mx-auto max-w-6xl space-y-5 p-6 md:p-8">
      <header className="space-y-2">
        <p className="text-sm font-medium text-primary">对比评测</p>
        <h2 className="text-2xl font-semibold tracking-tight">
          {traceA.model} vs {traceB.model}
        </h2>
        <p className="max-w-3xl text-sm text-muted-foreground">
          {compare?.conclusion || '基于迭代、Token、工具错误与已有产物分的确定性对比。'}
        </p>
      </header>

      <div className="grid gap-4 md:grid-cols-2">
        <SideCard title="A" report={reportA} trace={traceA} />
        <SideCard title="B" report={reportB} trace={traceB} />
      </div>

      {compare?.scores?.length ? (
        <Card>
          <CardHeader>
            <CardTitle>对比分数</CardTitle>
            <CardDescription>source=deterministic</CardDescription>
          </CardHeader>
          <CardContent className="space-y-3">
            {compare.scores.map((score, index) => (
              <div key={`${score.info.name}-${index}`} className="flex items-start justify-between gap-3 rounded-md border px-3 py-2">
                <div>
                  <div className="font-medium">{score.info.name}</div>
                  <p className="text-sm text-muted-foreground">{score.reason}</p>
                </div>
                <div className="flex items-center gap-2">
                  <PassBadge pass={score.pass} />
                  <span className="font-mono text-sm">{formatScore(score.score)}</span>
                </div>
              </div>
            ))}
          </CardContent>
        </Card>
      ) : null}
    </div>
  )
}

function SideCard({ title, report, trace }: { title: string; report: Report; trace: Trace }) {
  return (
    <Card>
      <CardHeader>
        <div className="flex flex-wrap items-center gap-2">
          <Badge variant="outline">{title}</Badge>
          <PassBadge pass={report.pass} />
          <RunnerBadge runner={report.runner || trace.runner} />
        </div>
        <CardTitle className="text-base">{trace.targetSkill || report.targetSkill || '未指定 skill'}</CardTitle>
        <CardDescription>
          {trace.model} · {shortId(trace.id)}
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-3">
        <p className="line-clamp-3 text-sm text-muted-foreground">{trace.userPrompt}</p>
        <div className="grid grid-cols-3 gap-2 text-center">
          <div className="rounded-md border p-2">
            <div className="font-semibold">{trace.iterations}</div>
            <div className="text-xs text-muted-foreground">迭代</div>
          </div>
          <div className="rounded-md border p-2">
            <div className="font-semibold">{formatTokens(trace.totalTokens)}</div>
            <div className="text-xs text-muted-foreground">Token</div>
          </div>
          <div className="rounded-md border p-2">
            <div className="font-semibold">{formatMs(report.duration)}</div>
            <div className="text-xs text-muted-foreground">耗时</div>
          </div>
        </div>
        <ul className="space-y-1">
          {report.scores.map((score, index) => (
            <li key={`${score.info.name}-${index}`} className="flex items-center justify-between text-sm">
              <span>{score.info.name}</span>
              <span className="font-mono text-muted-foreground">{formatScore(score.score)}</span>
            </li>
          ))}
        </ul>
      </CardContent>
    </Card>
  )
}
