# Skill Eval 技术设计文档

## 1. 项目概述

Skill Eval 是一个 Skill A/B 评测框架，用于评估 AI Agent 中技能（Skill）的调用和执行效果。支持：

- 两个不同版本 Skill 的对比评测（v1 vs v2）
- 有 Skill vs 无 Skill 的增量价值对比
- 基于产物文件的 LLM 自动评分 + 人工确认

## 2. 整体架构

```
eval.Runner
  ├── 加载 Cases
  ├── for each case:
  │     ├── 创建隔离 workspace A / B
  │     ├── go orchestrator.Run(agentA, case) ── 并发
  │     ├── go orchestrator.Run(agentB, case) ── 并发
  │     ├── wait both → PairResult
  │     └── 收集产物文件路径
  └── LLM 评分 → 人工确认 → 输出报告
```

## 3. 核心设计决策

### 3.1 分层架构：Agent → Orchestrator → RunContext

| 层级 | 职责 | 特性 |
|------|------|------|
| **Agent** | 静态配置（模型、Skill、工作空间等） | 不可变，可复用 |
| **Orchestrator** | Agent Loop 运行引擎 | 无状态，纯引擎 |
| **RunContext** | 单次运行的上下文 | 持有 State 和事件回调 |

**设计理由**：Agent 是可复用的配置，Orchestrator 是纯引擎，RunContext 是每次运行的全部状态。评测时对同一个 Agent 跑多个 Case 很自然。

### 3.2 事件机制：回调优于 Channel

选择 `EventHandler func(Event)` 回调而非 `chan Event`。

**回调方案优势**：
- **简单**——不需要管 chan 的创建、消费、关闭，不需要额外 goroutine
- **灵活**——调用方决定怎么处理：打日志、写文件、转发到 chan 都行
- **可组合**——多个 handler 可以组合成链

**为什么不用 Channel**：
- 事件流的生命周期应该跟"一次运行"绑定，不是跟"引擎"绑定
- chan 放在 Orchestrator 上会导致并发运行的事件混在一起
- 调用方必须起 goroutine 消费，否则 Agent 循环会阻塞
- chan 的创建、消费、关闭生命周期管理复杂

**使用示例**：
```go
// 持久化到 JSONL（事件回调的一种实现）
orch.Run(agent, input, func(e Event) {
    switch e.Type {
    case EventLLMCall:
        store.SaveLLMCall(e.Data)
    case EventToolExec:
        store.SaveToolCall(e.Data)
    }
})

// 组合多个 handler
orch.Run(agent, input, composeHandlers(
    jsonFileLogger("run_001.jsonl"),
    consoleLogger(),
    metricsCollector(),
))
```

### 3.3 Agent Loop 终止策略：Finish Tool + 纯文本兜底

判断 Agent 是否完成任务的方案：

```
if 有 tool_calls:
    if 包含 finish tool → 完成，取 finish 的 result 作为最终输出
    else → 执行 tools，继续循环
else (纯文本):
    → 也视为完成，取文本内容作为最终输出
```

**设计理由**：
- **正常路径**：模型调 `finish` 明确交付结果，拿到干净的结构化输出，评测好打分
- **兜底路径**：模型忘了调 `finish` 直接回复文本，也不会死循环卡住
- 不用特定文本标识（如 `[DONE]`），因为不够可靠和稳定

**终止原因区分**：
```go
const (
    StopFinish    StopReason = "finish"       // Agent 主动调用 finish
    StopMaxInters StopReason = "max_inters"   // 超出最大迭代次数
    StopMaxTokens StopReason = "max_tokens"   // 超出最大 token
    StopError     StopReason = "error"        // 不可恢复错误
    StopTextReply StopReason = "text_reply"   // 纯文本回复（兜底）
)
```

评测时需要知道是模型主动结束还是被强制截断。

### 3.4 直接使用 OpenAI SDK 类型，不做自定义封装

初始方案中定义了 `agent.Message`、`agent.ToolCall`、`agent.FunctionCall` 等自定义类型，后改为直接使用 `openai-go` SDK 原生类型。

**改造原因**：
- OpenAI 的消息格式已经是事实标准（智谱、通义、DeepSeek 都兼容）
- 项目只用 openai-go SDK，不会换 SDK
- 自定义类型引入了大量转换代码（`convertMessages`），是 bug 来源
- 改造后删除约 80 行转换代码

**改造后**：
- 消息历史直接使用 `[]openai.ChatCompletionMessageParamUnion`
- LLM 响应直接使用 `*openai.ChatCompletion`
- 用 `choice.Message.ToParam()` 把 LLM 响应追加到消息历史，零转换

