import { Card, Descriptions, Row, Col } from 'antd';
import { Radar } from '@ant-design/charts';
import { CheckCircleFilled, CloseCircleFilled } from '@ant-design/icons';
import type { Report, Trace, Verdict } from '../types';
import SpanTimeline from './SpanTimeline';

interface ReportDetailProps {
  report: Report;
  trace: Trace;
}

function ScoreBar({ score }: { score: number }) {
  const pct = score * 100;
  const cls = pct >= 70 ? 'high' : pct >= 40 ? 'mid' : 'low';
  return (
    <div className="score-bar-wrap">
      <div className="score-bar-outer">
        <div className={`score-bar-inner ${cls}`} style={{ width: `${pct}%` }} />
      </div>
    </div>
  );
}

export default function ReportDetail({ report, trace }: ReportDetailProps) {
  const radarData = report.scores.map((s: Verdict) => ({
    name: s.info.name,
    value: s.score * 10,
  }));

  const radarConfig = {
    data: radarData,
    xField: 'name',
    yField: 'value',
    area: { style: { fillOpacity: 0.15, fill: '#22d3ee' } },
    scale: { y: { domainMin: 0, domainMax: 10 } },
    axis: {
      x: { label: { style: { fontSize: 10, fill: '#a1a1aa', fontFamily: 'Outfit' } }, grid: true, gridStrokeOpacity: 0.06 },
      y: { label: false, gridStrokeOpacity: 0.06 },
    },
    line: { style: { stroke: '#22d3ee', lineWidth: 2 } },
    point: { style: { fill: '#22d3ee', r: 3 } },
    height: 280,
    theme: { background: 'transparent' },
  };

  return (
    <div className="gap-20 stagger">
      {/* Stat Grid */}
      <div className="stat-grid">
        <div className="stat-item cyan">
          <div className="stat-label">模型</div>
          <div className="stat-value" style={{ fontSize: 15 }}>{report.model}</div>
        </div>
        <div className="stat-item cyan">
          <div className="stat-label">目标 Skill</div>
          <div className="stat-value" style={{ fontSize: 15 }}>{report.targetSkill}</div>
        </div>
        <div className="stat-item amber">
          <div className="stat-label">迭代轮次</div>
          <div className="stat-value">{report.iterations}<span className="stat-suffix">轮</span></div>
        </div>
        <div className="stat-item violet">
          <div className="stat-label">Token 消耗</div>
          <div className="stat-value">{(report.totalTokens / 1000).toFixed(1)}<span className="stat-suffix">K</span></div>
        </div>
        <div className="stat-item green">
          <div className="stat-label">执行耗时</div>
          <div className="stat-value">{(report.duration / 1000).toFixed(1)}<span className="stat-suffix">s</span></div>
        </div>
        <div className="stat-item" style={{}}>
          <div className="stat-label">结果</div>
          <div className={`stat-value ${report.pass ? 'green' : 'rose'}`}>
            {report.pass ? 'PASS' : 'FAIL'}
          </div>
        </div>
      </div>

      {/* Info */}
      <Card className="section-card" title="基本信息" size="small">
        <Descriptions column={2} size="small" className="dark-descriptions">
          <Descriptions.Item label="Trace ID"><span className="mono-val">{report.traceId.slice(0, 12)}...</span></Descriptions.Item>
          <Descriptions.Item label="Agent">{report.agentName}</Descriptions.Item>
          <Descriptions.Item label="用户指令" span={2}>{report.userPrompt}</Descriptions.Item>
        </Descriptions>
      </Card>

      {/* Scores */}
      <Row gutter={16}>
        <Col span={9}>
          <Card className="section-card" title="评分雷达" size="small">
            <div className="radar-wrapper">
              <Radar {...radarConfig} />
            </div>
          </Card>
        </Col>
        <Col span={15}>
          <Card className="section-card" title="评分详情" size="small">
            <div className="score-list">
              {report.scores.map((s: Verdict, i: number) => (
                <div className="score-row" key={s.info.name} style={{ animationDelay: `${i * 60}ms` }}>
                  <div className={`score-status ${s.pass ? 'pass' : 'fail'}`}>
                    {s.pass ? <CheckCircleFilled /> : <CloseCircleFilled />}
                  </div>
                  <div className="score-info">
                    <div className="score-name">{s.info.name}</div>
                    <div className="score-desc">{s.info.desc}</div>
                  </div>
                  <ScoreBar score={s.score} />
                  <div className="score-number">{(s.score * 10).toFixed(1)}</div>
                  <div className="score-reason">{s.reason}</div>
                </div>
              ))}
            </div>
          </Card>
        </Col>
      </Row>

      {/* Trace */}
      <Card className="section-card" title="执行链路" size="small">
        <SpanTimeline spans={trace.spans} />
      </Card>
    </div>
  );
}
