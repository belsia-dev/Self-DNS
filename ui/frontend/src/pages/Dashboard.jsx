import { useState, useEffect } from 'react'
import {
  AreaChart, Area, BarChart, Bar,
  XAxis, YAxis, Tooltip, ResponsiveContainer, CartesianGrid,
} from 'recharts'
import {
  Activity, ShieldOff, Database, Clock,
  Trash2, RefreshCw, Power, Zap, Flame, Server,
} from 'lucide-react'
import StatCard from '../components/StatCard.jsx'

function fmt(n) {
  if (n == null) return '—'
  if (n >= 1_000_000) return (n / 1_000_000).toFixed(1) + 'M'
  if (n >= 1_000) return (n / 1_000).toFixed(1) + 'K'
  return String(n)
}

function fmtUptime(sec) {
  if (!sec) return '—'
  const h = Math.floor(sec / 3600)
  const m = Math.floor((sec % 3600) / 60)
  const s = Math.floor(sec % 60)
  if (h > 0) return `${h}h ${m}m uptime`
  if (m > 0) return `${m}m ${s}s uptime`
  return `${s}s uptime`
}

const chartTooltipStyle = {
  contentStyle: { background: '#111827', border: '1px solid #1f2937', borderRadius: 8, fontSize: 11 },
  labelStyle: { color: '#6b7280' },
}

