import { useEffect, useState } from 'react'
import { Download, Loader2 } from 'lucide-react'
import { Accordion, AccordionContent, AccordionItem, AccordionTrigger } from '@/components/ui/accordion'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { PassBadge, RunnerBadge, StatusBadge } from '@/components/StatusBadge'
import {
  artifactFileURL,
  fetchArtifacts,
  fetchEval,
  fetchLogs,
} from '@/lib/api'
import {
  fileName,
  formatBytes,
  formatMs,
  formatScore,
  formatTokens,
  formatWhen,
  pretty,
  shortId,
} from '@/lib/format'
import type { ArtifactsResponse, EvalRecord, Span } from '@/types'

function spanDuration(span: Span): string {
  if (span.startTime && span.endTime) {
    const ms = new Date(span.endTime).getTime() - new Date(span.startTime).getTime()
    if (Number.isFinite(ms) && ms >= 0) return formatMs(ms)
  }
  if (typeof span.duration === 'number') {
    // 旧文件可能是纳秒
    return formatMs(span.duration > 1e10 ? span.duration / 1e6 : span.duration)
  }
  return '—'
}

export function RunDetail({
  recordId,
  seed,
  inTray,
  onToggleTray,
  onLoaded,
}: {
  recordId: string
  seed?: EvalRecord
  inTray: boolean
  onToggleTray: () => void
  onLoaded?: (rec: EvalRecord) => void
}) {
  const [record, setRecord] = useState<EvalRecord | undefined>(seed)
  const [logs, setLogs] = useState('')
  const [artifacts, setArtifacts] = useState<ArtifactsResponse>()
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string>()

  useEffect(() => {
    let cancelled = false
    setLoading(true)
    setError(undefined)
    ;(async () => {
      try {
        const full = await fetchEval(recordId)
        if (cancelled) return
        setRecord(full)
        onLoaded?.(full)
        const [logText, arts] = await Promise.all([
          fetchLogs(recordId).catch(() => ''),
          fetchArtifacts(recordId).catch(() => ({ artifacts: [] as ArtifactsResponse['artifacts'] })),
        ])
        if (cancelled) return
        setLogs(logText)
        setArtifacts(arts)
      } catch (err) {
        if (!cancelled) setError(err instanceof Error ? err.message : '加载失败')
      } finally {
        if (!cancelled) setLoading(false)
      }
    })()
    return () => {
      cancelled = true
    }
    // 只在切换评测 id 时重新拉取；onLoaded 用于回写列表缓存。
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [recordId])

  if (loading && !record) {
    return (
      <div className="flex h-full items-center justify-center gap-2 text-sm text-muted-foreground">
        <Loader2 className="h-4 w-4 animate-spin" />
        加载评测详情…
      </div>
    )
  }

  if (error && !record) {
    return <div className="p-8 text-sm text-destructive">{error}</div>
  }

  if (!record?.trace || !record.report) {
    return <div className="p-8 text-sm text-muted-foreground">这条记录还没有完整的 Trace / Report。</div>
  }

  const { trace, report } = record
  const tools = trace.spans.filter((span) => span.kind === 'tool_call')

  return (
    <div className="mx-auto max-w-5xl space-y-5 p-6 md:p-8">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div className="space-y-2">
          <div className="flex flex-wrap items-center gap-2">
            <PassBadge pass={report.pass} />
            <StatusBadge status={record.status || report.status || trace.status} />
            <RunnerBadge runner={record.runner || report.runner || trace.runner} />
            {report.scores.some((s) => s.source === 'deterministic') && (
              <Badge variant="muted">含确定性评分</Badge>
            )}
          </div>
          <h2 className="text-2xl font-semibold tracking-tight">
            {trace.targetSkill || report.targetSkill || '未指定 skill'}
          </h2>
          <p className="text-sm text-muted-foreground">
            {trace.model} · Trace {shortId(trace.id)} · {formatWhen(record.createdAt)}
          </p>
        </div>
        <Button variant="outline" size="sm" onClick={onToggleTray}>
          {inTray ? '移出对比' : '加入对比'}
        </Button>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>任务</CardTitle>
          <CardDescription>{trace.agentName || report.agentName}</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <blockquote className="rounded-md border bg-muted/40 px-4 py-3 text-sm leading-relaxed">
            {trace.userPrompt || '（没有用户指令）'}
          </blockquote>
          {(trace.failureReason || record.error) && (
            <p className="rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm text-destructive">
              {trace.failureReason || record.error}
            </p>
          )}
          <div className="grid grid-cols-2 gap-3 sm:grid-cols-5">
            <Metric label="迭代" value={String(trace.iterations)} />
            <Metric label="Token" value={formatTokens(trace.totalTokens)} />
            <Metric label="耗时" value={formatMs(report.duration)} />
            <Metric label="工具" value={String(tools.length)} />
            <Metric label="LLM" value={String(trace.spans.filter((s) => s.kind === 'llm_call').length)} />
          </div>
          {tools.length > 0 && (
            <ol className="flex flex-wrap gap-2">
              {tools.map((span) => (
                <li
                  key={span.spanId}
                  className={`rounded-md border px-2.5 py-1 text-xs ${
                    span.error ? 'border-red-200 bg-red-50 text-destructive' : span.isTarget ? 'border-primary/30 bg-accent text-accent-foreground' : 'bg-card'
                  }`}
                >
                  <span className="font-medium">{span.name}</span>
                  <span className="ml-2 text-muted-foreground">{spanDuration(span)}</span>
                </li>
              ))}
            </ol>
          )}
        </CardContent>
      </Card>

      <Tabs defaultValue="scores">
        <TabsList>
          <TabsTrigger value="scores">评分</TabsTrigger>
          <TabsTrigger value="trace">执行过程</TabsTrigger>
          <TabsTrigger value="artifacts">产物</TabsTrigger>
          <TabsTrigger value="logs">日志</TabsTrigger>
        </TabsList>

        <TabsContent value="scores" className="space-y-3">
          {report.scores.map((score, index) => (
            <Card key={`${score.info.name}-${index}`}>
              <CardHeader className="pb-3">
                <div className="flex flex-wrap items-center justify-between gap-2">
                  <CardTitle className="text-sm">{score.info.name}</CardTitle>
                  <div className="flex items-center gap-2">
                    {score.source && <Badge variant="outline">{score.source}</Badge>}
                    <PassBadge pass={score.pass} />
                    <span className="font-mono text-sm">{formatScore(score.score)}</span>
                  </div>
                </div>
                <CardDescription>{score.info.desc}</CardDescription>
              </CardHeader>
              <CardContent>
                <p className="text-sm leading-relaxed text-muted-foreground">{score.reason || '—'}</p>
                {score.error && <p className="mt-2 text-sm text-destructive">{score.error}</p>}
              </CardContent>
            </Card>
          ))}
        </TabsContent>

        <TabsContent value="trace">
          <Card>
            <CardHeader>
              <CardTitle>Span 时间线</CardTitle>
              <CardDescription>按迭代展开模型输出与工具输入输出。</CardDescription>
            </CardHeader>
            <CardContent>
              {trace.spans.length === 0 ? (
                <p className="text-sm text-muted-foreground">没有 Span。</p>
              ) : (
                <Accordion type="multiple" className="w-full">
                  {trace.spans.map((span) => (
                    <AccordionItem key={span.spanId} value={span.spanId}>
                      <AccordionTrigger>
                        <div className="flex min-w-0 flex-1 flex-wrap items-center gap-2 pr-3">
                          <Badge variant="outline">{span.kind}</Badge>
                          <span className="font-medium">{span.name}</span>
                          <span className="text-xs text-muted-foreground">第 {span.iteration} 轮</span>
                          <span className="text-xs text-muted-foreground">{spanDuration(span)}</span>
                          {span.isTarget && <Badge variant="success">目标 skill</Badge>}
                          {span.error && <Badge variant="fail">错误</Badge>}
                        </div>
                      </AccordionTrigger>
                      <AccordionContent>
                        <div className="space-y-3">
                          {span.error && <p className="text-sm text-destructive">{span.error}</p>}
                          {span.llmOutput && <PreBlock title="模型输出" value={span.llmOutput} />}
                          {span.toolInput && <PreBlock title="工具输入" value={span.toolInput} />}
                          {span.toolOutput && <PreBlock title="工具输出" value={span.toolOutput} />}
                          {!span.llmOutput && !span.toolInput && !span.toolOutput && !span.error && (
                            <p className="text-sm text-muted-foreground">这一步没有额外正文。</p>
                          )}
                        </div>
                      </AccordionContent>
                    </AccordionItem>
                  ))}
                </Accordion>
              )}
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="artifacts">
          <Card>
            <CardHeader>
              <CardTitle>产物与最终输出</CardTitle>
              <CardDescription>
                {artifacts?.workspace ? `工作区 ${artifacts.workspace}` : '来自 finish / artifactsAndResult'}
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              {artifacts?.result && (
                <div>
                  <h4 className="mb-2 text-sm font-medium">最终输出</h4>
                  <pre className="overflow-auto rounded-md border bg-muted/40 p-3 font-mono text-xs leading-relaxed">
                    {artifacts.result}
                  </pre>
                </div>
              )}
              {!artifacts?.artifacts?.length ? (
                <p className="text-sm text-muted-foreground">没有记录产物文件。</p>
              ) : (
                <ul className="space-y-2">
                  {artifacts.artifacts.map((item) => (
                    <li key={item.path} className="flex items-center justify-between gap-3 rounded-md border px-3 py-2">
                      <div className="min-w-0">
                        <div className="truncate text-sm font-medium">{fileName(item.path)}</div>
                        <div className="truncate font-mono text-xs text-muted-foreground">
                          {item.path}
                          {item.exists ? ` · ${formatBytes(item.size)}` : ' · 文件不在工作区'}
                        </div>
                      </div>
                      {item.exists && (
                        <Button asChild variant="outline" size="sm">
                          <a href={artifactFileURL(record.id, item.path)}>
                            <Download className="h-4 w-4" />
                            下载
                          </a>
                        </Button>
                      )}
                    </li>
                  ))}
                </ul>
              )}
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="logs">
          <Card>
            <CardHeader>
              <CardTitle>运行日志</CardTitle>
              <CardDescription>{record.logPath || trace.runnerMeta?.logPath || 'stdout / 步骤摘要'}</CardDescription>
            </CardHeader>
            <CardContent>
              {logs ? (
                <pre className="max-h-[28rem] overflow-auto rounded-md border bg-[#122019] p-4 font-mono text-xs leading-relaxed text-[#d7ece2]">
                  {logs}
                </pre>
              ) : (
                <div className="space-y-2">
                  <p className="text-sm text-muted-foreground">没有独立日志文件，展示由 Span 压缩的步骤。</p>
                  <ul className="space-y-2">
                    {(trace.steps || []).map((step) => (
                      <li key={step.index} className="rounded-md border px-3 py-2 text-sm">
                        <div className="flex flex-wrap items-center gap-2">
                          <Badge variant="outline">#{step.index}</Badge>
                          <span className="font-medium">{step.name}</span>
                          <Badge variant={step.ok ? 'success' : 'fail'}>{step.ok ? 'ok' : 'err'}</Badge>
                        </div>
                        {step.summary && <p className="mt-1 text-xs text-muted-foreground">{step.summary}</p>}
                        {step.error && <p className="mt-1 text-xs text-destructive">{step.error}</p>}
                      </li>
                    ))}
                  </ul>
                </div>
              )}
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </div>
  )
}

function Metric({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-md border bg-card px-3 py-2">
      <div className="text-lg font-semibold tracking-tight">{value}</div>
      <div className="text-xs text-muted-foreground">{label}</div>
    </div>
  )
}

function PreBlock({ title, value }: { title: string; value: string }) {
  return (
    <div>
      <div className="mb-1 text-xs font-medium text-muted-foreground">{title}</div>
      <pre className="overflow-auto rounded-md border bg-muted/40 p-3 font-mono text-xs leading-relaxed">
        {pretty(value)}
      </pre>
    </div>
  )
}
