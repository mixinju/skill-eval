import { useMemo, useState } from 'react'
import type { EvalRecord, Span } from '../types'
import { formatClock, kindLabel } from '../lib/format'
import { parseAssistant } from '../lib/model'

type LogFilter = 'all' | 'llm_call' | 'tool_call' | 'error' | 'target'

function preview(span: Span): string {
  if (span.error) return span.error
  if (span.kind === 'tool_call') {
    const text = span.toolOutput || span.toolInput || ''
    return text.length > 240 ? `${text.slice(0, 240)}…` : text
  }
  const assistant = parseAssistant(span.llmOutput)
  if (!assistant) return ''
  if (assistant.content) return assistant.content.length > 240 ? `${assistant.content.slice(0, 240)}…` : assistant.content
  if (assistant.toolCalls.length) return `将调用 ${assistant.toolCalls.map((call) => call.name).join('、')}`
  return ''
}

export default function LogView({ record }: { record: EvalRecord }) {
  const [filter, setFilter] = useState<LogFilter>('all')
  const { trace } = record
  const lines = useMemo(() => {
    return trace.spans.map((span, index) => ({
      id: `${trace.id}-${index}`,
      time: span.startTime,
      iteration: span.iteration,
      kind: span.kind,
      error: Boolean(span.error),
      target: Boolean(span.isTarget),
      title: `${kindLabel(span.kind)}  ${span.name}`,
      body: preview(span),
    }))
  }, [trace])

  const visible = lines.filter((line) => {
    if (filter === 'all') return true
    if (filter === 'error') return line.error
    if (filter === 'target') return line.target
    return line.kind === filter
  })

  if (!trace.spans.length) return <p className="se-empty-inline">没有可还原的日志。事件流本身没有单独落盘，这里只能根据 Span 列出来。</p>

  const filters: { id: LogFilter; label: string }[] = [
    { id: 'all', label: '全部' },
    { id: 'llm_call', label: 'LLM' },
    { id: 'tool_call', label: '工具' },
    { id: 'error', label: '错误' },
    { id: 'target', label: '目标' },
  ]

  return (
    <div>
      <p className="se-section-note">由 Span 还原。原始的 run_start / tool_start 事件没有单独保存，完整输入输出在「过程」里。</p>
      <div className="se-filters">
        {filters.map((item) => (
          <button key={item.id} type="button" className="se-chip" aria-pressed={filter === item.id} onClick={() => setFilter(item.id)}>
            {item.label}
          </button>
        ))}
      </div>
      {visible.length === 0 && <p className="se-empty-inline">这个筛选下没有日志。</p>}
      <ol className="se-log">
        {visible.map((line) => (
          <li key={line.id} className={line.error ? 'error' : line.target ? 'target' : undefined}>
            <time>{formatClock(line.time)}</time>
            <span className="se-log-iter">R{line.iteration}</span>
            <div>
              <b>{line.title}</b>
              {line.body && <p>{line.body}</p>}
            </div>
          </li>
        ))}
      </ol>
    </div>
  )
}
