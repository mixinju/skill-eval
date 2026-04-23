import type { EvalRecord, Span, Trace, Report, CompareReport } from '../types';

function makeSpans(iterations: number, targetSkill?: string): Span[] {
  const spans: Span[] = [];
  let spanIdx = 0;
  for (let i = 1; i <= iterations; i++) {
    const llmId = `span-${++spanIdx}`;
    spans.push({
      spanId: llmId,
      kind: 'llm_call',
      name: 'chat_completion',
      iteration: i,
      startTime: new Date(Date.now() - (iterations - i + 1) * 30000).toISOString(),
      endTime: new Date(Date.now() - (iterations - i + 1) * 30000 + 2000).toISOString(),
      duration: 1800 + Math.floor(Math.random() * 1200),
      inputMessages: i === 1 ? 2 : 4 + i,
      totalTokens: 800 + Math.floor(Math.random() * 1200),
      finishReason: i === iterations ? 'stop' : 'tool_calls',
      llmInput: `[系统提示 + 用户输入] 第${i}轮对话`,
      llmOutput: i === iterations ? '任务完成，调用 finish 工具。' : `需要调用工具来完成任务，第${i}步...`,
    });

    if (i < iterations) {
      const toolName = i === 1 && targetSkill ? 'use_skill' : ['bash', 'filesystem', 'get_weather'][i % 3];
      spans.push({
        spanId: `span-${++spanIdx}`,
        parentId: llmId,
        kind: 'tool_call',
        name: toolName,
        iteration: i,
        startTime: new Date(Date.now() - (iterations - i + 1) * 30000 + 2500).toISOString(),
        endTime: new Date(Date.now() - (iterations - i + 1) * 30000 + 4000).toISOString(),
        duration: 800 + Math.floor(Math.random() * 2000),
        toolInput: toolName === 'use_skill'
          ? JSON.stringify({ name: targetSkill })
          : `{"command": "step-${i}"}`,
        toolOutput: `执行成功，返回结果...`,
        isTarget: toolName === 'use_skill',
      });
    } else {
      spans.push({
        spanId: `span-${++spanIdx}`,
        parentId: llmId,
        kind: 'tool_call',
        name: 'finish',
        iteration: i,
        startTime: new Date(Date.now() - 5000).toISOString(),
        endTime: new Date(Date.now() - 3000).toISOString(),
        duration: 200,
        toolInput: JSON.stringify({ result: '任务完成', artifacts: ['/output/result.pdf'] }),
        toolOutput: '已完成',
      });
    }
  }
  return spans;
}

function makeTrace(id: string, model: string, skill: string, iterations: number, success: boolean): Trace {
  const start = new Date(Date.now() - iterations * 30000);
  const end = new Date();
  return {
    id,
    agentName: '天气小助手',
    model,
    userPrompt: `查询南京的天气，最后把结果保存为pdf文件`,
    targetSkill: skill,
    startTime: start.toISOString(),
    endTime: end.toISOString(),
    totalTokens: 2000 + Math.floor(Math.random() * 5000),
    iterations,
    success,
    spans: makeSpans(iterations, skill),
  };
}

function makeReport(trace: Trace, pass: boolean): Report {
  return {
    traceId: trace.id,
    agentName: trace.agentName,
    model: trace.model,
    userPrompt: trace.userPrompt,
    targetSkill: trace.targetSkill || '',
    totalTokens: trace.totalTokens,
    iterations: trace.iterations,
    duration: new Date(trace.endTime!).getTime() - new Date(trace.startTime).getTime(),
    pass,
    scores: [
      {
        info: { name: 'Skill是否命中', desc: '根据执行链路，评测是否命中目标Skill' },
        pass: true,
        score: 1.0,
        reason: `命中目标 skill: ${trace.targetSkill}`,
      },
      {
        info: { name: '是否执行成功', desc: '整个智能体执行流程是否完成任务' },
        pass: trace.success,
        score: trace.success ? 1.0 : 0.0,
        reason: trace.success ? '执行流程正常完成' : '执行过程中出现错误',
      },
      {
        info: { name: '产物评分', desc: '由大模型评估生成的产物文件' },
        pass,
        score: pass ? 0.85 : 0.35,
        reason: pass ? '产物文件内容完整，符合用户要求' : '产物文件内容不完整',
      },
      {
        info: { name: '执行过程', desc: '评测执行过程的效率和质量' },
        pass: true,
        score: 0.72,
        reason: '执行过程效率适中，工具调用准确，Token消耗合理',
      },
    ],
  };
}

