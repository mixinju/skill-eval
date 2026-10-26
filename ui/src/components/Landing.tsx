import type { EvalRecord } from '../types'
import { formatWhen } from '../lib/format'
import { RUNNERS, runnerMeta } from '../lib/runners'
import { Mark, RunnerBadge } from './bits'

export default function Landing({
  records,
  onOpen,
  onRunner,
}: {
  records: EvalRecord[]
  onOpen: (id: string) => void
  onRunner: (id: EvalRecord['runner']) => void
}) {
  const real = records.filter((record) => record.source === 'trace-file')
  return (
    <div className="se-page se-landing">
      <p className="se-kicker-text">Skill Eval</p>
      <h2>看清一次 skill 评测的结果和过程</h2>
      <p className="se-lead">
        左边是已经落下的运行。点进去能看到通过与否、分数，以及模型调用和工具调用是怎么排开的。
      </p>
      <div className="se-runner-cards">
        {RUNNERS.map((runner) => {
          const count = records.filter((record) => record.runner === runner.id).length
          const realCount = records.filter((record) => record.runner === runner.id && record.source === 'trace-file').length
          return (
            <button key={runner.id} type="button" className="se-runner-card" onClick={() => onRunner(runner.id)}>
              <span className="se-kicker">
                <RunnerBadge id={runner.id} />
                <span className={runner.implemented ? 'se-badge kernel' : 'se-badge muted'}>
                  {runner.implemented ? '可执行' : '未接入'}
                </span>
              </span>
              <strong>{realCount} 条真实 · {count} 条可见</strong>
              <p>{runnerMeta(runner.id).summary}</p>
            </button>
          )
        })}
      </div>
      <section className="se-section">
        <header>
          <h2>最近的真实运行</h2>
          <p>来自 logs/traces。历史文件没有 runner，都归在自研内核。</p>
        </header>
        {real.length === 0 ? (
          <p className="se-empty-inline">还没有真实 Trace。</p>
        ) : (
          <ul className="se-recent">
            {real.map((record) => (
              <li key={record.id}>
                <button type="button" onClick={() => onOpen(record.id)}>
                  <Mark outcome={record.outcome} />
                  <span>
                    <b>{record.trace.targetSkill || '未指定 skill'}</b>
                    <small>{record.trace.userPrompt}</small>
                  </span>
                  <time>{formatWhen(record.createdAt)}</time>
                </button>
              </li>
            ))}
          </ul>
        )}
      </section>
    </div>
  )
}
