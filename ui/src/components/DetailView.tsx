import type { EvalRecord } from '../types'
import { formatMs, formatScore, formatTokens, pretty, shortId, spanDurationLabel } from '../lib/format'
import { outcomeCaption, traceStats } from '../lib/model'
import { RunnerBadge } from './bits'
import LogView from './LogView'
import ProcessView from './ProcessView'
import ScoreList from './ScoreList'

function fileName(path: string): string {
  const parts = path.split(/[/\\]/)
  return parts[parts.length - 1] || path
}

function scrollTo(id: string) {
  document.getElementById(id)?.scrollIntoView({ behavior: 'smooth', block: 'start' })
}

export default function DetailView({
  record,
  inTray,
  onToggleTray,
}: {
  record: EvalRecord
  inTray: boolean
  onToggleTray: () => void
}) {
  const stats = traceStats(record.trace)
  const tools = record.trace.spans.filter((span) => span.kind === 'tool_call')
  const caption = outcomeCaption(record)

  return (
    <div className="se-page">
      <nav className="se-nav">
        <button type="button" onClick={() => scrollTo('result')}>结果</button>
        <button type="button" onClick={() => scrollTo('scores')}>评分</button>
        <button type="button" onClick={() => scrollTo('process')}>过程</button>
        <button type="button" onClick={() => scrollTo('log')}>日志</button>
        <span className="se-nav-spacer" />
        <button type="button" onClick={onToggleTray}>{inTray ? '移出对比' : '加入对比'}</button>
      </nav>

      <article id="result" className={`se-anchor se-result ${record.outcome}`}>
        <header className="se-result-h">
          <div>
            <div className="se-kicker">
              <RunnerBadge id={record.runner} inferred={record.runnerInferred} />
              {record.source === 'fixture' && <span className="se-badge fixture">样例</span>}
              {record.scoreSource === 'persisted' && <span className="se-badge muted">报告</span>}
              {record.scoreSource === 'derived' && <span className="se-badge muted">确定性检查</span>}
            </div>
            <h2 className={`se-outcome-xl ${record.outcome}`}>{record.outcome === 'pass' ? '通过' : record.outcome === 'fail' ? '未通过' : '待定'}</h2>
            <p className="se-caption">{caption}</p>
          </div>
          <dl className="se-facts">
            <div><dt>Skill</dt><dd>{record.trace.targetSkill || '—'}</dd></div>
            <div><dt>模型</dt><dd>{record.trace.model}</dd></div>
            <div><dt>Agent</dt><dd>{record.trace.agentName}</dd></div>
            <div><dt>Trace</dt><dd title={record.trace.id}>{shortId(record.trace.id)}</dd></div>
          </dl>
        </header>

        <blockquote className="se-prompt">{record.trace.userPrompt || '（没有用户指令）'}</blockquote>
        <p className="se-summary">{record.summary}</p>

        <div className="se-metrics">
          <div><b>{record.trace.iterations}</b><span>迭代</span></div>
          <div><b>{formatTokens(stats.tokens)}</b><span>Token</span></div>
          <div><b>{stats.durationMs === null ? '—' : formatMs(stats.durationMs)}</b><span>耗时</span></div>
          <div><b>{stats.tool}</b><span>工具</span></div>
          <div><b>{stats.llm}</b><span>LLM</span></div>
        </div>

        {tools.length > 0 ? (
          <ol className="se-stepper">
            {tools.map((span, index) => (
              <li key={`${record.id}-step-${index}`} className={span.error ? 'bad' : span.isTarget ? 'target' : undefined}>
                <b>{span.name}</b>
                <span>{spanDurationLabel(span)}</span>
              </li>
            ))}
          </ol>
        ) : (
          <p className="se-muted">没有工具调用。</p>
        )}

        {record.droppedCalls.length > 0 && (
          <div className="se-callout">
            <strong>有请求没有落到工具 Span</strong>
            <ul>
              {record.droppedCalls.map((call, index) => (
                <li key={`${call.name}-${index}`}>
                  第 {call.iteration} 轮请求了 {call.name}
                  {call.arguments ? ` ${pretty(call.arguments).replace(/\s+/g, ' ').slice(0, 180)}` : ''}
                </li>
              ))}
            </ul>
          </div>
        )}

        {(record.finalResult || record.artifacts.length > 0) && (
          <div className="se-final">
            {record.finalResult && (
              <>
                <h3>最终输出</h3>
                <pre className="se-pre">{record.finalResult}</pre>
              </>
            )}
            {record.artifacts.length > 0 && (
              <>
                <h3>产物</h3>
                <ul className="se-artifacts">
                  {record.artifacts.map((artifact) => (
                    <li key={artifact} title={artifact}>{fileName(artifact)}</li>
                  ))}
                </ul>
              </>
            )}
          </div>
        )}

        {record.file && <p className="se-file">来源 {record.source === 'fixture' ? '界面样例' : `logs/traces/${record.file}`}</p>}
      </article>

      <section id="scores" className="se-anchor se-section">
        <header>
          <h2>评分</h2>
          <p>分数范围 0–1，和 eval.Verdict.score 一致。{scoreLine(record)}</p>
        </header>
        <ScoreList record={record} />
      </section>

      <section id="process" className="se-anchor se-section">
        <header>
          <h2>执行过程</h2>
          <p>每一行是一个 Span。点开可以看模型输出、工具参数和返回。</p>
        </header>
        <ProcessView key={record.id} trace={record.trace} />
      </section>

      <section id="log" className="se-anchor se-section">
        <header>
          <h2>日志</h2>
          <p>按时间排开的步骤摘要，用来快速扫错。</p>
        </header>
        <LogView record={record} />
      </section>

      {record.gaps.length > 0 && (
        <section className="se-gaps">
          <h2>这条记录缺什么</h2>
          <ul>
            {record.gaps.map((gap) => <li key={gap}>{gap}</li>)}
          </ul>
        </section>
      )}
    </div>
  )
}

function scoreLine(record: EvalRecord): string {
  const missing = record.scores.some((score) => score.score === null)
  const scores = record.scores.map((score) => score.score).filter((score): score is number => score !== null)
  if (!scores.length) return '还没有可以相加的分数。'
  const value = scores.reduce((sum, score) => sum + score, 0) / scores.length
  if (missing) return `已出分的均值是 ${formatScore(value)}，未落盘的项目没有算进去。`
  return `均分 ${formatScore(value)}。`
}
