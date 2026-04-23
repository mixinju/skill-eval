import { Card, Descriptions, Row, Col } from 'antd';
import { Column } from '@ant-design/charts';
import { CheckCircleFilled, CloseCircleFilled, BulbOutlined } from '@ant-design/icons';
import type { CompareReport, Verdict, Span } from '../types';
import SpanTimeline from './SpanTimeline';

interface CompareViewProps {
  compareReport: CompareReport;
}

function spanStats(spans: Span[]) {
  let llmCalls = 0, toolCalls = 0, totalTokens = 0, totalDuration = 0;
  for (const s of spans) {
    if (s.kind === 'llm_call') llmCalls++;
    if (s.kind === 'tool_call') toolCalls++;
    totalTokens += s.totalTokens || 0;
    totalDuration += s.duration || 0;
  }
  return { llmCalls, toolCalls, totalTokens, totalDuration };
}

export default function CompareView({ compareReport }: CompareViewProps) {
  const { traceA, traceB, reportA, reportB, scores, conclusion } = compareReport;
  const statsA = spanStats(traceA.spans);
  const statsB = spanStats(traceB.spans);

  const chartData = [
    { metric: 'LLM 调用', model: reportA.model, value: statsA.llmCalls },
    { metric: 'LLM 调用', model: reportB.model, value: statsB.llmCalls },
    { metric: 'Tool 调用', model: reportA.model, value: statsA.toolCalls },
    { metric: 'Tool 调用', model: reportB.model, value: statsB.toolCalls },
    { metric: '迭代轮次', model: reportA.model, value: reportA.iterations },
    { metric: '迭代轮次', model: reportB.model, value: reportB.iterations },
  ];

  const chartConfig = {
    data: chartData,
    xField: 'metric',
    yField: 'value',
    colorField: 'model',
    group: true,
    height: 260,
    style: { inset: 5, radiusTopLeft: 4, radiusTopRight: 4 },
    scale: { color: { range: ['#22d3ee', '#a78bfa'] } },
    axis: {
      x: { label: { style: { fill: '#a1a1aa', fontSize: 11, fontFamily: 'Outfit' } }, line: { style: { stroke: '#3f3f46' } } },
      y: { label: { style: { fill: '#71717a', fontSize: 10, fontFamily: 'Fira Code' } }, grid: { line: { style: { stroke: '#27272a' } } } },
    },
    legend: { color: { itemLabelFill: '#a1a1aa', itemLabelFontFamily: 'Fira Code', itemLabelFontSize: 10 } },
    theme: { background: 'transparent' },
  };

  const tokenDiff = reportA.totalTokens - reportB.totalTokens;
  const durationDiff = reportA.duration - reportB.duration;

  return (
    <div className="gap-20 stagger">
      {/* A vs B Info */}
      <Row gutter={12}>
        <Col span={12}>
          <Card className="section-card compare-card a" size="small"
            title={<><span className="compare-label a">A</span> <span style={{ marginLeft: 8 }}>{reportA.model}</span></>}>
            <Descriptions column={2} size="small" className="dark-descriptions">
              <Descriptions.Item label="Skill"><span className="mono-val">{reportA.targetSkill}</span></Descriptions.Item>
              <Descriptions.Item label="迭代"><span className="mono-val">{reportA.iterations}轮</span></Descriptions.Item>
              <Descriptions.Item label="Token"><span className="mono-val">{reportA.totalTokens.toLocaleString()}</span></Descriptions.Item>
              <Descriptions.Item label="耗时"><span className="mono-val">{(reportA.duration / 1000).toFixed(1)}s</span></Descriptions.Item>
              <Descriptions.Item label="LLM"><span className="mono-val">{statsA.llmCalls}次</span></Descriptions.Item>
              <Descriptions.Item label="Tool"><span className="mono-val">{statsA.toolCalls}次</span></Descriptions.Item>
            </Descriptions>
          </Card>
        </Col>
        <Col span={12}>
          <Card className="section-card compare-card b" size="small"
            title={<><span className="compare-label b">B</span> <span style={{ marginLeft: 8 }}>{reportB.model}</span></>}>
            <Descriptions column={2} size="small" className="dark-descriptions">
              <Descriptions.Item label="Skill"><span className="mono-val">{reportB.targetSkill}</span></Descriptions.Item>
              <Descriptions.Item label="迭代"><span className="mono-val">{reportB.iterations}轮</span></Descriptions.Item>
              <Descriptions.Item label="Token"><span className="mono-val">{reportB.totalTokens.toLocaleString()}</span></Descriptions.Item>
              <Descriptions.Item label="耗时"><span className="mono-val">{(reportB.duration / 1000).toFixed(1)}s</span></Descriptions.Item>
              <Descriptions.Item label="LLM"><span className="mono-val">{statsB.llmCalls}次</span></Descriptions.Item>
              <Descriptions.Item label="Tool"><span className="mono-val">{statsB.toolCalls}次</span></Descriptions.Item>
            </Descriptions>
          </Card>
        </Col>
      </Row>

      {/* Diff stats */}
      <div className="stat-grid">
        <div className={`stat-item ${tokenDiff > 0 ? 'rose' : 'green'}`}>
          <div className="stat-label">Token 差异</div>
          <div className="stat-value">
            {tokenDiff > 0 ? '+' : ''}{tokenDiff.toLocaleString()}
          </div>
        </div>
        <div className={`stat-item ${durationDiff > 0 ? 'rose' : 'green'}`}>
          <div className="stat-label">耗时差异</div>
          <div className="stat-value">
            {durationDiff > 0 ? '+' : ''}{(durationDiff / 1000).toFixed(1)}<span className="stat-suffix">s</span>
          </div>
        </div>
        <div className="stat-item cyan">
          <div className="stat-label">A 迭代</div>
          <div className="stat-value">{reportA.iterations}<span className="stat-suffix">轮</span></div>
        </div>
        <div className="stat-item violet">
          <div className="stat-label">B 迭代</div>
          <div className="stat-value">{reportB.iterations}<span className="stat-suffix">轮</span></div>
        </div>
      </div>

      {/* Chart + Scores */}
      <Row gutter={12}>
        <Col span={10}>
          <Card className="section-card" title="调用统计" size="small">
            <Column {...chartConfig} />
          </Card>
        </Col>
        <Col span={14}>
          <Card className="section-card" title="对比评分" size="small">
            <div className="score-list">
              {scores.map((s: Verdict, i: number) => (
                <div className="score-row" key={s.info.name} style={{ animationDelay: `${i * 60}ms` }}>
                  <div className={`score-status ${s.pass ? 'pass' : 'fail'}`}>
                    {s.pass ? <CheckCircleFilled /> : <CloseCircleFilled />}
                  </div>
                  <div className="score-info">
                    <div className="score-name">{s.info.name}</div>
                    <div className="score-desc">{s.info.desc}</div>
                  </div>
                  <div className="score-bar-wrap">
                    <div className="score-bar-outer">
                      <div
                        className={`score-bar-inner ${s.score >= 0.7 ? 'high' : s.score >= 0.4 ? 'mid' : 'low'}`}
                        style={{ width: `${s.score * 100}%` }}
                      />
                    </div>
                  </div>
                  <div className="score-number">{(s.score * 10).toFixed(1)}</div>
                </div>
              ))}
            </div>
          </Card>
        </Col>
      </Row>

      {/* Conclusion */}
      {conclusion && (
        <div className="conclusion-block">
          <div className="conclusion-title"><BulbOutlined /> AI 对比结论</div>
          <div className="conclusion-text">{conclusion}</div>
        </div>
      )}

      {/* Parallel Timelines */}
      <Row gutter={12}>
        <Col span={12}>
          <Card className="section-card compare-card a" title={`执行链路 — ${reportA.model}`} size="small">
            <div style={{ maxHeight: 500, overflow: 'auto' }}>
              <SpanTimeline spans={traceA.spans} />
            </div>
          </Card>
        </Col>
        <Col span={12}>
          <Card className="section-card compare-card b" title={`执行链路 — ${reportB.model}`} size="small">
            <div style={{ maxHeight: 500, overflow: 'auto' }}>
              <SpanTimeline spans={traceB.spans} />
            </div>
          </Card>
        </Col>
      </Row>
    </div>
  );
}
