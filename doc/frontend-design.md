# Skill-Eval 前端可视化技术方案

## Context

skill-eval 当前是纯 CLI 工具，评测过程和结果只能通过终端日志和 `report.json` 查看。目标：
1. **实时 Dashboard**：在 Web 页面上观察 Agent A/B 的执行过程（LLM 调用、Tool 调用、迭代进度）
2. **静态报告**：评测结束后生成可离线浏览的 HTML 报告
3. **双触发**：Web 页面和 CLI 都能启动评测任务

核心优势：项目已有完善的 `EventHandler` 回调机制（`agent/run_context.go`），只需将事件广播到 SSE 连接即可实现实时推送，改动成本低。

---

## 架构总览

```
┌─────────────────────────────────────────────────┐
│  React + Vite (web/)                             │
│  ├── 配置面板（触发评测）                          │
│  ├── 实时 Dashboard（SSE 事件流）                 │
│  └── 报告查看器（历史报告列表 + 详情）              │
└────────────────────┬────────────────────────────┘
                     │ HTTP API + SSE
┌────────────────────┴────────────────────────────┐
│  Go HTTP Server (server/)                        │
│  ├── POST /api/eval/start     启动评测            │
│  ├── GET  /api/eval/events    SSE 事件流          │
│  ├── GET  /api/eval/status    当前运行状态         │
│  ├── GET  /api/reports        历史报告列表         │
│  ├── GET  /api/reports/:id    报告详情             │
│  └── GET  /api/reports/:id/html  静态 HTML 报告   │
└────────────────────┬────────────────────────────┘
                     │
┌────────────────────┴────────────────────────────┐
│  Core (现有模块，少量改造)                         │
│  eval.Runner / agent.Orchestrator / providers     │
└─────────────────────────────────────────────────┘
```

---

## 一、Go 后端改造

### 1.1 新增 `server/` 包 — HTTP 服务层

**文件**: `server/server.go`

- 使用标准库 `net/http` + `ServeMux`（不引入 gin 等框架，保持零外部依赖风格）
- 提供 REST API 和 SSE 端点
- 管理评测任务的生命周期（启动、状态查询、事件广播）

