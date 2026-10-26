import { useState } from 'react'
import type { Span, Trace } from '../types'
import { barPlacement, groupIterations, isNestedSpan, parseAssistant, parseMessages, timelineScale } from '../lib/model'
import { cx, kindLabel, pretty, spanDurationLabel } from '../lib/format'
import { Reveal } from './bits'

function Messages({ raw }: { raw?: string }) {
  const [systemOpen, setSystemOpen] = useState(false)
  const messages = parseMessages(raw)
  if (!messages) return <Reveal text={raw || ''} limit={500} />
  return (
    <div className="se-msgs">
      {messages.map((message, index) => (
        <div key={`${message.role}-${index}`} className="se-msg">
          <div className="se-msg-role">{message.role}</div>
          {message.role === 'system' && !systemOpen ? (
            <button type="button" className="se-textbtn" onClick={() => setSystemOpen(true)}>
              系统提示 {message.content.length.toLocaleString('zh-CN')} 字，默认折叠
            </button>
          ) : (
            <Reveal text={message.content} limit={message.role === 'system' ? 1200 : 800} />
          )}
        </div>
      ))}
    </div>
  )
}

function Assistant({ raw }: { raw?: string }) {
  const view = parseAssistant(raw)
  if (!view) return <p className="se-muted">（无输出）</p>
  const hasStructure = Boolean(view.content) || view.toolCalls.length > 0 || Boolean(view.reasoning)
  if (!hasStructure) return <Reveal text={view.raw} />
  return (
    <div className="se-assistant">
      {view.content && <Reveal text={view.content} />}
      {view.toolCalls.length > 0 && (
        <ul className="se-calls">
          {view.toolCalls.map((call, index) => (
            <li key={`${call.name}-${index}`}>
              <b>{call.name}</b>
              <Reveal text={pretty(call.arguments)} limit={500} />
            </li>
          ))}
        </ul>
      )}
      {view.reasoning && (
        <details>
          <summary>思考</summary>
          <Reveal text={view.reasoning} limit={900} />
        </details>
      )}
    </div>
  )
}

function SpanBody({ span }: { span: Span }) {
  if (span.kind === 'tool_call') {
    return (
      <div className="se-io">
        <section>
          <h4>输入</h4>
          <div className="se-block">
            <Reveal text={pretty(span.toolInput) || '（无输入）'} limit={900} />
          </div>
        </section>
        <section>
          <h4>输出</h4>
          <div className="se-block">
            <Reveal text={span.toolOutput || '（无输出）'} limit={900} />
          </div>
        </section>
        {span.error && <p className="se-error">{span.error}</p>}
      </div>
    )
  }
  return (
    <div className="se-io">
      <section>
        <h4>输出</h4>
        <div className="se-block">
          <Assistant raw={span.llmOutput} />
        </div>
      </section>
      <section>
        <h4>输入</h4>
        <div className="se-block">
          <Messages raw={span.llmInput} />
        </div>
      </section>
      {span.error && <p className="se-error">{span.error}</p>}
    </div>
  )
}

export default function ProcessView({ trace }: { trace: Trace }) {
  const firstTool = trace.spans.findIndex((span) => span.kind === 'tool_call')
  const [open, setOpen] = useState(firstTool >= 0 ? firstTool : 0)
  const scale = timelineScale(trace)
  const groups = groupIterations(trace)

  if (!trace.spans.length) {
    return <p className="se-empty-inline">这条 Trace 没有 Span，过程还原不了。</p>
  }

  const focus = (index: number) => {
    setOpen(index)
    document.getElementById(`span-${trace.id}-${index}`)?.scrollIntoView({ behavior: 'smooth', block: 'nearest' })
  }

  return (
    <div>
      <p className="se-section-note">耗时用 Span 的起止时间。旧文件里的 duration 有时是纳秒，不拿来画时间条。</p>
      <div className="se-legend">
        <span><i className="llm_call" /> LLM</span>
        <span><i className="tool_call" /> 工具</span>
        <span><i className="llm_compress" /> 压缩</span>
        <span><i className="error" /> 错误</span>
      </div>
      <div className="se-wf">
        {groups.map((group) => (
          <div key={group.iteration} className="se-wf-group">
            <div className="se-round">第 {group.iteration} 轮</div>
            {group.spans.map(({ span, index }) => (
              <button
                key={`${trace.id}-wf-${index}`}
                type="button"
                className={cx('se-wf-row', open === index && 'open')}
                onClick={() => focus(index)}
              >
                <span className="se-wf-name">
                  <em>{kindLabel(span.kind)}</em>
                  {span.name}
                </span>
                <span className="se-wf-track">
                  <i
                    className={cx('se-wf-bar', span.error ? 'error' : span.kind)}
                    style={barPlacement(span, scale)}
                  />
                </span>
                <span className="se-wf-dur">{spanDurationLabel(span)}</span>
              </button>
            ))}
          </div>
        ))}
      </div>

      <div className="se-spans">
        {groups.map((group) => (
          <section key={`round-${group.iteration}`}>
            <h3 className="se-round">第 {group.iteration} 轮</h3>
            {group.spans.map(({ span, index }) => {
              const meta = [
                span.inputMessages ? `${span.inputMessages} 条输入` : '',
                span.totalTokens ? `${span.totalTokens} tok` : '',
                span.finishReason || '',
                span.isTarget ? '目标 skill' : '',
              ].filter(Boolean)
              return (
                <article
                  key={`${trace.id}-span-${index}`}
                  id={`span-${trace.id}-${index}`}
                  className={cx('se-span', isNestedSpan(span) && 'nested', span.error && 'bad', open === index && 'open')}
                >
                  <button type="button" className="se-span-h" aria-expanded={open === index} onClick={() => setOpen(open === index ? -1 : index)}>
                    <span className={cx('se-kind', span.error ? 'error' : span.kind)}>{kindLabel(span.kind)}</span>
                    <span>
                      <b>{span.name}</b>
                      {meta.length > 0 && <small>{meta.join(' · ')}</small>}
                    </span>
                    <span className="se-wf-dur">{spanDurationLabel(span)}</span>
                  </button>
                  {open === index && <SpanBody span={span} />}
                </article>
              )
            })}
          </section>
        ))}
      </div>
    </div>
  )
}
