# 评测 API

`skill-eval serve` 提供这些接口。响应是 JSON。跨域允许本地前端直接调用。

记录落在 `$EVAL_DATA_DIR/evals/<id>.json`。单次记录的字段兼容前端的 `EvalRecord`：`id`、`type`、`createdAt`、`report`、`trace`、`compareReport`。下面列出的是追加字段，旧前端可以忽略。

## 字段

### Record

| 字段 | 说明 |
| --- | --- |
| `type` | `single` 或 `compare` |
| `status` | `running`、`succeeded`、`failed`、`timeout`、`cancelled` |
| `runner` | `self`、`codex`、`claude-code` |
| `error` | 执行器或基础设施错误。评分失败另记在 `scores[].error` |
| `logPath` | stdout 日志路径 |
| `report` / `trace` | 与现有前端类型一致，并带下面的可选字段 |

### Trace

原有字段保持不变：`id`、`agentName`、`model`、`userPrompt`、`targetSkill`、`startTime`、`endTime`、`totalTokens`、`iterations`、`success`、`spans`。

| 追加字段 | 说明 |
| --- | --- |
| `schemaVersion` | 当前为 `1` |
| `runner` | 执行后端 |
| `runnerMeta` | `type`、`displayName`、`binary`、`version`、`model`、`isolated`、`skillDir`、`command`、`logPath`。没有密钥 |
| `status` | 同记录状态 |
| `failureReason` / `error` | 这次运行为什么没有成功 |
| `workspace` | 隔离工作区 |
| `skillPath` | 实际评分用的 `SKILL.md` |
| `externalId` | Codex thread id 或 Claude session id |
| `notes` | 重试、stderr、非 JSON 输出 |
| `steps` | 由 span 压缩出的步骤 |
| `artifactsAndResult` | `{"result","artifacts"}`，三个 runner 都用这个形状 |

`spans[].kind` 仍然只使用前端已经识别的 `llm_call`、`tool_call`、`llm_compress`。目标 skill 用 `isTarget`。

### Report 与 Verdict

Report 追加 `runner`、`runnerMeta`、`status`、`failureReason`、`workspace`、`skillPath`。

Verdict 追加：

| 字段 | 说明 |
| --- | --- |
| `source` | `deterministic` 规则、`static` 静态检查、`llm` 模型裁判 |
| `error` | 评分器自己的失败 |

默认评分器：

1. 是否执行成功，`source=deterministic`，失败原因来自 `failureReason`
2. Skill 是否命中，看 `isTarget` 或 `use_skill` / `Skill` 的名称参数
3. 执行过程(规则)，迭代、工具错误、命中和是否完成的平均值，`>= 0.6` 通过
4. Skill 静态检查，没有 `SKILL.md` 时跳过并记明原因
5. 仅当设置了 `EVAL_API_KEY`：执行过程、产物评分，`source=llm`

`report.pass` 是所有评分都通过。它和 `status=succeeded` 不是一回事。

## 接口

### `GET /api/health`

```json
{"status":"ok"}
```

### `GET /api/runners`

```json
{
  "runners": [
    {"type":"self","name":"自研内核","available":false,"detail":"自研内核已内置，但缺少环境变量: EVAL_API_KEY, EVAL_BASE_URL。参见 docs/runners.md"}
  ]
}
```

### `POST /api/evals`

```json
{
  "runner": "self",
  "name": "pdf-eval",
  "systemPrompt": "完成任务后调用 finish",
  "userPrompt": "把一句问候写成 pdf",
  "model": "glm-5",
  "targetSkill": "pdf",
  "skillDir": "/path/to/skills",
  "maxIterations": 10,
  "timeoutSec": 900,
  "wait": true
}
```

`userPrompt` 必填。`self` 还要求 `model`。`wait=true`（或查询参数 `?wait=true`）等待结束后返回完整记录，HTTP 200。否则 HTTP 202，先返回 `status=running` 的记录，再用 `GET /api/evals/{id}` 取结果。

执行失败但已经产出 trace 时，响应体里仍有 `report` 和 `trace`，用 `status`、`error`、`failureReason` 区分原因。

### `GET /api/evals`

```json
{"records": []}
```

列表里的 trace 不带 span 正文，避免把大 trace 一次拉完。`GET /api/evals/{id}` 返回完整记录。

### `GET /api/evals/{id}/trace`

### `GET /api/evals/{id}/report`

### `GET /api/evals/{id}/logs`

stdout 纯文本。没有日志时返回 404 和 `error`。

### `POST /api/evals/compare`

```json
{"evalIdA":"<id>","evalIdB":"<id>"}
```

两边都结束后才对比，否则 409。对比不调用大模型。`compareReport` 的形状与前端 `CompareReport` 一致：`traceA`、`traceB`、`reportA`、`reportB`、`scores`、`conclusion`。分数来自迭代、Token、工具错误和已有产物分，`source=deterministic`。

## 例子

```bash
curl -s localhost:8080/api/runners

curl -s -X POST localhost:8080/api/evals \
  -H 'content-type: application/json' \
  -d '{"runner":"self","model":"glm-5","targetSkill":"pdf","skillDir":"/path/to/skills","userPrompt":"写成 pdf","wait":true}'
```

命令行等价入口是 `skill-eval run` 和 `skill-eval compare`。
