import { useState } from 'react';
import { Timeline, Card, Collapse } from 'antd';
import {
  RobotOutlined,
  ToolOutlined,
  CompressOutlined,
  CheckCircleOutlined,
  CloseCircleOutlined,
  AimOutlined,
  DownOutlined,
} from '@ant-design/icons';
import type { Span } from '../types';

interface SpanTimelineProps {
  spans: Span[];
}

const kindMeta: Record<string, { cls: string; icon: React.ReactNode; label: string }> = {
  llm_call: { cls: 'llm', icon: <RobotOutlined />, label: 'LLM' },
  tool_call: { cls: 'tool', icon: <ToolOutlined />, label: 'TOOL' },
  llm_compress: { cls: 'compress', icon: <CompressOutlined />, label: 'COMPRESS' },
};

const kindColor: Record<string, string> = {
  llm_call: '#22d3ee',
  tool_call: '#4ade80',
  llm_compress: '#fbbf24',
};

function SpanCard({ span }: { span: Span }) {
  const [expanded, setExpanded] = useState(false);
  const meta = kindMeta[span.kind] || kindMeta.llm_call;

  const details = [];
  if (span.kind === 'llm_call' || span.kind === 'llm_compress') {
    if (span.llmInput) details.push({ key: 'in', label: 'Input', children: <pre style={{ whiteSpace: 'pre-wrap', margin: 0, fontSize: 11, lineHeight: 1.5, color: '#d4d4d8' }}>{span.llmInput}</pre> });
    if (span.llmOutput) details.push({ key: 'out', label: 'Output', children: <pre style={{ whiteSpace: 'pre-wrap', margin: 0, fontSize: 11, lineHeight: 1.5, color: '#d4d4d8' }}>{span.llmOutput}</pre> });
  }
  if (span.kind === 'tool_call') {
    if (span.toolInput) details.push({ key: 'in', label: 'Input', children: <pre style={{ whiteSpace: 'pre-wrap', margin: 0, fontSize: 11, lineHeight: 1.5, color: '#d4d4d8' }}>{span.toolInput}</pre> });
    if (span.toolOutput) details.push({ key: 'out', label: 'Output', children: <pre style={{ whiteSpace: 'pre-wrap', margin: 0, fontSize: 11, lineHeight: 1.5, color: '#d4d4d8' }}>{span.toolOutput}</pre> });
  }

  const hasDetails = details.length > 0;

  return (
    <Card
      size="small"
      className={`span-card ${hasDetails ? 'expandable' : ''}`}
      onClick={() => hasDetails && setExpanded(!expanded)}
    >
      <div className="span-header">
        <span className={`span-kind ${meta.cls}`}>{meta.icon} {meta.label}</span>
        <span className="span-name">{span.name}</span>
        {span.duration != null && <span className="span-meta">{span.duration}ms</span>}
        {span.totalTokens != null && <span className="span-meta">{span.totalTokens} tok</span>}
        {span.isTarget && <span className="span-target"><AimOutlined /> TARGET</span>}
        {span.error && <span className="mtag mtag-fail"><CloseCircleOutlined /> ERR</span>}
        {!span.error && span.kind === 'tool_call' && <span style={{ color: '#4ade80', fontSize: 11 }}><CheckCircleOutlined /></span>}
        {hasDetails && (
          <DownOutlined style={{
            fontSize: 9,
            color: '#71717a',
            marginLeft: 'auto',
            transition: 'transform 0.2s',
            transform: expanded ? 'rotate(180deg)' : 'rotate(0deg)',
          }} />
        )}
      </div>
      {expanded && details.length > 0 && (
        <div onClick={(e) => e.stopPropagation()} style={{ marginTop: 8 }}>
          <Collapse ghost size="small" items={details} />
        </div>
      )}
    </Card>
  );
}

export default function SpanTimeline({ spans }: SpanTimelineProps) {
  const grouped = new Map<number, Span[]>();
  for (const span of spans) {
    const list = grouped.get(span.iteration) || [];
    list.push(span);
    grouped.set(span.iteration, list);
  }

  const iterations = Array.from(grouped.keys()).sort((a, b) => a - b);

  return (
    <div>
      {iterations.map((iter) => (
        <div key={iter} style={{ marginBottom: 16 }}>
          <div className="iteration-badge">
            <span style={{ width: 5, height: 5, borderRadius: '50%', background: '#22d3ee', display: 'inline-block' }} />
            Round {iter}
          </div>
          <Timeline
            items={grouped.get(iter)!.map((span) => ({
              color: span.error ? '#fb7185' : kindColor[span.kind] || '#22d3ee',
              children: <SpanCard span={span} />,
            }))}
          />
        </div>
      ))}
    </div>
  );
}
