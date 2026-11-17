# Skill Eval UI

基于 React + Vite + shadcn/ui（Radix + Tailwind）的评测控制台。

## 开发

先启动后端：

```bash
go run . serve --addr :8080
```

再启动前端：

```bash
cd ui
pnpm install
pnpm dev
```

Vite 会把 `/api` 代理到 `http://127.0.0.1:8080`。

## 页面

- 总览：执行后端可用性、最近运行
- 运行详情：评分、Span 过程、产物、日志
- 对比：两条评测的确定性对比
- 新建评测：提交到 self / Codex / Claude Code
