import { useState, useEffect, useRef } from 'react'
import { Pause, Play, Download, Search, X } from 'lucide-react'

const RESULT_STYLE = {
  RESOLVED: 'badge-green',
  CACHED: 'badge-orange',
  BLOCKED: 'badge-red',
  ERROR: 'badge-gray',
}

function ts(iso) {
  const d = new Date(iso)
  return d.toLocaleTimeString('en-US', { hour12: false })
}

export default function QueryLog({ api }) {
  const [queries, setQueries] = useState([])
  const [paused, setPaused] = useState(false)
  const [filter, setFilter] = useState('')
  const [typeFilter, setTypeFilter] = useState('ALL')
  const pausedRef = useRef(false)

  useEffect(() => { pausedRef.current = paused }, [paused])

  useEffect(() => {
    const poll = async () => {
      if (pausedRef.current) return
      try {
        const data = await api.get('/queries')
        setQueries(data ?? [])
      } catch { }
    }
    poll()
    const id = setInterval(poll, 500)
    return () => clearInterval(id)
  }, [api])

  const visible = queries.filter(q => {
    const domainMatch = !filter || q.domain.toLowerCase().includes(filter.toLowerCase())
    const typeMatch = typeFilter === 'ALL' || q.result === typeFilter
    return domainMatch && typeMatch
  })

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
    <div className="flex flex-col h-full p-6 space-y-4">
      <div className="flex items-center justify-between pt-2">
        <div>
          <h1 className="text-xl font-bold text-white">Query Log</h1>
          <p className="text-sm text-gray-400">{visible.length} entries shown · live updates every 500ms</p>
        </div>
        <div className="flex items-center gap-2">
          <button className="btn-ghost flex items-center gap-2" onClick={exportCSV}>
            <Download size={14} /> Export CSV
          </button>
          <button
            className={`flex items-center gap-2 text-sm font-medium px-4 py-2 rounded-lg transition-colors ${paused ? 'btn-primary' : 'bg-yellow-700 hover:bg-yellow-600 text-white'
              }`}
            onClick={() => setPaused(p => !p)}
          >
            {paused ? <><Play size={14} /> Resume</> : <><Pause size={14} /> Pause</>}
          </button>
        </div>
      </div>

      <div className="flex gap-3">
        <div className="relative flex-1">
          <Search size={14} className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-500" />
          <input
            className="input pl-8"
            placeholder="Filter by domain…"
            value={filter}
            onChange={e => setFilter(e.target.value)}
          />
          {filter && (
            <button className="absolute right-3 top-1/2 -translate-y-1/2 text-gray-500 hover:text-gray-300"
              onClick={() => setFilter('')}>
              <X size={14} />
            </button>
          )}
        </div>
        <select
          className="input w-40"
          value={typeFilter}
          onChange={e => setTypeFilter(e.target.value)}
        >
          {['ALL', 'RESOLVED', 'CACHED', 'BLOCKED', 'ERROR'].map(t => (
            <option key={t} value={t}>{t}</option>
          ))}
        </select>
      </div>

      <div className="flex-1 overflow-auto rounded-xl border border-gray-700">
        <table className="w-full text-sm border-collapse">
          <thead className="sticky top-0 bg-gray-900 z-10">
            <tr className="text-xs text-gray-500 uppercase tracking-wide">
              <th className="text-left px-4 py-2.5 font-medium">Time</th>
              <th className="text-left px-4 py-2.5 font-medium">Domain</th>
              <th className="text-left px-4 py-2.5 font-medium">Type</th>
              <th className="text-left px-4 py-2.5 font-medium">Result</th>
              <th className="text-right px-4 py-2.5 font-medium">Latency</th>
              <th className="text-left px-4 py-2.5 font-medium">Upstream</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-800">
            {visible.length === 0 && (
              <tr>
                <td colSpan={6} className="text-center text-gray-500 py-12 text-sm">
                  {paused ? 'Updates paused.' : 'Waiting for queries…'}
                </td>
              </tr>
            )}
            {visible.map((q, i) => (
              <tr key={i} className="hover:bg-gray-800/50 transition-colors">
                <td className="px-4 py-2 font-mono text-xs text-gray-400 whitespace-nowrap">{ts(q.timestamp)}</td>
                <td className="px-4 py-2 font-mono text-xs text-gray-200 max-w-xs truncate">{q.domain}</td>
                <td className="px-4 py-2">
                  <span className="badge-gray font-mono">{q.type}</span>
                </td>
                <td className="px-4 py-2">
                  <span className={RESULT_STYLE[q.result] ?? 'badge-gray'}>{q.result}</span>
                </td>
                <td className="px-4 py-2 text-right font-mono text-xs text-gray-400">
                  {q.latency_ms > 0 ? `${q.latency_ms.toFixed(1)}ms` : '—'}
                </td>
                <td className="px-4 py-2 font-mono text-xs text-gray-500 max-w-[140px] truncate">
                  {q.upstream || '—'}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}
