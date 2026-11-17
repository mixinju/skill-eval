import { useCallback, useEffect, useMemo, useState } from 'react'
import { CompareView } from '@/components/CompareView'
import { Overview } from '@/components/Overview'
import { RunDetail } from '@/components/RunDetail'
import { Sidebar } from '@/components/Sidebar'
import { TooltipProvider } from '@/components/ui/tooltip'
import { compareEvals, fetchEvals, fetchHealth, fetchRunners } from '@/lib/api'
import type { EvalRecord, RunnerAvailability } from '@/types'

type View =
  | { kind: 'overview' }
  | { kind: 'detail'; id: string }
  | { kind: 'compare'; a: string; b: string; reportId?: string }

function readHash(): View {
  const raw = window.location.hash.replace(/^#/, '')
  if (!raw || raw === '/') return { kind: 'overview' }
  const parts = raw.split('/').map((part) => {
    try {
      return decodeURIComponent(part)
    } catch {
      return part
    }
  })
  if (parts[0] === 'run' && parts[1]) return { kind: 'detail', id: parts[1] }
  if (parts[0] === 'compare' && parts[1] && parts[2]) return { kind: 'compare', a: parts[1], b: parts[2] }
  return { kind: 'overview' }
}

function hashFor(view: View): string {
  if (view.kind === 'overview') return '#/'
  if (view.kind === 'detail') return `#run/${encodeURIComponent(view.id)}`
  return `#compare/${encodeURIComponent(view.a)}/${encodeURIComponent(view.b)}`
}

export default function App() {
  const [records, setRecords] = useState<EvalRecord[]>([])
  const [runners, setRunners] = useState<RunnerAvailability[]>([])
  const [apiOnline, setApiOnline] = useState(false)
  const [loading, setLoading] = useState(true)
  const [note, setNote] = useState<string>()
  const [view, setView] = useState<View>(() => readHash())
  const [tray, setTray] = useState<string[]>([])
  const [compareRec, setCompareRec] = useState<EvalRecord>()

  const refresh = useCallback(async () => {
    setLoading(true)
    const online = await fetchHealth()
    setApiOnline(online)
    if (!online) {
      setNote('无法连接评测 API。请先运行 go run . serve --addr :8080')
      setLoading(false)
      return
    }
    try {
      const [list, runnerList] = await Promise.all([fetchEvals(), fetchRunners()])
      setRecords(list)
      setRunners(runnerList)
      setNote(list.length ? undefined : 'API 在线，但还没有评测记录。')
    } catch (err) {
      setNote(err instanceof Error ? err.message : '加载评测列表失败')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    void refresh()
  }, [refresh])

  useEffect(() => {
    const next = hashFor(view)
    if (window.location.hash !== next) window.history.replaceState(null, '', next)
  }, [view])

  useEffect(() => {
    const onHash = () => setView(readHash())
    window.addEventListener('hashchange', onHash)
    return () => window.removeEventListener('hashchange', onHash)
  }, [])

  useEffect(() => {
    if (view.kind !== 'compare') {
      setCompareRec(undefined)
      return
    }
    let cancelled = false
    compareEvals(view.a, view.b)
      .then((rec) => {
        if (!cancelled) {
          setCompareRec(rec)
          setRecords((prev) => (prev.some((item) => item.id === rec.id) ? prev : [rec, ...prev]))
        }
      })
      .catch((err) => {
        if (!cancelled) setNote(err instanceof Error ? err.message : '对比失败')
      })
    return () => {
      cancelled = true
    }
  }, [view])

  const byId = useMemo(() => new Map(records.map((item) => [item.id, item])), [records])
  const selectedId = view.kind === 'detail' ? view.id : undefined

  const onToggleTray = (id: string) => {
    setTray((prev) => {
      if (prev.includes(id)) return prev.filter((item) => item !== id)
      return prev.length >= 2 ? [prev[1], id] : [...prev, id]
    })
  }

  const onCreated = (rec: EvalRecord) => {
    setRecords((prev) => [rec, ...prev.filter((item) => item.id !== rec.id)])
    setView({ kind: 'detail', id: rec.id })
    const poll = window.setInterval(() => {
      void refresh()
    }, 2000)
    window.setTimeout(() => window.clearInterval(poll), 30_000)
  }

  return (
    <TooltipProvider>
      <div className="grid h-full grid-cols-1 md:grid-cols-[320px_minmax(0,1fr)]">
        <Sidebar
          records={records}
          selectedId={selectedId}
          tray={tray}
          runners={runners}
          loading={loading}
          apiOnline={apiOnline}
          note={note}
          onSelect={(id) => setView({ kind: 'detail', id })}
          onHome={() => setView({ kind: 'overview' })}
          onRefresh={() => void refresh()}
          onToggleTray={onToggleTray}
          onOpenCompare={() => {
            if (tray.length === 2) setView({ kind: 'compare', a: tray[0], b: tray[1] })
          }}
          onCreated={onCreated}
        />
        <main className="min-h-0 overflow-auto">
          {view.kind === 'overview' && (
            <Overview
              records={records}
              runners={runners}
              onOpen={(id) => setView({ kind: 'detail', id })}
              onFilterRunner={() => setView({ kind: 'overview' })}
            />
          )}
          {view.kind === 'detail' && (
            <RunDetail
              recordId={view.id}
              seed={byId.get(view.id)}
              inTray={tray.includes(view.id)}
              onToggleTray={() => onToggleTray(view.id)}
              onLoaded={(rec) =>
                setRecords((prev) => {
                  const next = prev.filter((item) => item.id !== rec.id)
                  return [rec, ...next]
                })
              }
            />
          )}
          {view.kind === 'compare' && (
            <CompareView
              left={byId.get(view.a)}
              right={byId.get(view.b)}
              compare={compareRec?.compareReport}
            />
          )}
        </main>
      </div>
    </TooltipProvider>
  )
}