**API 设计**:

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/eval/start` | 启动评测任务，接收 JSON 参数 |
| GET | `/api/eval/events` | SSE 端点，实时推送 Event |
| GET | `/api/eval/status` | 返回当前评测状态（running/idle/scoring） |
| GET | `/api/reports` | 列出 outputDir 下所有历史报告 |
| GET | `/api/reports/{id}` | 返回 report.json 内容 |
| GET | `/api/reports/{id}/events` | 返回 events.jsonl 内容 |

**启动评测请求体**:
```json
{
  "cases_path": "./cases.json",
  "skill_a_path": "./skills/pdf-v1/SKILL.md",
  "skill_b_path": "",
  "model": "glm-5",
  "base_url": "https://open.bigmodel.cn/api/paas/v4",
  "api_key": "xxx",
  "max_iters": 10
}
```

### 1.2 新增 `server/sse.go` — SSE 事件广播器

**文件**: `server/sse.go`

- `SSEBroker` 结构体：管理多个 SSE 客户端连接
- 方法：`Subscribe(w http.ResponseWriter)` / `Broadcast(event)`
- 利用现有 `agent.EventHandler` 回调——构造一个 handler 将 Event 同时写入 JSONL 文件 + 广播到 SSE

**事件格式（SSE data 字段）**:
```json
{
  "type": "tool_exec",
  "case_id": "case_1",
  "label": "A",
  "iteration": 3,
  "data": { "tool_name": "bash", "input": "...", "output": "..." },
  "timestamp": "2026-04-17T10:30:00Z"
}
```

新增广播的事件类型（在现有 4 种之外扩展）:

| 事件类型 | 触发时机 | 前端用途 |
|---------|---------|---------|
| `eval_start` | Runner.Run 开始 | 显示整体进度条，初始化面板 |
| `case_start` | 每个 Case 开始 | 切换 Case Tab |
| `case_done` | 每个 Case A/B 都完成 | 更新进度，显示 Case 结果 |
| `llm_call` | 已有 | 显示 LLM 调用详情 |
| `tool_exec` | 已有 | 显示 Tool 调用和输出 |
| `finish` | 已有 | 标记 Agent 完成 |
| `error` | 已有 | 显示错误信息 |
| `scoring_start` | Scorer 开始打分 | 显示 "评分中" 状态 |
| `scoring_done` | 评分完成 | 显示最终评分 |
| `eval_done` | 整个评测结束 | 显示完成状态，生成报告链接 |

### 1.3 改造 `main.go` — 双模式入口

**文件**: `main.go`

新增 `--serve` 标志：
- `skill-eval --serve [--port 8080] [--output ./eval-output]`：启动 Web 服务模式
- 不带 `--serve`：保持现有 CLI 行为不变
- CLI 模式下，如果同时指定了 `--serve`，则 CLI 触发评测的同时也启动 HTTP 服务

### 1.4 改造 `eval/runner.go` — 增加阶段性事件

**文件**: `eval/runner.go`

在 `Runner.Run()` 循环中增加 `eval_start`、`case_start`、`case_done`、`eval_done` 等事件的 Emit，复用现有的 `EventHandler` 机制。改动约 20 行。

### 1.5 新增 `server/report.go` — 静态 HTML 报告生成

**文件**: `server/report.go`

- 评测结束后自动调用，读取 `report.json` + `events.jsonl`
- 使用 Go `html/template` 生成自包含的单 HTML 文件（CSS/JS 内联）
- 报告内容：
  - 评测概览（时间、模型、Skill 信息、Case 数量）
  - 每个 Case 的 A/B 对比卡片（迭代数、Token 用量、StopReason、评分）
  - 可展开的执行时间线（每轮迭代的 LLM 调用和 Tool 调用）
  - 评分汇总表格和柱状图
- 输出路径：`{outputDir}/{timestamp}/report.html`

---

## 二、React 前端

### 2.1 项目结构

```
web/
├── package.json
├── vite.config.ts        # dev proxy → Go 后端
├── index.html
├── src/
│   ├── main.tsx
│   ├── App.tsx            # 路由：/ → Dashboard, /reports → 报告列表
│   ├── api/
│   │   ├── client.ts      # fetch 封装
│   │   └── sse.ts         # EventSource 封装，自动重连
│   ├── pages/
│   │   ├── Dashboard.tsx      # 主页：配置 + 实时执行
│   │   ├── ReportList.tsx     # 历史报告列表
│   │   └── ReportDetail.tsx   # 单次报告详情
│   ├── components/
│   │   ├── ConfigPanel.tsx        # 评测参数配置表单
│   │   ├── ProgressBar.tsx        # 整体进度条
│   │   ├── CaseTimeline.tsx       # 单个 Case 的执行时间线
│   │   ├── AgentPanel.tsx         # 单个 Agent 的执行面板
│   │   ├── ToolCallCard.tsx       # Tool 调用卡片
│   │   ├── LLMCallCard.tsx        # LLM 调用卡片（可折叠）
│   │   ├── ScoreCompare.tsx       # A/B 评分对比
│   │   └── ArtifactViewer.tsx     # 产物文件预览
│   ├── hooks/
│   │   └── useEvalSSE.ts      # SSE 连接 hook，管理事件状态
│   └── types/
│       └── eval.ts            # TypeScript 类型定义
```

### 2.2 核心页面设计

#### Dashboard 页面（主页）

```
┌─────────────────────────────────────────────────────┐
│  Skill Eval Dashboard                    [历史报告]  │
├─────────────────────────────────────────────────────┤
│  ┌─ 配置面板 ─────────────────────────────────────┐ │
│  │ Cases: [./cases.json    ]  Model: [glm-5     ] │ │
│  │ Skill A: [path/SKILL.md ]  Skill B: [可选     ] │ │
│  │ API URL: [_____________ ]  API Key: [_________ ] │ │
│  │ Max Iters: [10]                [▶ 开始评测]      │ │
│  └────────────────────────────────────────────────┘ │
│                                                     │
│  ┌─ 进度 ────────────────────────────────────────┐  │
│  │ Case 3/10  ████████░░░░░░░░  30%              │  │
│  │ Status: Running case_3                        │  │
│  └───────────────────────────────────────────────┘  │
│                                                     │
│  ┌─ Case Tabs ───────────────────────────────────┐  │
│  │ [case_1 ✓] [case_2 ✓] [case_3 ◉] [case_4 ○] │  │
│  └───────────────────────────────────────────────┘  │
│                                                     │
│  ┌─ Agent A ──────────────┬─ Agent B ────────────┐  │
│  │ Iteration 3/10         │ Iteration 2/10       │  │
│  │ Tokens: 12,340         │ Tokens: 8,920        │  │
│  │                        │                      │  │
│  │ ▼ Iter 1               │ ▼ Iter 1             │  │
│  │   LLM Call (1.2s)      │   LLM Call (0.8s)    │  │
│  │   bash: ls -la         │   read_file: ...     │  │
│  │     → output...        │     → output...      │  │
│  │ ▼ Iter 2               │ ▼ Iter 2             │  │
│  │   LLM Call (2.1s)      │   LLM Call (1.5s)    │  │
│  │   write_file: ...      │   bash: python..     │  │
│  │ ▼ Iter 3 (current)     │     → running...     │  │
│  │   LLM Call...          │                      │  │
│  │   waiting...           │                      │  │
│  └────────────────────────┴──────────────────────┘  │
│                                                     │
│  ┌─ 评分结果（Case 完成后显示）──────────────────┐    │
│  │ Case case_1:  A=8  B=6  |  原因: A 产物更完整  │    │
│  │ Case case_2:  A=7  B=7  |  原因: 两者表现相当  │    │
│  └───────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────┘
```

#### 报告详情页面

- 评测元信息（时间、模型、Skill A/B 名称和版本）
- A/B 评分汇总表格 + 柱状图（可用 recharts 库）
- 每个 Case 可展开查看：
  - 时间线回放（与 Dashboard 相同组件，但数据来自 events.jsonl）
  - 产物文件内容对比（代码高亮 diff 视图）
  - 评分和理由

### 2.3 SSE 事件处理流程

```typescript
// hooks/useEvalSSE.ts
// 1. 连接 GET /api/eval/events
// 2. 收到事件后按 type 更新 state：
//    eval_start  → 初始化 cases 列表，状态改为 running
//    case_start  → 当前 case 高亮，创建 A/B panel
//    llm_call    → 按 label(A/B) 追加到对应 panel
//    tool_exec   → 按 label(A/B) 追加 tool 调用卡片
//    finish      → 标记对应 agent 完成
//    case_done   → 更新进度条，标记 case 完成
//    scoring_done → 显示评分
//    eval_done   → 状态改为 completed，显示报告链接
```

### 2.4 依赖选择

| 用途 | 库 | 说明 |
|------|-----|------|
| 路由 | react-router-dom | 页面切换 |
| UI | tailwindcss | 轻量，不引入组件库 |
| 图表 | recharts | 评分对比柱状图 |
| 代码高亮 | react-syntax-highlighter | 产物文件预览 |

---

## 三、文件改动清单

### 新增文件

| 文件 | 说明 |
|------|------|
| `server/server.go` | HTTP 服务主逻辑，路由注册 |
| `server/sse.go` | SSE 广播器 |
| `server/handler.go` | API handler 实现 |
| `server/report.go` | 静态 HTML 报告生成 |
| `server/template.go` | HTML 报告模板 |
| `web/` 目录 | 完整 React 前端项目 |

### 修改文件

| 文件 | 改动 | 幅度 |
|------|------|------|
| `main.go` | 增加 `--serve`/`--port` 参数，条件分支启动 HTTP 服务 | 中 (~40 行) |
| `eval/runner.go` | 在 Run 循环中增加阶段性事件 Emit | 小 (~20 行) |
| `agent/run_context.go` | 新增 `EventCaseStart` 等事件类型常量 | 小 (~10 行) |
| `go.mod` | 无新外部依赖 | 无 |

### 不改动的文件

- `agent/orchestrator.go` — 内层循环不变
- `agent/types.go` — 数据结构不变
- `eval/scorer.go` — 评分逻辑不变，只在外层 Runner 包装事件
- `providers/openai.go` — 不动
- `tool/*` — 不动

---

## 四、实施步骤（建议顺序）

### Phase 1：后端 HTTP + SSE 层
1. 新增 `server/sse.go` — SSE Broker 实现
2. 新增 `server/handler.go` — API handlers
3. 新增 `server/server.go` — 路由和服务启动
4. 改造 `agent/run_context.go` — 增加新事件类型
5. 改造 `eval/runner.go` — 增加阶段性事件
6. 改造 `main.go` — 增加 `--serve` 模式

### Phase 2：React 前端
7. 初始化 `web/` 项目（Vite + React + TypeScript + Tailwind）
8. 实现 SSE hook 和 API client
9. 实现 Dashboard 页面（配置面板 + 实时执行视图）
10. 实现报告列表和详情页面

### Phase 3：静态 HTML 报告
11. 新增 `server/report.go` + `server/template.go` — HTML 报告生成
12. 在评测结束后自动调用生成

### Phase 4：联调和优化
13. Vite dev proxy 配置，前后端联调
14. 生产部署：`go:embed` 嵌入前端构建产物（可选）

---

## 五、验证方式

1. **后端 API 验证**：`curl POST /api/eval/start` 启动评测，`curl GET /api/eval/events` 收到 SSE 事件流
2. **实时 Dashboard 验证**：浏览器打开 Dashboard，点击开始评测，观察 A/B Agent 面板实时更新
3. **CLI 兼容性验证**：不带 `--serve` 运行原有 CLI 命令，行为不变
4. **静态报告验证**：评测完成后 `{outputDir}/{timestamp}/report.html` 存在，浏览器打开可正常浏览
5. **历史报告验证**：Web 页面报告列表能列出所有历史评测，点击可查看详情
