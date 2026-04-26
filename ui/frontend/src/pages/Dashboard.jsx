import { useState, useEffect } from 'react'
import {
  AreaChart, Area, BarChart, Bar,
  XAxis, YAxis, Tooltip, ResponsiveContainer,
} from 'recharts'
import {
  Activity, ShieldOff, Database, Clock,
  Trash2, RefreshCw, Power, Zap, Flame, Server, ArrowDown,
} from 'lucide-react'
import StatCard from '../components/StatCard.jsx'

function fmt(n) {
  if (n == null) return '—'
  if (n >= 1_000_000) return (n / 1_000_000).toFixed(1) + 'M'
  if (n >= 1_000) return (n / 1_000).toFixed(1) + 'K'
  return String(n)
}

function fmtUptime(sec) {
  if (!sec) return 'just started'
  const h = Math.floor(sec / 3600)
  const m = Math.floor((sec % 3600) / 60)
  const s = Math.floor(sec % 60)
  if (h >= 24) return `${Math.floor(h / 24)}d ${h % 24}h`
  if (h > 0) return `${h}h ${m}m`
  if (m > 0) return `${m}m ${s}s`
  return `${s}s`
}

const tooltipProps = {
  contentStyle: {
    background: '#0f0f10',
    border: '1px solid #1f1f23',
    borderRadius: 6,
    fontSize: 11,
    fontFamily: 'JetBrains Mono, monospace',
    padding: '4px 8px',
  },
  labelStyle: { color: '#6e6e74' },
  cursor: { fill: 'rgba(163, 230, 53, 0.05)' },
}

export default function Dashboard({ api }) {
  const [stats, setStats]         = useState(null)
  const [status, setStatus]       = useState(null)
  const [upstreams, setUpstreams] = useState([])
  const [hot, setHot]             = useState([])
  const [netDNS, setNetDNS]       = useState({ network_dns: [], system_dns: [] })
  const [toast, setToast]         = useState('')

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
    try { await api.post(path, {}); setToast(`${label} · ok`) }
    catch { setToast(`${label} · failed`) }
    setTimeout(() => setToast(''), 2500)
  }

  const qpsData    = (stats?.qps_history ?? Array(60).fill(0)).map((v, i) => ({ t: i, qps: v }))
  const topDomains = (stats?.top_domains ?? []).slice(0, 5)
  const topBlocked = (stats?.top_blocked ?? []).slice(0, 5)
  const running    = status?.running

  return (
    <div className="px-7 py-6 space-y-6 max-w-[1400px]">
      <header className="flex items-end justify-between pt-2">
        <div>
          <h1 className="h-page">Dashboard</h1>
          <p className="mt-1 text-xs text-text-muted">
            <span className="font-mono">{fmtUptime(status?.uptime)}</span>
            <span className="mx-2 text-text-dim">·</span>
            <span className="font-mono">{status?.listen ?? '—'}</span>
          </p>
        </div>

        <RuntimeBadge running={running} />
      </header>

      <section className="grid grid-cols-2 xl:grid-cols-4 gap-3">
        <StatCard
          label="Queries"
          value={fmt(stats?.total_queries)}
          sub={`${(stats?.queries_per_sec ?? 0).toFixed(1)}/s · today`}
          accent="accent"
          icon={Activity}
        />
        <StatCard
          label="Cache"
          value={stats ? stats.cache_hit_rate.toFixed(0) + '%' : '—'}
          sub={`${fmt(stats?.total_cached)} cached`}
          accent="info"
          icon={Database}
        />
        <StatCard
          label="Blocked"
          value={fmt(stats?.total_blocked)}
          sub="rejected by policy"
          accent="bad"
          icon={ShieldOff}
        />
        <StatCard
          label="Latency"
          value={stats ? stats.avg_latency_ms.toFixed(1) + 'ms' : '—'}
          sub="upstream avg"
          accent="warn"
          icon={Clock}
        />
      </section>

      <section className="panel p-4">
        <div className="flex items-end justify-between mb-2">
          <div>
            <p className="h-card">Queries per second</p>
            <p className="mt-0.5 num-display text-[20px] leading-none">
              {(stats?.queries_per_sec ?? 0).toFixed(1)}
              <span className="ml-1 text-2xs font-mono text-text-muted tracking-normal">/s</span>
            </p>
          </div>
          <p className="text-2xs font-mono text-text-dim uppercase tracking-widest">
            rolling 60s
          </p>
        </div>
        <div className="h-[120px] -mx-2">
          <ResponsiveContainer width="100%" height="100%">
            <AreaChart data={qpsData} margin={{ top: 4, right: 8, left: 0, bottom: 0 }}>
              <defs>
                <linearGradient id="qg" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="0%"   stopColor="#a3e635" stopOpacity={0.35} />
                  <stop offset="100%" stopColor="#a3e635" stopOpacity={0} />
                </linearGradient>
              </defs>
              <XAxis dataKey="t" hide />
              <YAxis tick={{ fill: '#4a4a50', fontSize: 9 }} allowDecimals={false} width={28} />
              <Tooltip {...tooltipProps} itemStyle={{ color: '#a3e635' }} />
              <Area
                type="monotone" dataKey="qps"
                stroke="#a3e635" strokeWidth={1.5}
                fill="url(#qg)" dot={false}
                activeDot={{ r: 3, fill: '#a3e635', stroke: '#0a0a0a', strokeWidth: 2 }}
              />
            </AreaChart>
          </ResponsiveContainer>
        </div>
      </section>

      <section className="grid stable-grid-cols-2 gap-3">
        <TopList title="Most queried"  data={topDomains} color="#7dd3fc" />
        <TopList title="Most blocked"  data={topBlocked} color="#f87171" />
      </section>

      <section className="grid stable-grid-cols-2 gap-3">
        <UpstreamPanel upstreams={upstreams} netDNS={netDNS} />
        <HotPanel hot={hot} />
      </section>

      <section className="panel p-3.5">
        <div className="flex items-center justify-between">
          <p className="h-card">Quick actions</p>
          {toast && (
            <span className="text-2xs font-mono text-accent uppercase tracking-widest animate-fade-in">
              {toast}
            </span>
          )}
        </div>
        <div className="mt-3 flex flex-wrap gap-2">
          <button className="btn-ghost" onClick={() => action('/cache/flush',   'flush')}>
            <Trash2 size={12} /> Flush cache
          </button>
          <button className="btn-ghost" onClick={() => action('/prefetch/run',  'prefetch')}>
            <Zap size={12} /> Prefetch hot
          </button>
          <button className="btn-ghost" onClick={() => action('/server/restart','restart')}>
            <RefreshCw size={12} /> Restart
          </button>
          <button className="btn-danger" onClick={() => action('/server/stop',  'stop')}>
            <Power size={12} /> Stop server
          </button>
        </div>
      </section>
    </div>
  )
}

