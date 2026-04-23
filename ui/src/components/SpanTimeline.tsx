import { useState } from 'react';
import { Timeline, Tag, Card, Typography, Space, Collapse } from 'antd';
import {
  RobotOutlined,
  ToolOutlined,
  CompressOutlined,
  CheckCircleOutlined,
  CloseCircleOutlined,
  AimOutlined,
} from '@ant-design/icons';
import type { Span } from '../types';

const { Text, Paragraph } = Typography;

interface SpanTimelineProps {
  spans: Span[];
  title?: string;
}

const kindConfig: Record<string, { color: string; icon: React.ReactNode; label: string }> = {
  llm_call: { color: '#1677ff', icon: <RobotOutlined />, label: 'LLM' },
  tool_call: { color: '#52c41a', icon: <ToolOutlined />, label: 'Tool' },
  llm_compress: { color: '#faad14', icon: <CompressOutlined />, label: '压缩' },
};

function SpanCard({ span }: { span: Span }) {
  const [expanded, setExpanded] = useState(false);
  const cfg = kindConfig[span.kind] || kindConfig.llm_call;

  const items = [];
  if (span.kind === 'llm_call' || span.kind === 'llm_compress') {
    if (span.llmInput) items.push({ key: 'input', label: 'LLM 输入', children: <Paragraph copyable style={{ whiteSpace: 'pre-wrap', maxHeight: 200, overflow: 'auto', margin: 0 }}>{span.llmInput}</Paragraph> });
    if (span.llmOutput) items.push({ key: 'output', label: 'LLM 输出', children: <Paragraph copyable style={{ whiteSpace: 'pre-wrap', maxHeight: 200, overflow: 'auto', margin: 0 }}>{span.llmOutput}</Paragraph> });
  }
  if (span.kind === 'tool_call') {
    if (span.toolInput) items.push({ key: 'input', label: '工具输入', children: <Paragraph copyable style={{ whiteSpace: 'pre-wrap', maxHeight: 200, overflow: 'auto', margin: 0 }}>{span.toolInput}</Paragraph> });
    if (span.toolOutput) items.push({ key: 'output', label: '工具输出', children: <Paragraph copyable style={{ whiteSpace: 'pre-wrap', maxHeight: 200, overflow: 'auto', margin: 0 }}>{span.toolOutput}</Paragraph> });
  }

  return (
    <Card
      size="small"
      style={{ cursor: items.length > 0 ? 'pointer' : 'default' }}
      onClick={() => items.length > 0 && setExpanded(!expanded)}
    >
      <Space wrap>
        <Tag color={cfg.color} icon={cfg.icon}>{cfg.label}</Tag>
        <Text strong>{span.name}</Text>
        {span.duration != null && <Text type="secondary">{span.duration}ms</Text>}
        {span.totalTokens != null && <Tag>{span.totalTokens} tokens</Tag>}
        {span.isTarget && <Tag color="gold" icon={<AimOutlined />}>目标Skill</Tag>}
        {span.error && <Tag color="red" icon={<CloseCircleOutlined />}>错误</Tag>}
        {!span.error && span.kind === 'tool_call' && <Tag color="green" icon={<CheckCircleOutlined />}>成功</Tag>}
      </Space>
      {expanded && items.length > 0 && (
        <div onClick={(e) => e.stopPropagation()} style={{ marginTop: 8 }}>
          <Collapse ghost size="small" items={items} />
        </div>
      )}
    </Card>
  );
}

export default function SpanTimeline({ spans, title }: SpanTimelineProps) {
  const grouped = new Map<number, Span[]>();
  for (const span of spans) {
    const list = grouped.get(span.iteration) || [];
    list.push(span);
    grouped.set(span.iteration, list);
  }

  const iterations = Array.from(grouped.keys()).sort((a, b) => a - b);

  return (
    <div>
      {title && <Typography.Title level={5} style={{ marginBottom: 16 }}>{title}</Typography.Title>}
      {iterations.map((iter) => (
        <div key={iter} style={{ marginBottom: 16 }}>
          <Text type="secondary" style={{ fontSize: 12, marginBottom: 4, display: 'block' }}>
            第 {iter} 轮迭代
          </Text>
          <Timeline
            items={grouped.get(iter)!.map((span) => ({
              color: span.error ? 'red' : kindConfig[span.kind]?.color || 'blue',
              children: <SpanCard span={span} />,
            }))}
          />
        </div>
      ))}
    </div>
  );
}
