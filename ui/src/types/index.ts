export type SpanKind = 'llm_call' | 'tool_call' | 'llm_compress';

export interface Span {
  spanId: string;
  parentId?: string;
  kind: SpanKind;
  name: string;
  iteration: number;
  startTime: string;
  endTime?: string;
  duration?: number;

  inputMessages?: number;
  totalTokens?: number;
  finishReason?: string;
  llmInput?: string;
  llmOutput?: string;

  toolInput?: string;
  toolOutput?: string;
  isTarget?: boolean;

  error?: string;
}

export interface Trace {
  id: string;
  agentName: string;
  model: string;
  userPrompt: string;
  targetSkill?: string;
  startTime: string;
  endTime?: string;
  totalTokens: number;
  iterations: number;
  success: boolean;
  spans: Span[];
}

export interface ScoreItem {
  name: string;
  desc: string;
}

export interface Verdict {
  info: ScoreItem;
  pass: boolean;
  score: number;
  reason: string;
}

export interface Report {
  traceId: string;
  agentName: string;
  model: string;
  userPrompt: string;
  targetSkill: string;
  totalTokens: number;
  iterations: number;
  duration: number;
  scores: Verdict[];
  pass: boolean;
}

export interface CompareReport {
  traceA: Trace;
  traceB: Trace;
  reportA: Report;
  reportB: Report;
  scores: Verdict[];
  conclusion: string;
}

export interface EvalRecord {
  id: string;
  type: 'single' | 'compare';
  createdAt: string;
  report?: Report;
  trace?: Trace;
  compareReport?: CompareReport;
}