function RuntimeBadge({ running }) {
  if (running == null) {
    return (
      <div className="flex items-center gap-2 px-2.5 h-8 rounded-md bg-ink-200 ring-1 ring-line">
        <span className="dot-mute" />
        <span className="text-2xs font-mono uppercase tracking-widest text-text-muted">
          connecting
        </span>
      </div>
    )
  }
  if (running) {
    return (
      <div className="flex items-center gap-2 px-2.5 h-8 rounded-md bg-accent/10 ring-1 ring-accent/30">
        <span className="relative inline-flex h-2 w-2">
          <span className="absolute inset-0 rounded-full bg-accent opacity-50 animate-ping" />
          <span className="relative inline-flex rounded-full h-2 w-2 bg-accent" />
        </span>
        <span className="text-2xs font-mono uppercase tracking-widest text-accent">
          running
        </span>
      </div>
    )
  }
  return (
    <div className="flex items-center gap-2 px-2.5 h-8 rounded-md bg-bad/10 ring-1 ring-bad/30">
      <span className="dot-bad" />
      <span className="text-2xs font-mono uppercase tracking-widest text-bad">
        stopped
      </span>
    </div>
  )
}

function TopList({ title, data, color }) {
  return (
    <div className="panel p-4">
      <p className="h-card mb-3">{title}</p>
      {data.length === 0
        ? <p className="py-6 text-center text-2xs text-text-dim font-mono">no data</p>
        : (
          <div className="h-[150px]">
            <ResponsiveContainer width="100%" height="100%">
              <BarChart data={data} layout="vertical" margin={{ left: 0, right: 8, top: 2, bottom: 2 }}>
                <XAxis type="number" tick={{ fill: '#4a4a50', fontSize: 9 }} allowDecimals={false} />
                <YAxis
                  type="category" dataKey="domain"
                  tick={{ fill: '#a3a3a8', fontSize: 10, fontFamily: 'JetBrains Mono' }}
                  width={120}
                />
                <Tooltip {...tooltipProps} itemStyle={{ color }} />
                <Bar dataKey="count" fill={color} radius={[0, 2, 2, 0]} />
              </BarChart>
            </ResponsiveContainer>
          </div>
        )
      }
    </div>
  )
}

