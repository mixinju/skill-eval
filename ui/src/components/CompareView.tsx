import { Card, Descriptions, Table, Tag, Typography, Row, Col, Alert, Statistic } from 'antd';
import { Column } from '@ant-design/charts';
import { CheckCircleFilled, CloseCircleFilled, ArrowUpOutlined, ArrowDownOutlined } from '@ant-design/icons';
import type { CompareReport, Verdict, Span } from '../types';
import SpanTimeline from './SpanTimeline';

const { Text, Paragraph } = Typography;

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

const scoreColumns = [
  {
    title: '评分项',
    dataIndex: ['info', 'name'],
    key: 'name',
    width: 140,
  },
  {
    title: '描述',
    dataIndex: ['info', 'desc'],
    key: 'desc',
    width: 180,
  },
  {
    title: '分数',
    dataIndex: 'score',
    key: 'score',
    width: 80,
    render: (v: number) => <Text strong>{(v * 10).toFixed(1)}</Text>,
  },
  {
    title: '状态',
    dataIndex: 'pass',
    key: 'pass',
    width: 80,
    render: (pass: boolean) =>
      pass
        ? <Tag icon={<CheckCircleFilled />} color="success">PASS</Tag>
        : <Tag icon={<CloseCircleFilled />} color="error">FAIL</Tag>,
  },
  {
    title: '理由',
    dataIndex: 'reason',
    key: 'reason',
    ellipsis: true,
  },
];

export default function CompareView({ compareReport }: CompareViewProps) {
  const { traceA, traceB, reportA, reportB, scores, conclusion } = compareReport;
  const statsA = spanStats(traceA.spans);
  const statsB = spanStats(traceB.spans);

  const chartData = [
    { metric: 'LLM 调用', model: reportA.model, value: statsA.llmCalls },
    { metric: 'LLM 调用', model: reportB.model, value: statsB.llmCalls },
    { metric: 'Tool 调用', model: reportA.model, value: statsA.toolCalls },
    { metric: 'Tool 调用', model: reportB.model, value: statsB.toolCalls },
    { metric: '迭代次数', model: reportA.model, value: reportA.iterations },
    { metric: '迭代次数', model: reportB.model, value: reportB.iterations },
  ];

  const chartConfig = {
    data: chartData,
    xField: 'metric',
    yField: 'value',
    colorField: 'model',
    group: true,
    height: 280,
    style: { inset: 5 },
  };

  const tokenDiff = reportA.totalTokens - reportB.totalTokens;
  const durationDiff = reportA.duration - reportB.duration;

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
      <Row gutter={16}>
        <Col span={12}>
          <Card
            title={<><Tag color="blue">A</Tag> {reportA.model}</>}
            size="small"
          >
            <Descriptions column={2} size="small">
              <Descriptions.Item label="目标 Skill">
                <Tag color="blue">{reportA.targetSkill}</Tag>
              </Descriptions.Item>
              <Descriptions.Item label="迭代次数">{reportA.iterations}</Descriptions.Item>
              <Descriptions.Item label="Token 消耗">
                {reportA.totalTokens.toLocaleString()}
              </Descriptions.Item>
              <Descriptions.Item label="耗时">{(reportA.duration / 1000).toFixed(1)}s</Descriptions.Item>
              <Descriptions.Item label="LLM 调用">{statsA.llmCalls}</Descriptions.Item>
              <Descriptions.Item label="Tool 调用">{statsA.toolCalls}</Descriptions.Item>
            </Descriptions>
          </Card>
        </Col>
        <Col span={12}>
          <Card
            title={<><Tag color="purple">B</Tag> {reportB.model}</>}
            size="small"
          >
            <Descriptions column={2} size="small">
              <Descriptions.Item label="目标 Skill">
                <Tag color="purple">{reportB.targetSkill}</Tag>
              </Descriptions.Item>
              <Descriptions.Item label="迭代次数">{reportB.iterations}</Descriptions.Item>
              <Descriptions.Item label="Token 消耗">
                {reportB.totalTokens.toLocaleString()}
              </Descriptions.Item>
              <Descriptions.Item label="耗时">{(reportB.duration / 1000).toFixed(1)}s</Descriptions.Item>
              <Descriptions.Item label="LLM 调用">{statsB.llmCalls}</Descriptions.Item>
              <Descriptions.Item label="Tool 调用">{statsB.toolCalls}</Descriptions.Item>
            </Descriptions>
          </Card>
        </Col>
      </Row>

      <Card title="差异概览" size="small">
        <Row gutter={16}>
          <Col span={6}>
            <Statistic
              title="Token 差异"
              value={Math.abs(tokenDiff)}
              prefix={tokenDiff > 0 ? <ArrowUpOutlined /> : <ArrowDownOutlined />}
              suffix={tokenDiff > 0 ? 'A 更多' : 'B 更多'}
              valueStyle={{ color: tokenDiff > 0 ? '#cf1322' : '#3f8600', fontSize: 16 }}
            />
          </Col>
          <Col span={6}>
            <Statistic
              title="耗时差异"
              value={Math.abs(durationDiff / 1000).toFixed(1)}
              prefix={durationDiff > 0 ? <ArrowUpOutlined /> : <ArrowDownOutlined />}
              suffix={`s ${durationDiff > 0 ? 'A 更慢' : 'B 更慢'}`}
              valueStyle={{ color: durationDiff > 0 ? '#cf1322' : '#3f8600', fontSize: 16 }}
            />
          </Col>
          <Col span={6}>
            <Statistic title="A 迭代" value={reportA.iterations} suffix="轮" valueStyle={{ fontSize: 16 }} />
          </Col>
          <Col span={6}>
            <Statistic title="B 迭代" value={reportB.iterations} suffix="轮" valueStyle={{ fontSize: 16 }} />
          </Col>
        </Row>
      </Card>

      <Row gutter={16}>
        <Col span={10}>
          <Card title="调用统计对比" size="small">
            <Column {...chartConfig} />
          </Card>
        </Col>
        <Col span={14}>
          <Card title="对比评分" size="small">
            <Table
              dataSource={scores}
              columns={scoreColumns}
              pagination={false}
              size="small"
              rowKey={(r: Verdict) => r.info.name}
            />
          </Card>
        </Col>
      </Row>

      {conclusion && (
        <Alert
          message="AI 对比结论"
          description={<Paragraph style={{ margin: 0 }}>{conclusion}</Paragraph>}
          type="info"
          showIcon
        />
      )}

      <Row gutter={16}>
        <Col span={12}>
          <Card title={`执行链路 — ${reportA.model}`} size="small">
            <div style={{ maxHeight: 600, overflow: 'auto' }}>
              <SpanTimeline spans={traceA.spans} />
            </div>
          </Card>
        </Col>
        <Col span={12}>
          <Card title={`执行链路 — ${reportB.model}`} size="small">
            <div style={{ maxHeight: 600, overflow: 'auto' }}>
              <SpanTimeline spans={traceB.spans} />
            </div>
          </Card>
        </Col>
      </Row>
    </div>
  );
}
