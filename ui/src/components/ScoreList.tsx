import type { EvalRecord } from '../types'
import { clampScore, cx, formatScore } from '../lib/format'
import { Availability, Mark } from './bits'

export default function ScoreList({ record }: { record: EvalRecord }) {
  if (!record.scores.length) return <p className="se-empty-inline">没有评分。</p>
  return (
    <div className="se-scores">
      {record.scores.map((score) => {
        const width = clampScore(score.score)
        const outcome = score.pass === null ? 'unknown' : score.pass ? 'pass' : 'fail'
        return (
          <article key={score.name} className="se-score">
            <div className="se-score-top">
              <div>
                <h3>{score.name}</h3>
                <p>{score.desc}</p>
              </div>
              <div className="se-score-side">
                <Mark outcome={outcome} />
                <b>{formatScore(score.score)}</b>
                <Availability value={score.availability} />
              </div>
            </div>
            <div className={cx('se-bar', outcome)}>
              {width !== null && <span style={{ width: `${width * 100}%` }} />}
            </div>
            <p className="se-score-reason">{score.reason}</p>
          </article>
        )
      })}
    </div>
  )
}