**保留 ToolCallRecord 的理由**：SDK 没有对应的完整调用记录类型——SDK 将 tool call 的请求侧（`ChatCompletionMessageToolCallUnion`，包含 ID/函数名/参数）和响应侧（`ToolMessage`，包含返回内容）分开存储，没有一个类型把调用输入 + 执行输出 + 错误 + 迭代轮次组合在一起。`ToolCallRecord` 补充了这个聚合视图。

### 3.5 Workspace 隔离

A/B 评测必须使用独立的工作目录，避免：
- 两个 Agent 产出同名文件互相覆盖
- 文件系统操作互相干扰
- 产物归属不清

每个 Case 的每个 Agent 有独立的 workspace：
```
{outputDir}/{caseID}/a/   ← Agent A 的工作目录
{outputDir}/{caseID}/b/   ← Agent B 的工作目录
```

FileSystem 和 Bash 工具都绑定 workspace，限制访问范围。

### 3.6 外层循环与内层循环分离

- **外层循环**（eval.Runner）：遍历评测 Case，不在 Orchestrator 里
- **内层循环**（Orchestrator.Run）：单个 Case 的 Agent Loop

**设计理由**：评测用例的遍历是**评测逻辑**，不是**编排逻辑**。Orchestrator 的 `Run` 只负责跑单个 case，外层循环放在 eval 层。

## 4. Tool 清单

| Tool | 文件 | 职责 |
|------|------|------|
| **FileSystem** | `tool/filesystem.go` | 文件读写、编辑、目录列表（workspace 隔离） |
| **Bash** | `tool/bash.go` | 通用命令执行（workspace 作为 cwd） |
| **Finish** | `tool/finish.go` | Agent 主动交付结果 + 产物文件列表 |
| **UseSkill** | `tool/use_skill.go` | 从 skill registry 加载技能内容 |
| **GetWeather** | `tool/get_weather.go` | 基础验证用的示例工具 |

**关于 Python/JS 执行**：不单独设 CodeExec 工具，直接通过 Bash 执行 `python3 xxx.py` / `node xxx.js`。Claude Code 也是同样的做法——模型足够聪明，会自己拼安装和执行命令。单独封装是过度设计。

## 5. 模块结构

```
skill-eval/
├── agent/
│   ├── types.go          # Agent, RunResult, StopReason, ToolCallRecord
│   ├── run_context.go    # RunContext, State, Event, EventHandler
│   └── orchestrator.go   # Orchestrator, ChatFunc, Agent Loop
├── eval/
│   ├── case.go           # Case 定义与 JSON 加载
│   ├── runner.go         # EvalPair, PairResult, Runner（A/B 并发评测）
│   └── scorer.go         # LLM 对比评分
├── providers/
│   └── openai.go         # OpenAIProvider（直接返回 SDK 类型）
├── skill/
│   └── skill.go          # Skill 定义、SKILL.md 解析、资源加载
├── tool/
│   ├── types.go          # Tool 接口、BaseToolInfo
│   ├── filesystem.go     # 文件操作（workspace 隔离）
│   ├── bash.go           # Shell 命令执行
│   ├── finish.go         # 任务完成信号
│   ├── use_skill.go      # 技能加载
│   └── get_weather.go    # 示例工具
├── doc/
│   └── design.md         # 本文档
├── main.go               # CLI 入口
└── go.mod
```

## 6. 运行流程

```
1. main.go 解析命令行参数
2. 加载 Skill A / B（从 SKILL.md）
3. 构造 Agent A / B（绑定不同 Skill）
4. 加载评测 Cases（从 JSON 文件）
5. 创建 Runner，配置事件回调（JSONL 持久化）
6. 对每个 Case：
   a. 创建隔离 workspace A / B
   b. goroutine 并发运行 Agent A 和 Agent B
   c. 等待两者完成，收集 PairResult
7. 所有 Case 完成后，LLM 评分（读取产物文件内容对比打分）
8. 输出评测报告（report.json）
```

## 7. 使用方式

```bash
skill-eval cases.json \
  --skill-a ./skills/pdf-v1/SKILL.md \
  --skill-b ./skills/pdf-v2/SKILL.md \
  --model glm-5 \
  --base-url https://open.bigmodel.cn/api/paas/v4 \
  --api-key <your-key> \
  --max-iters 10 \
  --output ./eval-output
```

不指定 `--skill-b` 时，Agent B 为无 Skill 对照组。
