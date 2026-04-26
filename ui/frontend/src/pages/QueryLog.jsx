import { useState, useEffect, useRef, useMemo } from 'react'
import { Pause, Play, Download, Search, X } from 'lucide-react'

const CHIP = {
  RESOLVED: 'chip-good',
  CACHED:   'chip-violet',
  BLOCKED:  'chip-bad',
  ERROR:    'chip-warn',
}

const FILTERS = ['ALL', 'RESOLVED', 'CACHED', 'BLOCKED', 'ERROR']

function ts(iso) {
  const d = new Date(iso)
  const hh = String(d.getHours()).padStart(2, '0')
  const mm = String(d.getMinutes()).padStart(2, '0')
  const ss = String(d.getSeconds()).padStart(2, '0')
  const ms = String(d.getMilliseconds()).padStart(3, '0')
  return `${hh}:${mm}:${ss}.${ms}`
}

export default function QueryLog({ api }) {
  const [queries, setQueries]       = useState([])
  const [paused, setPaused]         = useState(false)
  const [filter, setFilter]         = useState('')
  const [typeFilter, setTypeFilter] = useState('ALL')
  const pausedRef = useRef(false)

  useEffect(() => { pausedRef.current = paused }, [paused])

  useEffect(() => {
    const poll = async () => {
      if (pausedRef.current) return
      try {
        const data = await api.get('/queries')
        setQueries(data ?? [])
      } catch {}
    }
    poll()
    const id = setInterval(poll, 500)
    return () => clearInterval(id)
  }, [api])

  const visible = useMemo(() => queries.filter(q => {
    const dm = !filter || q.domain.toLowerCase().includes(filter.toLowerCase())
    const tm = typeFilter === 'ALL' || q.result === typeFilter
    return dm && tm
  }), [queries, filter, typeFilter])

  const counts = useMemo(() => {
    const c = { ALL: queries.length, RESOLVED: 0, CACHED: 0, BLOCKED: 0, ERROR: 0 }
    for (const q of queries) c[q.result] = (c[q.result] || 0) + 1
    return c
  }, [queries])

  const exportCSV = () => {
    const header = 'Timestamp,Domain,Type,Result,Latency(ms),Upstream\n'
    const rows = visible.map(q =>
      `${q.timestamp},${q.domain},${q.type},${q.result},${q.latency_ms.toFixed(2)},${q.upstream}`
    ).join('\n')
    const blob = new Blob([header + rows], { type: 'text/csv' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url; a.download = 'selfdns-queries.csv'; a.click()
    URL.revokeObjectURL(url)
  }

  return (
    <div className="flex flex-col h-full px-7 py-6 gap-4">
      <header className="flex items-end justify-between pt-2">
        <div>
          <h1 className="h-page">Query Log</h1>
          <p className="mt-1 text-xs text-text-muted">
            <span className="font-mono">{visible.length}</span> shown
            <span className="mx-2 text-text-dim">·</span>
            <span className="font-mono">live · 500ms</span>
          </p>
        </div>

        <div className="flex items-center gap-2">
          <button className="btn-ghost" onClick={exportCSV}>
            <Download size={12} /> Export CSV
          </button>
          <button
            className={paused ? 'btn-primary' : 'btn-ghost'}
            onClick={() => setPaused(p => !p)}
          >
            {paused
              ? <><Play size={12} /> Resume</>
              : <><Pause size={12} /> Pause</>}
          </button>
        </div>
      </header>

      <div className="flex gap-2.5 items-center">
        <div className="relative flex-1">
          <Search size={12} className="absolute left-2.5 top-1/2 -translate-y-1/2 text-text-dim" />
          <input
            className="input pl-7 h-9"
            placeholder="filter by domain…"
            value={filter}
            onChange={e => setFilter(e.target.value)}
          />
          {filter && (
            <button
              className="absolute right-2.5 top-1/2 -translate-y-1/2 text-text-dim hover:text-text-primary"
              onClick={() => setFilter('')}
            >
              <X size={12} />
            </button>
          )}
        </div>

        <div className="flex items-center gap-1 p-1 panel">
          {FILTERS.map(t => (
            <button
              key={t}
              onClick={() => setTypeFilter(t)}
              className={`px-2 h-6 rounded text-2xs font-semibold uppercase tracking-widest transition-colors
                          ${typeFilter === t
                            ? 'bg-ink-400 text-text-primary'
                            : 'text-text-muted hover:text-text-secondary hover:bg-ink-300/60'}`}
            >
              {t}
              <span className="ml-1.5 font-mono text-text-dim">{counts[t] ?? 0}</span>
            </button>
          ))}
        </div>
      </div>

      <div className="flex-1 overflow-hidden panel-flat flex flex-col">
        <div className="flex-1 overflow-auto">
          <table className="table-base">
            <thead>
              <tr>
                <th className="w-[120px]">Time</th>
                <th>Domain</th>
                <th className="w-[60px]">Type</th>
                <th className="w-[100px]">Result</th>
                <th className="w-[80px] !text-right">Latency</th>
                <th className="w-[180px]">Upstream</th>
              </tr>
            </thead>
            <tbody>
              {visible.length === 0 && (
                <tr>
                  <td colSpan={6} className="!h-32 text-center !text-text-dim font-mono">
                    {paused ? '— paused —' : 'waiting for queries…'}
                  </td>
                </tr>
              )}
              {visible.map((q, i) => {
                const lat = q.latency_ms
                const latColor = lat === 0 ? 'text-text-dim'
                  : lat < 30 ? 'text-accent'
                  : lat < 80 ? 'text-warn'
                  : 'text-bad'
                return (
                  <tr key={i}>
                    <td className="font-mono text-text-muted">{ts(q.timestamp)}</td>
                    <td className="font-mono text-text-primary truncate max-w-[280px]">
                      {q.domain}
                    </td>
                    <td>
                      <span className="text-2xs font-mono text-text-muted">{q.type}</span>
                    </td>
                    <td>
                      <span className={CHIP[q.result] ?? 'chip-mute'}>{q.result}</span>
                    </td>
                    <td className={`!text-right font-mono tabular-nums ${latColor}`}>
                      {lat > 0 ? lat.toFixed(1) + 'ms' : '—'}
                    </td>
                    <td className="font-mono text-text-muted truncate max-w-[180px]">
                      {q.upstream || '—'}
                    </td>
                  </tr>
                )
              })}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  )
}