const trace1 = makeTrace('trace-001', 'glm-5', 'pdf', 4, true);
const trace2 = makeTrace('trace-002', 'gpt-4o', 'pdf', 3, true);
const trace3 = makeTrace('trace-003', 'glm-5', 'xlsx', 5, false);
const trace4 = makeTrace('trace-004', 'claude-sonnet-4-6', 'pdf', 3, true);
const trace5 = makeTrace('trace-005', 'glm-5', 'pdf', 6, true);
const trace6 = makeTrace('trace-006', 'gpt-4o', 'pdf', 4, true);
const trace7 = makeTrace('trace-007', 'claude-sonnet-4-6', 'xlsx', 3, false);
const trace8 = makeTrace('trace-008', 'glm-5', 'docx', 4, true);

export const mockRecords: EvalRecord[] = [
  {
    id: 'eval-001',
    type: 'single',
    createdAt: '2026-04-23T10:30:00Z',
    trace: trace1,
    report: makeReport(trace1, true),
  },
  {
    id: 'eval-002',
    type: 'single',
    createdAt: '2026-04-23T11:00:00Z',
    trace: trace2,
    report: makeReport(trace2, true),
  },
  {
    id: 'eval-003',
    type: 'single',
    createdAt: '2026-04-23T11:30:00Z',
    trace: trace3,
    report: makeReport(trace3, false),
  },
  {
    id: 'eval-004',
    type: 'compare',
    createdAt: '2026-04-23T12:00:00Z',
    compareReport: {
      traceA: trace1,
      traceB: trace2,
      reportA: makeReport(trace1, true),
      reportB: makeReport(trace2, true),
      scores: [
        {
          info: { name: '执行路径对比', desc: '对比两个Skill的执行路径差异' },
          pass: true,
          score: 0.78,
          reason: 'glm-5 使用了4轮迭代，gpt-4o 仅用3轮，gpt-4o 执行效率更高。两者均命中目标 Skill，但 glm-5 的 Token 消耗更高。',
        },
        {
          info: { name: '产物质量对比', desc: '对比两个Skill产出的产物质量' },
          pass: true,
          score: 0.82,
          reason: '两者产出的 PDF 文件质量相当，gpt-4o 生成的格式略优。',
        },
      ],
      conclusion: '综合对比：gpt-4o 在执行效率（3轮 vs 4轮）和 Token 消耗上优于 glm-5。glm-5 的工具调用路径更长，存在一次额外的 bash 调用。产物质量两者相当，建议优先使用 gpt-4o 执行该类任务。',
    } as CompareReport,
  },
  {
    id: 'eval-005',
    type: 'single',
    createdAt: '2026-04-23T13:00:00Z',
    trace: trace4,
    report: makeReport(trace4, true),
  },
  {
    id: 'eval-006',
    type: 'compare',
    createdAt: '2026-04-23T14:00:00Z',
    compareReport: {
      traceA: trace5,
      traceB: trace6,
      reportA: makeReport(trace5, true),
      reportB: makeReport(trace6, true),
      scores: [
        {
          info: { name: '执行路径对比', desc: '对比两个Skill的执行路径差异' },
          pass: false,
          score: 0.55,
          reason: 'glm-5 使用了6轮迭代，明显多于 gpt-4o 的4轮。glm-5 在第3-4轮存在重复的工具调用，执行效率较低。',
        },
        {
          info: { name: '产物质量对比', desc: '对比两个Skill产出的产物质量' },
          pass: true,
          score: 0.75,
          reason: '产物质量 gpt-4o 略优，glm-5 生成的内容存在格式问题。',
        },
      ],
      conclusion: '综合对比：本次测试中 gpt-4o 表现明显优于 glm-5。glm-5 迭代次数过多（6轮 vs 4轮），存在重复工具调用，Token 消耗高出约40%。建议检查 glm-5 的 system prompt 是否需要优化以减少冗余调用。',
    } as CompareReport,
  },
  {
    id: 'eval-007',
    type: 'single',
    createdAt: '2026-04-23T15:00:00Z',
    trace: trace7,
    report: makeReport(trace7, false),
  },
  {
    id: 'eval-008',
    type: 'single',
    createdAt: '2026-04-23T16:00:00Z',
    trace: trace8,
    report: makeReport(trace8, true),
  },
];
