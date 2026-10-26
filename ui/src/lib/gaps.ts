export interface GapNote {
  title: string
  body: string
}

/** 前端已经接上的契约，以及后端还没提供的部分。 */
export const BACKEND_GAPS: GapNote[] = [
  {
    title: '没有评测 HTTP API',
    body: 'Go 程序是命令行。DefaultTracer 把 Trace 写到 logs/traces/*.json。页面通过 Vite 的 /__local/evals 读这些文件，这是开发适配层，不是后端接口。',
  },
  {
    title: '执行后端目前只有自研内核',
    body: 'Trace.runner 可以是 kernel、codex、claude-code。Orchestrator 只会写成 kernel。Codex 和 Claude Code 没有执行器；这两条记录是界面样例，用来把选择和详情先做出来。',
  },
  {
    title: '评测报告没有落盘',
    body: 'eval.Exec 在 main 里把报告打到标准输出，不写 JSON。产物质量和执行过程分都要再调模型，页面不会编造。若以后把 eval.Report 放到 logs/reports/*.json，刷新后会直接显示。',
  },
  {
    title: '单条「是否执行成功」评分器是空的',
    body: 'SuccessScorer 对单条 Trace 目前返回空 Verdict。页面改用 Trace.success。Skill 是否命中则按 SkillHitScorer 的规则，从 use_skill 的参数计算。',
  },
  {
    title: '旧 Span 的 duration 单位不一致',
    body: '部分历史文件把 span.duration 存成纳秒，且所有 Span 共用同一个 spanId。耗时改为用 startTime / endTime。新的运行会写下 runner，并为每个 Span 生成独立 id。',
  },
]
