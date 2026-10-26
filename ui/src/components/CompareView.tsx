import type { EvalRecord } from '../types'
import { formatMs, formatScore, formatSignedMs, formatTokens } from '../lib/format'
import { traceStats } from '../lib/model'
import { runnerMeta } from '../lib/runners'
import { Mark, RunnerBadge } from './bits'
import ProcessView from './ProcessView'

function deltaNumber(a: number, b: number): string {
  const diff = a - b
  if (diff === 0) return '0'
  const sign = diff > 0 ? '+' : '−'
  const abs = Math.abs(diff)
  return `${sign}${Number.isInteger(abs) ? abs : abs.toFixed(1)}`
}

export default function CompareView({ a, b, onClose }: { a: EvalRecord; b: EvalRecord; onClose: () => void }) {
  const left = traceStats(a.trace)
  const right = traceStats(b.trace)
  const rows = [
    { label: '结果', left: a.outcome, right: b.outcome, text: true as const },
    { label: '迭代', left: a.trace.iterations, right: b.trace.iterations },
    { label: 'Token', left: left.tokens, right: right.tokens, format: formatTokens },
    { label: '耗时', left: left.durationMs ?? 0, right: right.durationMs ?? 0, missing: left.durationMs === null || right.durationMs === null, format: formatMs, signed: formatSignedMs },
    { label: '工具', left: left.tool, right: right.tool },
    { label: 'LLM', left: left.llm, right: right.llm },
    { label: '错误', left: left.errors, right: right.errors },
  ]

  return (
    <div className="se-page">
      <nav className="se-nav">
        <strong>对比</strong>
        <span className="se-nav-spacer" />
        <button type="button" onClick={onClose}>返回 {a.trace.targetSkill || '详情'}</button>
      </nav>
      <p className="se-section-note">差值是左边减去右边。后端没有把对比结论落盘，所以这里只并列 Trace 上能算出来的数字，不写模型评语。</p>

      <div className="se-compare-head">
        {[a, b].map((record) => (
          <article key={record.id} className={`se-result ${record.outcome}`}>
            <div className="se-kicker">
              <RunnerBadge id={record.runner} inferred={record.runnerInferred} short />
              {record.source === 'fixture' && <span className="se-badge fixture">样例</span>}
            </div>
            <h2 className={`se-outcome-xl ${record.outcome}`}>{record.outcome === 'pass' ? '通过' : record.outcome === 'fail' ? '未通过' : '待定'}</h2>
            <p className="se-caption">{record.trace.targetSkill || '未指定 skill'} · {record.trace.model} · {runnerMeta(record.runner).label}</p>
            <p className="se-summary">{record.trace.userPrompt}</p>
          </article>
        ))}
      </div>

      <div className="se-table-wrap">
        <table className="se-table">
          <thead>
            <tr>
              <th>指标</th>
              <th>{a.trace.model}</th>
              <th>{b.trace.model}</th>
              <th>差值</th>
            </tr>
          </thead>
          <tbody>
            {rows.map((row) => (
              <tr key={row.label}>
                <th>{row.label}</th>
                {row.text ? (
                  <>
                    <td><Mark outcome={a.outcome} /></td>
                    <td><Mark outcome={b.outcome} /></td>
                    <td>—</td>
                  </>
                ) : (
                  <>
                    <td>{row.missing ? '—' : row.format ? row.format(row.left) : row.left}</td>
                    <td>{row.missing ? '—' : row.format ? row.format(row.right) : row.right}</td>
                    <td>
                      {row.missing
                        ? '—'
                        : row.signed
                          ? row.signed(row.left - row.right)
                          : row.format && row.label === 'Token'
                            ? deltaNumber(row.left, row.right)
                            : deltaNumber(row.left, row.right)}
                    </td>
                  </>
                )}
              </tr>
            ))}
            <ScoreRows a={a} b={b} />
          </tbody>
        </table>
      </div>

      <div className="se-compare-grid">
        {[a, b].map((record) => (
          <section key={record.id} className="se-section">
            <header>
              <h2>{runnerMeta(record.runner).label}</h2>
              <p>{record.summary}</p>
            </header>
            <ProcessView trace={record.trace} />
          </section>
        ))}
      </div>
    </div>
  )
}

function ScoreRows({ a, b }: { a: EvalRecord; b: EvalRecord }) {
  const names = [...new Set([...a.scores.map((score) => score.name), ...b.scores.map((score) => score.name)])]
  return (
    <>
      {names.map((name) => {
        const left = a.scores.find((score) => score.name === name)
        const right = b.scores.find((score) => score.name === name)
        return (
          <tr key={name}>
            <th>{name}</th>
            <td>{left ? formatScore(left.score) : '—'}</td>
            <td>{right ? formatScore(right.score) : '—'}</td>
            <td>
              {left?.score !== null && left?.score !== undefined && right?.score !== null && right?.score !== undefined
                ? deltaNumber(left.score, right.score)
                : '—'}
            </td>
          </tr>
        )
      })}
    </>
  )
}
