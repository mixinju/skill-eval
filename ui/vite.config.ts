import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import type { Connect, Plugin } from 'vite'
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

function readJsonDir(dir: string): { file: string; data?: unknown; error?: string }[] {
  if (!fs.existsSync(dir)) return []
  return fs
    .readdirSync(dir)
    .filter((name) => name.endsWith('.json'))
    .sort()
    .map((file) => {
      try {
        const data = JSON.parse(fs.readFileSync(path.join(dir, file), 'utf8')) as unknown
        return { file, data }
      } catch {
        return { file, error: '无法解析 JSON' }
      }
    })
}

/** 读取 DefaultTracer 落下的文件。这不是 Go 后端的 HTTP API。 */
function localEvalPlugin(): Plugin {
  const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..')
  const handler: Connect.NextHandleFunction = (req, res, next) => {
    const url = req.url?.split('?')[0]
    if (url !== '/__local/evals') {
      next()
      return
    }
    const body = JSON.stringify({
      traces: readJsonDir(path.join(root, 'logs', 'traces')),
      reports: readJsonDir(path.join(root, 'logs', 'reports')),
      note: 'Vite 本地适配层，读取 logs/traces 与可选的 logs/reports。',
    })
    res.statusCode = 200
    res.setHeader('Content-Type', 'application/json; charset=utf-8')
    res.setHeader('Cache-Control', 'no-store')
    res.end(body)
  }
  return {
    name: 'local-eval-files',
    configureServer(server) {
      server.middlewares.use(handler)
    },
    configurePreviewServer(server) {
      server.middlewares.use(handler)
    },
  }
}

export default defineConfig({
  plugins: [react(), localEvalPlugin()],
  server: {
    host: '0.0.0.0',
    port: 5173,
  },
  preview: {
    host: '0.0.0.0',
    port: 4173,
  },
})
