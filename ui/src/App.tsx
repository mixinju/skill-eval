import { useEffect, useMemo, useState } from 'react'
import { ConfigProvider, theme as antdTheme } from 'antd'
import zhCN from 'antd/locale/zh_CN'
import type { EvalRecord } from './types'
import { fixtureRecords } from './lib/fixtures'
import { loadTraces } from './lib/loadEvals'
import CompareView from './components/CompareView'
import DetailView from './components/DetailView'
import GapsDialog from './components/GapsDialog'
import Landing from './components/Landing'
import Sidebar from './components/Sidebar'

type ThemeName = 'dark' | 'light'
type View =
  | { kind: 'landing' }
  | { kind: 'detail'; id: string }
  | { kind: 'compare'; a: string; b: string }

function readHash(): View | 'default' {
  const raw = window.location.hash.replace(/^#/, '')
  if (raw === '' ) return 'default'
  if (raw === '/') return { kind: 'landing' }
  const parts = raw.split('/').map((part) => {
    try {
      return decodeURIComponent(part)
    } catch {
      return part
    }
  })
  if (parts[0] === 'trace' && parts[1]) return { kind: 'detail', id: parts[1] }
  if (parts[0] === 'compare' && parts[1] && parts[2]) return { kind: 'compare', a: parts[1], b: parts[2] }
  return 'default'
}

function hashFor(view: View): string {
  if (view.kind === 'landing') return '#/'
  if (view.kind === 'detail') return `#trace/${encodeURIComponent(view.id)}`
  return `#compare/${encodeURIComponent(view.a)}/${encodeURIComponent(view.b)}`
}

function findRecord(records: EvalRecord[], id: string): EvalRecord | undefined {
  return records.find((record) => record.id === id)
}

export default function App() {
  const [theme, setTheme] = useState<ThemeName>(() => {
    const saved = localStorage.getItem('skill-eval-theme')
    return saved === 'light' ? 'light' : 'dark'
  })
  const [live, setLive] = useState<EvalRecord[]>([])
  const [note, setNote] = useState<string | undefined>('正在读取 logs/traces…')
  const [loading, setLoading] = useState(true)
  const [view, setView] = useState<View | null>(null)
  const [tray, setTray] = useState<string[]>([])
  const [gapsOpen, setGapsOpen] = useState(false)

  const records = useMemo(() => [...live, ...fixtureRecords], [live])

  useEffect(() => {
    document.documentElement.dataset.theme = theme
    localStorage.setItem('skill-eval-theme', theme)
  }, [theme])

  useEffect(() => {
    let cancelled = false
    loadTraces().then((result) => {
      if (cancelled) return
      setLive(result.records)
      const skipped = result.skipped.length ? `有 ${result.skipped.length} 个文件没能解析。` : ''
      setNote([result.note, skipped].filter(Boolean).join(' ') || undefined)
      setLoading(false)
      setView((current) => {
        const all = [...result.records, ...fixtureRecords]
        const initial = current ?? readHash()
        if (initial !== 'default' && initial.kind === 'detail' && findRecord(all, initial.id)) return initial
        if (initial !== 'default' && initial.kind === 'compare' && findRecord(all, initial.a) && findRecord(all, initial.b)) return initial
        if (initial !== 'default' && initial.kind === 'landing') return initial
        if (result.records[0]) return { kind: 'detail', id: result.records[0].id }
        return { kind: 'landing' }
      })
    })
    return () => {
      cancelled = true
    }
  }, [])

  useEffect(() => {
    if (!view) return
    const next = hashFor(view)
    if (window.location.hash !== next) window.history.replaceState(null, '', next)
  }, [view])

  useEffect(() => {
    const onHash = () => {
      const next = readHash()
      if (next !== 'default') setView(next)
    }
    window.addEventListener('hashchange', onHash)
    return () => window.removeEventListener('hashchange', onHash)
  }, [])

  const toggleTray = (id: string) => {
    if (tray.includes(id)) {
      setTray(tray.filter((item) => item !== id))
      return
    }
    const next = tray.length >= 2 ? [tray[1], id] : [...tray, id]
    setTray(next)
    if (next.length === 2) setView({ kind: 'compare', a: next[0], b: next[1] })
  }

  const selectedId = view?.kind === 'detail' ? view.id : undefined
  const active = view?.kind === 'detail' ? findRecord(records, view.id) : undefined
  const compareA = view?.kind === 'compare' ? findRecord(records, view.a) : undefined
  const compareB = view?.kind === 'compare' ? findRecord(records, view.b) : undefined

  return (
    <ConfigProvider
      locale={zhCN}
      theme={{
        algorithm: theme === 'dark' ? antdTheme.darkAlgorithm : antdTheme.defaultAlgorithm,
        token: {
          colorPrimary: theme === 'dark' ? '#3dbea5' : '#0f766e',
          borderRadius: 10,
          fontFamily: 'inherit',
        },
      }}
    >
      <div className="se-shell">
        <Sidebar
          records={records}
          selectedId={selectedId}
          tray={tray}
          loading={loading}
          note={note}
          theme={theme}
          onSelect={(id) => setView({ kind: 'detail', id })}
          onToggleTray={toggleTray}
          onHome={() => setView({ kind: 'landing' })}
          onOpenCompare={() => {
            if (tray.length === 2) setView({ kind: 'compare', a: tray[0], b: tray[1] })
          }}
          onClearTray={() => setTray([])}
          onToggleTheme={() => setTheme((value) => (value === 'dark' ? 'light' : 'dark'))}
          onOpenGaps={() => setGapsOpen(true)}
        />
        <main className="se-main">
          {loading || !view ? (
            <p className="se-boot">正在读取 logs/traces…</p>
          ) : view.kind === 'landing' ? (
            <Landing
              records={records}
              onOpen={(id) => setView({ kind: 'detail', id })}
              onRunner={(id) => {
                const match = records.find((record) => record.runner === id && record.source === 'trace-file')
                  ?? records.find((record) => record.runner === id)
                if (match) setView({ kind: 'detail', id: match.id })
              }}
            />
          ) : view.kind === 'compare' ? (
            compareA && compareB ? (
              <CompareView
                a={compareA}
                b={compareB}
                onClose={() => setView({ kind: 'detail', id: compareA.id })}
              />
            ) : (
              <p className="se-boot">对比的记录不在当前列表里。</p>
            )
          ) : active ? (
            <DetailView record={active} inTray={tray.includes(active.id)} onToggleTray={() => toggleTray(active.id)} />
          ) : (
            <p className="se-boot">这条记录不在当前列表里。</p>
          )}
        </main>
        <GapsDialog open={gapsOpen} onClose={() => setGapsOpen(false)} />
      </div>
    </ConfigProvider>
  )
}
