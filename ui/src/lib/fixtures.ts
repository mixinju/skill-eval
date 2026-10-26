import type { RawTrace, ScoreView, Span } from '../types'
import { recordFromTrace, withFixtureScores } from './model'

const BASE = Date.parse('2026-04-20T02:00:00Z')

function iso(offsetMs: number): string {
  return new Date(BASE + offsetMs).toISOString()
}

function span(id: string, fields: Omit<Span, 'spanId' | 'startTime' | 'endTime'> & { at: number; dur: number }): Span {
  return {
    spanId: id,
    parentId: fields.parentId,
    kind: fields.kind,
    name: fields.name,
    iteration: fields.iteration,
    startTime: iso(fields.at),
    endTime: iso(fields.at + fields.dur),
    duration: fields.dur,
    inputMessages: fields.inputMessages,
    totalTokens: fields.totalTokens,
    finishReason: fields.finishReason,
    llmInput: fields.llmInput,
    llmOutput: fields.llmOutput,
    toolInput: fields.toolInput,
    toolOutput: fields.toolOutput,
    isTarget: fields.isTarget,
    error: fields.error,
  }
}

const codexTrace: RawTrace = {
  id: 'fixture-codex-pdf',
  agentName: 'Codex',
  model: 'codex',
  runner: 'codex',
  userPrompt: '查询南京天气，并把结果保存成 PDF',
  targetSkill: 'pdf',
  startTime: iso(0),
  endTime: iso(9800),
  totalTokens: 4200,
  iterations: 3,
  success: true,
  spans: [
    span('codex-llm-1', {
      kind: 'llm_call',
      name: 'response',
      iteration: 1,
      at: 200,
      dur: 1600,
      inputMessages: 2,
      totalTokens: 1400,
      finishReason: 'tool_calls',
      llmOutput: JSON.stringify({
        role: 'assistant',
        content: '先加载 pdf skill，再查天气。',
        tool_calls: [{ function: { name: 'use_skill', arguments: '{"name":"pdf"}' } }],
      }),
    }),
    span('codex-tool-1', {
      kind: 'tool_call',
      name: 'use_skill',
      iteration: 1,
      at: 1900,
      dur: 180,
      parentId: 'codex-llm-1',
      isTarget: true,
      toolInput: '{"name":"pdf"}',
      toolOutput: 'SKILL.md已加载',
    }),
    span('codex-llm-2', {
      kind: 'llm_call',
      name: 'response',
      iteration: 2,
      at: 2300,
      dur: 1400,
      inputMessages: 4,
      totalTokens: 1500,
      finishReason: 'tool_calls',
      llmOutput: JSON.stringify({
        role: 'assistant',
        content: '',
        tool_calls: [{ function: { name: 'shell', arguments: '{"command":"python3 create_weather_pdf.py"}' } }],
      }),
    }),
    span('codex-tool-2', {
      kind: 'tool_call',
      name: 'shell',
      iteration: 2,
      at: 3800,
      dur: 900,
      parentId: 'codex-llm-2',
      toolInput: '{"command":"python3 create_weather_pdf.py"}',
      toolOutput: '天气信息已写入 南京天气.pdf',
    }),
    span('codex-llm-3', {
      kind: 'llm_call',
      name: 'response',
      iteration: 3,
      at: 4900,
      dur: 1200,
      inputMessages: 6,
      totalTokens: 1300,
      finishReason: 'tool_calls',
      llmOutput: JSON.stringify({
        role: 'assistant',
        content: '',
        tool_calls: [
          {
            function: {
              name: 'finish',
              arguments: JSON.stringify({
                result: '已查询南京天气并写成 PDF。',
                artifacts: ['南京天气.pdf'],
              }),
            },
          },
        ],
      }),
    }),
    span('codex-tool-3', {
      kind: 'tool_call',
      name: 'finish',
      iteration: 3,
      at: 6200,
      dur: 40,
      parentId: 'codex-llm-3',
      toolInput: JSON.stringify({ result: '已查询南京天气并写成 PDF。', artifacts: ['南京天气.pdf'] }),
      toolOutput: JSON.stringify({ result: '已查询南京天气并写成 PDF。', artifacts: ['南京天气.pdf'] }),
    }),
  ],
}

const claudeTrace: RawTrace = {
  id: 'fixture-claude-xlsx',
  agentName: 'Claude Code',
  model: 'claude-code',
  runner: 'claude-code',
  userPrompt: '把南京天气整理成 xlsx 表格',
  targetSkill: 'xlsx',
  startTime: iso(0),
  endTime: iso(6400),
  totalTokens: 3100,
  iterations: 2,
  success: false,
  spans: [
    span('claude-llm-1', {
      kind: 'llm_call',
      name: 'assistant',
      iteration: 1,
      at: 150,
      dur: 1800,
      inputMessages: 2,
      totalTokens: 1600,
      finishReason: 'tool_calls',
      llmOutput: JSON.stringify({
        role: 'assistant',
        content: '直接写一个表格脚本。',
        tool_calls: [{ function: { name: 'Bash', arguments: '{"command":"python3 write_weather_xlsx.py"}' } }],
      }),
    }),
    span('claude-tool-1', {
      kind: 'tool_call',
      name: 'Bash',
      iteration: 1,
      at: 2100,
      dur: 420,
      parentId: 'claude-llm-1',
      toolInput: '{"command":"python3 write_weather_xlsx.py"}',
      toolOutput: '',
      error: 'python3: write_weather_xlsx.py: No such file or directory',
    }),
    span('claude-llm-2', {
      kind: 'llm_call',
      name: 'assistant',
      iteration: 2,
      at: 2700,
      dur: 1500,
      inputMessages: 4,
      totalTokens: 1500,
      finishReason: 'stop',
      llmOutput: JSON.stringify({
        role: 'assistant',
        content: '脚本不存在，任务没有完成。',
      }),
    }),
  ],
}

const codexArtifact: ScoreView = {
  name: '产物评分',
  desc: '针对输出',
  availability: 'fixture',
  pass: true,
  score: 0.86,
  reason: '样例：PDF 写了城市和天气，能对上用户要求。',
}

const codexProcess: ScoreView = {
  name: '执行过程',
  desc: '评测执行过程',
  availability: 'fixture',
  pass: true,
  score: 0.78,
  reason: '样例：先命中 pdf，再执行脚本，路径短。',
}

const claudeProcess: ScoreView = {
  name: '执行过程',
  desc: '评测执行过程',
  availability: 'fixture',
  pass: false,
  score: 0.3,
  reason: '样例：没有加载 xlsx，Bash 直接失败。',
}

export const fixtureRecords = [
  withFixtureScores(recordFromTrace({ trace: codexTrace, source: 'fixture', file: '样例' }), [codexArtifact, codexProcess]),
  withFixtureScores(recordFromTrace({ trace: claudeTrace, source: 'fixture', file: '样例' }), [claudeProcess]),
]
