# skill-eval

对 skill 做可复核评测的后端。同一次任务可以跑在自研 agent loop、Codex CLI 或 Claude Code CLI 上，trace、评分和失败原因使用同一套结构。

前端仍读取原来的 `Trace` / `Report` / `EvalRecord` 字段。本仓库新增的 `runner`、`status`、`failureReason`、`runnerMeta`、`source` 都是可选字段，旧界面可以忽略。

## 跑一次评测

```bash
export EVAL_API_KEY=...
export EVAL_BASE_URL=https://your-openai-compatible-host/v1
export EVAL_DEFAULT_SKILL_DIR=/path/to/skills

go run . run \
  --runner self \
  --model glm-5 \
  --skill pdf \
  --skill-dir "$EVAL_DEFAULT_SKILL_DIR" \
  --prompt "把一句问候写成 pdf，并调用 finish"
```

技能目录的布局是 `<skill-dir>/<name>/SKILL.md`。`self` 必须提供 `--model`、`EVAL_API_KEY` 和 `EVAL_BASE_URL`。

查看本机有哪些执行后端：

```bash
go run . runners
```

启动 API：

```bash
go run . serve --addr :8080
```

对比两次已经完成的评测：

```bash
go run . compare <evalIdA> <evalIdB>
```

`go run . demo` 保留原来的天气示例，同样依赖 `EVAL_API_KEY` 和 `EVAL_BASE_URL`。

## 执行后端

| runner | 说明 |
| --- | --- |
| `self` | 仓库里的自研 agent loop |
| `codex` | `codex exec --json` |
| `claude-code` | `claude -p --output-format stream-json` |

安装、环境变量和隔离方式见 [docs/runners.md](docs/runners.md)。HTTP 字段见 [docs/api.md](docs/api.md)。

缺 CLI 或密钥时，命令和 `GET /api/runners` 会返回中文原因，并指向文档。密钥只从环境变量读取，不会写进仓库或 trace。

## 测试

```bash
go test ./...
```

自研内核的 loop、Codex / Claude Code 的事件解析，以及 HTTP 评测接口都有不访问外网的测试。外部 CLI 本身需要本机安装后才能做真实调用。

## 目录

- `agent/` 自研 loop、trace、重试
- `runner/` `self`、`codex`、`claude-code`
- `eval/` 规则评分、可选的大模型裁判、对比
- `server/` 评测 API 和记录存储
- `tool/` 自研内核使用的工具，文件工具限制在当次工作区