export default function Dashboard({ api }) {
  const [stats, setStats] = useState(null)
  const [status, setStatus] = useState(null)
  const [upstreams, setUpstreams] = useState([])
  const [hot, setHot] = useState([])
  const [netDNS, setNetDNS] = useState({ network_dns: [], system_dns: [] })
  const [toast, setToast] = useState('')

  useEffect(() => {
    let alive = true
    const poll = async () => {
      try {
        const [s, st] = await Promise.all([api.get('/stats'), api.get('/status')])
        if (alive) { setStats(s); setStatus(st) }
      } catch {}
    }
    poll()
    const id = setInterval(poll, 1000)
    return () => { alive = false; clearInterval(id) }
  }, [api])

  useEffect(() => {
    let alive = true
    const poll = async () => {
      try {
        const [u, h, n] = await Promise.all([
          api.get('/upstreams'),
          api.get('/cache/hot'),
          api.get('/network-dns'),
        ])
        if (alive) { setUpstreams(u ?? []); setHot(h ?? []); setNetDNS(n ?? {}) }
      } catch {}
    }
    poll()
    const id = setInterval(poll, 3000)
    return () => { alive = false; clearInterval(id) }
  }, [api])

  const action = async (path, label) => {
    try { await api.post(path, {}); setToast(`${label} OK`) }
    catch { setToast(`${label} failed`) }
    setTimeout(() => setToast(''), 3000)
  }

  const qpsData = (stats?.qps_history ?? Array(60).fill(0)).map((v, i) => ({ t: i, qps: v }))
  const topDomains = (stats?.top_domains ?? []).slice(0, 5)
  const topBlocked = (stats?.top_blocked ?? []).slice(0, 5)
  const running = status?.running

  return (
    <div className="p-6 space-y-5">

      <div className="flex items-start justify-between pt-2">
        <div>
          <h1 className="text-xl font-bold text-white">Dashboard</h1>
          <p className="mt-0.5 text-sm text-gray-500">{fmtUptime(status?.uptime)}</p>
        </div>

        <div className={`flex items-center gap-2.5 rounded-xl px-4 py-2 text-sm font-semibold
                         border transition-colors ${running == null
            ? 'bg-gray-800 text-gray-500 border-gray-700'
            : running
              ? 'bg-green-900/30 text-green-400 border-green-700/40'
              : 'bg-red-900/30 text-red-400 border-red-700/40'
          }`}>
          <span className={`h-2.5 w-2.5 rounded-full ${running == null ? 'bg-gray-500'
            : running ? 'bg-green-400 animate-pulse'
              : 'bg-red-400'
            }`} />
          {running == null ? 'Connecting…'
            : running ? 'SelfDNS RUNNING'
              : 'SelfDNS STOPPED'}
        </div>
      </div>

      <div className="grid grid-cols-2 xl:grid-cols-4 gap-3">
        <StatCard label="Queries Today" value={fmt(stats?.total_queries)}
          color="blue" icon={Activity} sub={`${(stats?.queries_per_sec ?? 0).toFixed(1)} /sec`} />
        <StatCard label="Cache Hit Rate" value={stats ? stats.cache_hit_rate.toFixed(1) + '%' : '—'}
          color="green" icon={Database} sub={`${fmt(stats?.total_cached)} cached`} />
        <StatCard label="Blocked Today" value={fmt(stats?.total_blocked)}
          color="red" icon={ShieldOff} sub="NXDOMAIN responses" />
        <StatCard label="Avg Latency" value={stats ? stats.avg_latency_ms.toFixed(1) + ' ms' : '—'}
          color="orange" icon={Clock} sub="upstream round-trip" />
      </div>

      <div className="card">
        <div className="flex items-center justify-between mb-4">
          <p className="text-sm font-semibold text-gray-200">Queries / Second</p>
          <p className="text-xs text-gray-600">rolling 60 s</p>
        </div>
        <ResponsiveContainer width="100%" height={110}>
          <AreaChart data={qpsData} margin={{ top: 2, right: 0, left: -24, bottom: 0 }}>
            <defs>
              <linearGradient id="qg" x1="0" y1="0" x2="0" y2="1">
                <stop offset="5%" stopColor="#3b82f6" stopOpacity={0.25} />
                <stop offset="95%" stopColor="#3b82f6" stopOpacity={0} />
              </linearGradient>
            </defs>
            <CartesianGrid strokeDasharray="3 3" stroke="#1f2937" vertical={false} />
            <XAxis dataKey="t" hide />
            <YAxis tick={{ fill: '#4b5563', fontSize: 10 }} allowDecimals={false} />
            <Tooltip {...chartTooltipStyle} itemStyle={{ color: '#60a5fa' }} />
            <Area type="monotone" dataKey="qps" stroke="#3b82f6" strokeWidth={2}
              fill="url(#qg)" dot={false} activeDot={{ r: 3 }} />
          </AreaChart>
        </ResponsiveContainer>
      </div>

      <div className="grid grid-cols-2 gap-3">
        {[
          { title: 'Top Queried Domains', data: topDomains, color: '#3b82f6' },
          { title: 'Top Blocked Domains', data: topBlocked, color: '#ef4444' },
        ].map(({ title, data, color }) => (
          <div key={title} className="card">
            <p className="mb-3 text-sm font-semibold text-gray-200">{title}</p>
            {data.length === 0
              ? <p className="py-6 text-center text-xs text-gray-600">No data yet</p>
              : <ResponsiveContainer width="100%" height={150}>
                <BarChart data={data} layout="vertical" margin={{ left: 0, right: 8, top: 2, bottom: 2 }}>
                  <XAxis type="number" tick={{ fill: '#4b5563', fontSize: 10 }} allowDecimals={false} />
                  <YAxis type="category" dataKey="domain"
                    tick={{ fill: '#9ca3af', fontSize: 10 }} width={110} />
                  <Tooltip {...chartTooltipStyle} itemStyle={{ color }} />
                  <Bar dataKey="count" fill={color} radius={[0, 4, 4, 0]} />
                </BarChart>
              </ResponsiveContainer>
            }
          </div>
        ))}
      </div>

      <div className="grid grid-cols-2 gap-3">
        <div className="card">
          <div className="flex items-center justify-between mb-3">
            <p className="flex items-center gap-1.5 text-sm font-semibold text-gray-200">
              <Server size={13} className="text-gray-500" /> Upstream Health
            </p>
            <p className="text-[10px] text-gray-600 uppercase tracking-wider">live</p>
          </div>
          {upstreams.length === 0
            ? <p className="py-6 text-center text-xs text-gray-600">No upstream data yet</p>
            : <div className="space-y-1.5">
              {upstreams.map(u => {
                const healthy = u.fail_streak === 0
                const latBucket = u.avg_ms === 0 ? 'gray'
                  : u.avg_ms < 30 ? 'green'
                    : u.avg_ms < 80 ? 'orange' : 'red'
                const colors = {
                  green: 'text-green-400 bg-green-900/20 border-green-800/40',
                  orange: 'text-orange-400 bg-orange-900/20 border-orange-800/40',
                  red: 'text-red-400 bg-red-900/20 border-red-800/40',
                  gray: 'text-gray-400 bg-gray-800 border-gray-700',
                }
                return (
                  <div key={u.server} className="flex items-center justify-between px-3 py-1.5 rounded-lg border border-gray-700/50 bg-gray-900/40">
                    <div className="flex items-center gap-2 min-w-0">
                      <span className={`h-2 w-2 rounded-full ${healthy ? 'bg-green-400' : 'bg-red-400'}`} />
                      <span className="text-xs font-mono text-gray-300 truncate">{u.server}</span>
                    </div>
                    <div className="flex items-center gap-2 flex-shrink-0">
                      {u.fail_streak > 0 && (
                        <span className="text-[10px] text-red-400 font-mono">{u.fail_streak} fails</span>
                      )}
                      <span className={`text-[10px] font-mono px-2 py-0.5 rounded border ${colors[latBucket]}`}>
                        {u.avg_ms > 0 ? u.avg_ms.toFixed(1) + 'ms' : '—'}
                      </span>
                    </div>
                  </div>
                )
              })}

              {(netDNS.network_dns?.length > 0 || netDNS.system_dns?.length > 0) && (
                <div className="mt-2 pt-2 border-t border-gray-700/40 space-y-1">
                  <p className="text-[10px] text-gray-600 uppercase tracking-wider px-1">Resolution order</p>
                  {netDNS.system_dns?.filter(s => !s.startsWith('127.')).map((s, i) => (
                    <div key={s} className="flex items-center justify-between px-3 py-1 rounded-lg border border-gray-700/30 bg-gray-900/20">
                      <div className="flex items-center gap-2">
                        <span className="text-[9px] font-mono text-gray-600 w-3">{i + 1}</span>
                        <span className="h-1.5 w-1.5 rounded-full bg-blue-400/60" />
                        <span className="text-[11px] font-mono text-gray-400">{s}</span>
                      </div>
                      <span className="text-[10px] text-blue-500 bg-blue-900/20 px-1.5 py-0.5 rounded border border-blue-800/30">system</span>
                    </div>
                  ))}
                  {netDNS.network_dns?.map((s, i) => {
                    const offset = (netDNS.system_dns?.filter(x => !x.startsWith('127.')).length ?? 0) + i + 1
                    return (
                      <div key={s} className="flex items-center justify-between px-3 py-1 rounded-lg border border-yellow-800/30 bg-yellow-900/10">
                        <div className="flex items-center gap-2">
                          <span className="text-[9px] font-mono text-gray-600 w-3">{offset}</span>
                          <span className="h-1.5 w-1.5 rounded-full bg-yellow-400/80" />
                          <span className="text-[11px] font-mono text-yellow-300/80">{s}</span>
                        </div>
                        <span className="text-[10px] text-yellow-600 bg-yellow-900/20 px-1.5 py-0.5 rounded border border-yellow-800/30">router</span>
                      </div>
                    )
                  })}
                  <div className="flex items-center justify-between px-3 py-1 rounded-lg border border-purple-800/30 bg-purple-900/10">
                    <div className="flex items-center gap-2">
                      <span className="text-[9px] font-mono text-gray-600 w-3">↓</span>
                      <span className="h-1.5 w-1.5 rounded-full bg-purple-400/80" />
                      <span className="text-[11px] font-mono text-purple-300/70">DoT upstreams (1.1.1.1, 8.8.8.8…)</span>
                    </div>
                    <span className="text-[10px] text-purple-500 bg-purple-900/20 px-1.5 py-0.5 rounded border border-purple-800/30">last resort</span>
                  </div>
                </div>
              )}
            </div>
          }
        </div>

        <div className="card">
          <div className="flex items-center justify-between mb-3">
            <p className="flex items-center gap-1.5 text-sm font-semibold text-gray-200">
              <Flame size={13} className="text-orange-400" /> Hot Domains
            </p>
            <p className="text-[10px] text-gray-600 uppercase tracking-wider">prefetched</p>
          </div>
          {hot.length === 0
            ? <p className="py-6 text-center text-xs text-gray-600">No hot entries yet</p>
            : <div className="space-y-1 max-h-[200px] overflow-y-auto pr-1">
              {hot.slice(0, 10).map((h, i) => (
                <div key={i} className="flex items-center justify-between px-2 py-1 rounded-lg hover:bg-gray-900/40">
                  <span className="text-xs font-mono text-gray-300 truncate">{h.name}</span>
                  <div className="flex items-center gap-2 flex-shrink-0">
                    <span className="text-[10px] text-gray-600 font-mono">{h.ttl_sec}s</span>
                    <span className="text-[10px] font-mono text-orange-400">{h.hits}×</span>
                  </div>
                </div>
              ))}
            </div>
          }
        </div>
      </div>

      <div className="card">
        <p className="mb-3 text-sm font-semibold text-gray-200">Quick Actions</p>
        <div className="flex flex-wrap gap-2">
          <button className="btn-ghost flex items-center gap-2"
            onClick={() => action('/cache/flush', 'Flush cache')}>
            <Trash2 size={13} /> Flush Cache
          </button>
          <button className="btn-ghost flex items-center gap-2"
            onClick={() => action('/prefetch/run', 'Prefetch')}>
            <Zap size={13} /> Prefetch Hot
          </button>
          <button className="btn-primary flex items-center gap-2"
            onClick={() => action('/server/restart', 'Restart')}>
            <RefreshCw size={13} /> Restart Server
          </button>
          <button className="btn-danger flex items-center gap-2"
            onClick={() => action('/server/stop', 'Stop')}>
            <Power size={13} /> Stop Server
          </button>
          {toast && <span className="ml-2 text-xs text-blue-400 self-center">{toast}</span>}
        </div>
      </div>
    </div>
  )
}