function UpstreamPanel({ upstreams, netDNS }) {
  const sysDNS = (netDNS.system_dns ?? []).filter(s => !s.startsWith('127.'))
  const netRouter = netDNS.network_dns ?? []

  return (
    <div className="panel p-4">
      <div className="flex items-center justify-between mb-3">
        <p className="h-card flex items-center gap-1.5">
          <Server size={11} className="text-text-dim" /> Upstream chain
        </p>
        <span className="dot-good" />
      </div>

      {upstreams.length === 0 ? (
        <p className="py-6 text-center text-2xs text-text-dim font-mono">no data</p>
      ) : (
        <div className="space-y-3">
          <div className="space-y-1">
            <Tier label="DoT" tone="violet" />
            {upstreams.map(u => <UpstreamRow key={u.server} u={u} />)}
          </div>

          {netRouter.length > 0 && (
            <div className="space-y-1">
              <FallArrow />
              <Tier label="router" tone="warn" />
              {netRouter.map(s => <FallbackRow key={s} addr={s} />)}
            </div>
          )}

          {sysDNS.length > 0 && (
            <div className="space-y-1">
              <FallArrow />
              <Tier label="system" tone="info" />
              {sysDNS.map(s => <FallbackRow key={s} addr={s} />)}
            </div>
          )}
        </div>
      )}
    </div>
  )
}

function Tier({ label, tone }) {
  const cls = tone === 'violet' ? 'chip-violet'
            : tone === 'warn'   ? 'chip-warn'
            : tone === 'info'   ? 'chip-info'
            : 'chip-mute'
  return (
    <div className="flex items-center gap-2">
      <span className={cls}>{label}</span>
      <span className="h-px flex-1 bg-line/40" />
    </div>
  )
}

function FallArrow() {
  return (
    <div className="flex justify-center py-0.5">
      <ArrowDown size={11} className="text-text-dim" />
    </div>
  )
}

function UpstreamRow({ u }) {
  const healthy = u.fail_streak === 0
  const tone =
    u.avg_ms === 0 ? 'mute'
    : u.avg_ms < 30 ? 'good'
    : u.avg_ms < 80 ? 'warn'
    : 'bad'
  const toneClass = {
    good: 'text-accent',
    warn: 'text-warn',
    bad: 'text-bad',
    mute: 'text-text-muted',
  }[tone]

  return (
    <div className="flex items-center justify-between rounded-md px-2.5 h-7 bg-ink-300/40">
      <div className="flex items-center gap-2 min-w-0">
        <span className={healthy ? 'dot-good' : 'dot-bad'} />
        <span className="text-2xs font-mono text-text-secondary truncate">{u.server}</span>
      </div>
      <div className="flex items-center gap-2 flex-shrink-0">
        {u.fail_streak > 0 && (
          <span className="text-2xs font-mono text-bad">{u.fail_streak}f</span>
        )}
        <span className={`text-2xs font-mono tabular-nums ${toneClass}`}>
          {u.avg_ms > 0 ? u.avg_ms.toFixed(1) + 'ms' : '—'}
        </span>
      </div>
    </div>
  )
}

function FallbackRow({ addr }) {
  return (
    <div className="flex items-center gap-2 px-2.5 h-7 rounded-md bg-ink-300/30">
      <span className="dot-mute" />
      <span className="text-2xs font-mono text-text-muted">{addr}</span>
    </div>
  )
}

function HotPanel({ hot }) {
  return (
    <div className="panel p-4">
      <div className="flex items-center justify-between mb-3">
        <p className="h-card flex items-center gap-1.5">
          <Flame size={11} className="text-warn" /> Hot domains
        </p>
        <p className="text-2xs font-mono text-text-dim uppercase tracking-widest">prefetched</p>
      </div>
      {hot.length === 0
        ? <p className="py-6 text-center text-2xs text-text-dim font-mono">no entries</p>
        : (
          <div className="space-y-0.5 max-h-[220px] overflow-y-auto">
            {hot.slice(0, 12).map((h, i) => (
              <div key={i} className="flex items-center justify-between rounded px-2 h-7 row-hover">
                <span className="text-2xs font-mono text-text-secondary truncate">{h.name}</span>
                <div className="flex items-center gap-2.5 flex-shrink-0">
                  <span className="text-2xs font-mono text-text-dim">{h.ttl_sec}s</span>
                  <span className="text-2xs font-mono text-warn tabular-nums">{h.hits}×</span>
                </div>
              </div>
            ))}
          </div>
        )
      }
    </div>
  )
}
