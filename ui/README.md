# Skill Eval 前端

评测台。打开一条记录就能看到结果（通过 / 未通过、分数、产物）和执行过程（时间条、步骤、日志）。

技术栈沿用原来的 React + TypeScript + Vite。界面不再套 Ant Design 的后台布局，Ant Design 只留下主题容器。

## 预览

```bash
cd ui
pnpm install
pnpm dev
```

锁文件里的 tarball 指向内网源。内网不可达时，可以用公共源安装，不必改锁文件：

```bash
npm install --registry https://registry.npmjs.org --no-package-lock
```

浏览器打开 Vite 打印的本地地址。生产构建：

```bash
pnpm build
pnpm preview
```

`pnpm dev` 和 `pnpm preview` 都会从仓库的 `logs/traces/*.json` 读真实 Trace（DefaultTracer 落下的文件）。如果这个本地读取不可用，页面退回到构建时打包进去的 Trace。

## 执行后端

侧边栏可以按执行后端筛选：

| 标识 | 含义 | 现在有没有数据 |
| --- | --- | --- |
| `kernel` | 自研内核 `agent.Orchestrator` | 有。`logs/traces` 里的真实运行。旧文件没有 `runner` 字段，归到这一类并标成「未标注」。 |
| `codex` | Codex | 没有执行器。列表里那条是界面样例。 |
| `claude-code` | Claude Code | 没有执行器。列表里那条是界面样例。 |

新的自研内核运行会在 Trace 里写 `runner: "kernel"`。样例可以用侧边栏的「样例」开关隐藏。

## 评分怎么来的

仓库没有评测 HTTP API，`eval.Exec` 只把报告打到标准输出。页面因此分成三种分数：

- **Trace**：按现有规则从 Span 算出来。Skill 是否命中对齐 `SkillHitScorer`（看 `use_skill` 的 `name`）。没有产物时对齐 `ArtifactScorer` 的「未产出任何文件」。是否执行成功看 `Trace.success`，因为 `SuccessScorer` 对单条 Trace 目前返回空结果。
- **未落盘**：有产物时的质量分，以及执行过程分。这两项要再调模型，页面不编造。
- **样例**：只出现在 Codex / Claude Code 样例上，用来把评分卡的样子先摆出来。

如果以后把 `eval.Report` 写到 `logs/reports/*.json`（字段与 Go 结构体一致，至少要有 `traceId` 和 `scores`），刷新后会直接用那份报告。

耗时用 Span 的 `startTime` / `endTime`。历史文件里的 `duration` 有的是纳秒。

更完整的说明在页面左下角「数据从哪来」。
