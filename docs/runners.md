# 执行后端

一次评测的输入是 `runner.Task`：用户任务、目标 skill、模型、技能目录、迭代上限和超时。三个后端都返回 `agent.Trace`，再用同一组评分器生成 `eval.Report`。

`trace.runner` 取值：

- `self`：自研内核
- `codex`：Codex CLI
- `claude-code`：Claude Code CLI

`trace.status` 取值：`running`、`succeeded`、`failed`、`timeout`、`cancelled`。`succeeded` 只表示执行过程正常结束。skill 是否命中、产物是否合格看 `report.scores`，每条评分有 `source`。

## 环境变量

| 变量 | 用途 |
| --- | --- |
| `EVAL_API_KEY` | 自研内核，以及可选的大模型裁判 |
| `EVAL_BASE_URL` | OpenAI 兼容接口地址 |
| `EVAL_JUDGE_MODEL` | 裁判模型，默认 `kimi-2.5`。未设置 `EVAL_API_KEY` 时不调用裁判 |
| `EVAL_DEFAULT_SKILL_DIR` | 默认技能根目录 |
| `EVAL_DATA_DIR` | 评测记录和运行目录，默认 `logs` |
| `EVAL_HTTP_ADDR` | `serve` 的监听地址，默认 `:8080` |
| `EVAL_LLM_RETRIES` | 自研 loop 调用模型的最大次数，默认 3 |
| `EVAL_WORKPLACE` | 未指定当次工作区时，自研工具使用的根目录 |
| `CODEX_BIN` | Codex 可执行文件，默认在 `PATH` 中找 `codex` |
| `CODEX_SANDBOX` | 传给 `codex exec --sandbox`，默认 `workspace-write` |
| `CODEX_APPROVAL` | 传给 `--ask-for-approval`，默认 `never` |
| `CLAUDE_BIN` | Claude Code 可执行文件，默认在 `PATH` 中找 `claude` |
| `CLAUDE_PERMISSION_MODE` | 传给 `--permission-mode`，默认 `bypassPermissions` |

Codex 的登录态使用进程里的 `OPENAI_API_KEY`，或本机已有的 Codex 登录。Claude Code 使用 `ANTHROPIC_API_KEY`，或本机已有的 Claude 登录。这些值不会写入 trace、日志参数或仓库。

## self

自研内核不需要额外 CLI。每次运行会建一个目录：

```text
$EVAL_DATA_DIR/runs/<id>/workspace
$EVAL_DATA_DIR/runs/<id>/logs
```

技能被复制到 `workspace/.skill-eval/skills`，bash 和文件工具共用这个工作区。文件工具拒绝写出工作区。bash 只保证工作目录在工作区内，它不是安全沙箱。

模型调用失败且错误像是限流或临时网络问题时，loop 会按 200ms、400ms、800ms 重试，最多 3 次。重试记在 `trace.notes`。达到最大迭代、连续两次不调用工具、超时或取消时，`trace.failureReason` 和返回错误都会写明原因。`trace.spans` 是链路，`trace.steps` 是同一条链路的短摘要。

完成任务必须调用 `finish`。目标 skill 要通过 `use_skill`，命中后对应 span 的 `isTarget` 为 true。

## codex

安装 Codex CLI，并保证下面之一成立：

```bash
codex --version
# 或
export CODEX_BIN=/absolute/path/to/codex
```

找不到二进制文件时，错误类似：

```text
未找到 codex 可执行文件。请安装 CLI 并加入 PATH，或设置 CODEX_BIN。参见 docs/runners.md
```

实际执行的是：

```bash
codex exec --json --skip-git-repo-check \
  --sandbox workspace-write \
  --ask-for-approval never \
  -C <workspace> \
  --model <model> \
  --output-last-message <log>/last-message.txt \
  "<prompt>"
```

工作区会先 `git init`，因为 Codex 要求在 git 仓库里运行。技能复制到 `<workspace>/.agents/skills/<name>/SKILL.md`。提示词里也会给出这份 `SKILL.md` 的绝对路径，即使自动发现失败，事件流里仍能看到它有没有去读这个文件。

JSONL 里的 `command_execution`、`file_change`、`agent_message` 会收成 `tool_call` / `llm_call` span。读到目标 skill 路径时 `isTarget=true`。最终文本和新增文件写入 `artifactsAndResult`，格式与自研 `finish` 工具一致：

```json
{"result":"...","artifacts":["out.txt"]}
```

`workspace-write` 把 Codex 的写操作限制在工作区。不要把密钥放进 prompt。

## claude-code

```bash
claude --version
# 或
export CLAUDE_BIN=/absolute/path/to/claude
```

执行命令：

```bash
claude -p \
  --output-format stream-json \
  --verbose \
  --permission-mode bypassPermissions \
  --no-session-persistence \
  --add-dir <workspace> \
  --model <model>
```

任务从标准输入送入。技能复制到 `<workspace>/.claude/skills/<name>/SKILL.md`。`Skill` 工具调用和 skill 路径会映射成 `isTarget`。`result` 事件里的 `subtype=success` 视为这次执行结束；`is_error` 或带 error 的 subtype 写入 `failureReason`。

`bypassPermissions` 是为了让无交互评测能调用工具。这个模式可以碰到工作区以外的文件，不应当作安全沙箱。需要更硬的隔离时，把整个 `skill-eval` 放到容器里再跑。

同时会设置 `CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC=1`，减少 CLI 的额外网络行为。不会改写 `CLAUDE_CONFIG_DIR`，避免弄丢本机登录态。

## 本地没有 CLI 时怎么验证

`go test ./...` 用录好的 JSONL 检查 Codex 和 Claude Code 的解析、参数和缺二进制时的错误。安装 CLI 之后，用同一条 `skill-eval run --runner codex|claude-code` 做真实调用。原始 stdout / stderr 在 `trace.runnerMeta.logPath`。
