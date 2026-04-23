import { Component, useState, useEffect, useCallback } from 'react'
import {
  Save, RefreshCw, Plus, Trash2,
  CheckCircle, AlertCircle, Shield, Database,
  Eye, Network, Gauge, Info, Download, Upload, Zap,
  ArrowDown, Globe, Router, Lock,
} from 'lucide-react'

class ErrorBoundary extends Component {
  constructor(props) {
    super(props)
    this.state = { hasError: false, error: null }
  }

  static getDerivedStateFromError(error) {
    return { hasError: true, error }
  }

  componentDidCatch(error, info) {
    console.error('[Settings] render error:', error, info.componentStack)
  }

  render() {
    if (this.state.hasError) {
      return (
        <div className="flex flex-col items-center justify-center h-full gap-5 p-8">
          <div className="flex items-center gap-2 text-red-400">
            <AlertCircle size={18} />
            <p className="font-semibold">Settings crashed</p>
          </div>
          <pre className="text-[11px] text-gray-500 bg-gray-900 rounded-lg px-4 py-3 max-w-lg overflow-auto border border-gray-800">
            {String(this.state.error)}
          </pre>
          <button
            className="btn-primary text-sm"
            onClick={() => this.setState({ hasError: false, error: null })}
          >
            Try again
          </button>
        </div>
      )
    }
    return this.props.children
  }
}

function Toggle({ value, onChange }) {
  return (
    <button
      type="button"
      role="switch"
      aria-checked={!!value}
      onClick={() => onChange(!value)}
      className={`relative inline-flex h-6 w-11 flex-shrink-0 cursor-pointer rounded-full
                  border-2 border-transparent transition-colors duration-200 ease-in-out
                  focus:outline-none focus:ring-2 focus:ring-blue-500/40 focus:ring-offset-1
                  focus:ring-offset-gray-800 ${value ? 'bg-blue-500' : 'bg-gray-600'}`}
    >
      <span
        aria-hidden="true"
        className={`pointer-events-none inline-block h-5 w-5 transform rounded-full
                    bg-white shadow ring-0 transition duration-200 ease-in-out
                    ${value ? 'translate-x-5' : 'translate-x-0'}`}
      />
    </button>
  )
}

function Row({ label, desc, children, indented = false }) {
  return (
    <div className={`flex items-center justify-between gap-6 px-5 py-3.5
                     ${indented ? 'pl-11 bg-gray-900/40 border-l-2 border-blue-800/30' : ''}`}>
      <div className="flex-1 min-w-0">
        <p className={`text-sm font-medium leading-snug
                       ${indented ? 'text-gray-400' : 'text-gray-100'}`}>{label}</p>
        {desc && <p className="mt-0.5 text-xs text-gray-500 leading-relaxed">{desc}</p>}
      </div>
      <div className="flex-shrink-0">{children}</div>
    </div>
  )
}

function SliderRow({ label, desc, value, onChange, min, max, unit = '', step = 1, indented = false }) {
  return (
    <div className={`px-5 py-3.5 ${indented ? 'pl-11 bg-gray-900/40 border-l-2 border-blue-800/30' : ''}`}>
      <div className="flex items-center justify-between mb-2.5">
        <div>
          <p className={`text-sm font-medium ${indented ? 'text-gray-400' : 'text-gray-100'}`}>{label}</p>
          {desc && <p className="mt-0.5 text-xs text-gray-500">{desc}</p>}
        </div>
        <span className="font-mono text-sm text-blue-400 bg-blue-900/20 border border-blue-800/30
                         px-2.5 py-0.5 rounded-md min-w-[64px] text-center">
          {Number(value).toLocaleString()}{unit}
        </span>
      </div>
      <input
        type="range" min={min} max={max} step={step} value={value}
        onChange={e => onChange(Number(e.target.value))}
        className="w-full h-1.5 appearance-none rounded-full bg-gray-600 accent-blue-500 cursor-pointer"
      />
      <div className="mt-1.5 flex justify-between text-[10px] text-gray-600">
        <span>{Number(min).toLocaleString()}{unit}</span>
        <span>{Number(max).toLocaleString()}{unit}</span>
      </div>
    </div>
  )
}

function Section({ title, icon: Icon, children }) {
  return (
    <div className="space-y-1.5">
      <div className="flex items-center gap-1.5 px-0.5">
        {Icon && <Icon size={11} className="text-gray-500" />}
        <p className="text-[10.5px] font-semibold uppercase tracking-widest text-gray-500">{title}</p>
      </div>
      <div className="overflow-hidden rounded-xl border border-gray-700/50 bg-gray-800 divide-y divide-gray-700/40">
        {children}
      </div>
    </div>
  )
}

function Hint({ text }) {
  return (
    <span className="group relative inline-flex items-center ml-1.5 cursor-help">
      <Info size={11} className="text-gray-600 hover:text-gray-400 transition-colors" />
      <span className="pointer-events-none absolute left-full ml-2 top-1/2 -translate-y-1/2
                       hidden group-hover:flex z-50 bg-gray-900 text-gray-300 text-xs
                       rounded-lg px-3 py-2 border border-gray-700 shadow-xl w-56">
        {text}
      </span>
    </span>
  )
}

function IPListEditor({ ips, onAdd, onRemove }) {
  const [val, setVal] = useState('')

  const add = () => {
    const s = val.trim()
    if (!s) return
    onAdd(s)
    setVal('')
  }

  return (
    <div className="px-5 py-3.5 pl-11 bg-gray-900/40 border-l-2 border-blue-800/30 space-y-2">
      <p className="text-xs font-medium text-gray-400 mb-2">
        Whitelisted IPs
        <Hint text="These source IPs are never rate-limited, even during a flood." />
      </p>

      {ips.length === 0 && (
        <p className="text-xs italic text-gray-600">No whitelisted IPs.</p>
      )}
      {ips.map((ip, i) => (
        <div key={i} className="flex items-center gap-2 group">
          <span className="flex-1 rounded-lg border border-gray-700 bg-gray-800 px-3 py-1
                           text-xs font-mono text-gray-300">
            {ip}
          </span>
          <button
            onClick={() => onRemove(i)}
            className="text-gray-600 opacity-0 group-hover:opacity-100 hover:text-red-400 transition-all"
          >
            <Trash2 size={13} />
          </button>
        </div>
      ))}

      <div className="flex gap-2 pt-1">
        <input
          placeholder="192.168.1.0/24 or 10.0.0.5"
          value={val}
          onChange={e => setVal(e.target.value)}
          onKeyDown={e => e.key === 'Enter' && add()}
          className="flex-1 rounded-lg border border-dashed border-gray-600 bg-gray-700/60
                     px-3 py-1.5 text-xs font-mono text-gray-300 placeholder-gray-600
                     focus:outline-none focus:ring-1 focus:ring-blue-500/50"
        />
        <button
          onClick={add}
          className="flex items-center gap-1.5 rounded-lg border border-gray-600 bg-gray-700
                     px-3 py-1.5 text-xs font-medium text-gray-300 hover:bg-gray-600 transition-colors"
        >
          <Plus size={11} /> Add
        </button>
      </div>
    </div>
  )
}

function CacheTools({ api }) {
  const [msg, setMsg] = useState('')
  const say = t => { setMsg(t); setTimeout(() => setMsg(''), 3000) }

  const exportCache = async () => {
    try {
      const data = await api.get('/cache/export')
      const blob = new Blob([JSON.stringify(data, null, 2)], { type: 'application/json' })
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = 'selfdns-cache.json'
      a.click()
      URL.revokeObjectURL(url)
      say(`Exported ${data.entries?.length ?? 0} entries`)
    } catch (e) { say('Export failed: ' + e.message) }
  }

  const importCache = async file => {
    if (!file) return
    try {
      const text = await file.text()
      const data = JSON.parse(text)
      const res = await api.post('/cache/import', data)
      say(`Imported ${res.entries} entries`)
    } catch (e) { say('Import failed: ' + e.message) }
  }

  const prefetch = async () => {
    try { await api.post('/prefetch/run', {}); say('Prefetching top domains…') }
    catch (e) { say('Prefetch failed: ' + e.message) }
  }

  const flush = async () => {
    try { await api.post('/cache/flush', {}); say('Cache flushed') }
    catch (e) { say('Flush failed: ' + e.message) }
  }

  return (
    <div className="px-5 py-3.5 pl-11 bg-gray-900/40 border-l-2 border-blue-800/30">
      <div className="flex flex-wrap items-center gap-2">
        <button onClick={exportCache}
          className="flex items-center gap-1.5 rounded-lg border border-gray-600 bg-gray-700
                     px-3 py-1.5 text-xs font-medium text-gray-300 hover:bg-gray-600 transition-colors">
          <Download size={12} /> Export
        </button>
        <label className="flex items-center gap-1.5 rounded-lg border border-gray-600 bg-gray-700
                          px-3 py-1.5 text-xs font-medium text-gray-300 hover:bg-gray-600 transition-colors cursor-pointer">
          <Upload size={12} /> Import
          <input type="file" accept="application/json" className="hidden"
            onChange={e => importCache(e.target.files?.[0])} />
        </label>
        <button onClick={prefetch}
          className="flex items-center gap-1.5 rounded-lg border border-blue-700/40 bg-blue-900/20
                     px-3 py-1.5 text-xs font-medium text-blue-300 hover:bg-blue-900/40 transition-colors">
          <Zap size={12} /> Prefetch hot domains
        </button>
        <button onClick={flush}
          className="flex items-center gap-1.5 rounded-lg border border-red-700/40 bg-red-900/10
                     px-3 py-1.5 text-xs font-medium text-red-300 hover:bg-red-900/30 transition-colors">
          <Trash2 size={12} /> Flush
        </button>
        {msg && <span className="text-[11px] text-blue-400 ml-2">{msg}</span>}
      </div>
    </div>
  )
}

function ResolutionOrder({ api }) {
  const [data, setData] = useState(null)
  const [refreshing, setRefreshing] = useState(false)

  const load = useCallback(async () => {
    try { setData(await api.get('/network-dns')) } catch {}
  }, [api])

  useEffect(() => { load() }, [load])

  const refresh = async () => {
    setRefreshing(true)
    try { await api.post('/network-dns', {}); setTimeout(load, 800) }
    catch {}
    setRefreshing(false)
  }

  const sysDNS = (data?.system_dns ?? []).filter(s => !s.startsWith('127.') && !s.startsWith('[::1]'))
  const netDNS = data?.network_dns ?? []

  const tiers = [
    {
      label: 'System DNS',
      sublabel: 'from /etc/resolv.conf',
      servers: sysDNS,
      icon: Globe,
      color: 'blue',
      badge: 'plain UDP',
      empty: 'None detected (system DNS may be set to 127.0.0.1)',
    },
    {
      label: 'Router / DHCP DNS',
      sublabel: 'your Wi-Fi gateway',
      servers: netDNS,
      icon: Router,
      color: 'yellow',
      badge: 'plain UDP',
      empty: 'None detected',
    },
    {
      label: 'DoT Upstreams',
      sublabel: 'configured servers — last resort',
      servers: null,
      icon: Lock,
      color: 'purple',
      badge: 'DNS-over-TLS',
      empty: null,
    },
  ]

  const colorMap = {
    blue:   { row: 'border-blue-800/30 bg-blue-900/10', dot: 'bg-blue-400', badge: 'text-blue-400 bg-blue-900/20 border-blue-800/30', icon: 'text-blue-400', label: 'text-blue-300', num: 'text-blue-500' },
    yellow: { row: 'border-yellow-800/30 bg-yellow-900/10', dot: 'bg-yellow-400', badge: 'text-yellow-500 bg-yellow-900/20 border-yellow-800/30', icon: 'text-yellow-400', label: 'text-yellow-300', num: 'text-yellow-600' },
    purple: { row: 'border-purple-800/30 bg-purple-900/10', dot: 'bg-purple-400', badge: 'text-purple-400 bg-purple-900/20 border-purple-800/30', icon: 'text-purple-400', label: 'text-purple-300', num: 'text-purple-500' },
  }

  return (
    <div className="px-5 py-4 space-y-2">
      <div className="flex items-center justify-between mb-3">
        <p className="text-xs text-gray-500">
          Queries resolve top-to-bottom. Lower tiers only fire if all servers above fail.
        </p>
        <button
          onClick={refresh}
          disabled={refreshing}
          className="flex items-center gap-1.5 text-[11px] text-gray-500 hover:text-gray-300
                     transition-colors disabled:opacity-40"
        >
          <RefreshCw size={11} className={refreshing ? 'animate-spin' : ''} />
          Re-detect
        </button>
      </div>

      {tiers.map((tier, idx) => {
        const c = colorMap[tier.color]
        const Icon = tier.icon
        return (
          <div key={tier.label}>
            <div className={`rounded-xl border ${c.row} overflow-hidden`}>
              <div className="flex items-center gap-3 px-4 py-2.5">
                <span className={`flex h-6 w-6 items-center justify-center rounded-lg text-xs font-bold
                                  ${c.badge} border`}>
                  {idx + 1}
                </span>
                <Icon size={13} className={c.icon} />
                <div className="flex-1 min-w-0">
                  <p className={`text-sm font-semibold ${c.label}`}>{tier.label}</p>
                  <p className="text-[10px] text-gray-500">{tier.sublabel}</p>
                </div>
                <span className={`text-[10px] font-medium px-2 py-0.5 rounded border ${c.badge}`}>
                  {tier.badge}
                </span>
              </div>

              {tier.servers !== null && (
                <div className="px-4 pb-3 pt-0 space-y-1">
                  {tier.servers.length === 0 ? (
                    <p className="text-[11px] italic text-gray-600 pl-9">{tier.empty}</p>
                  ) : tier.servers.map(s => (
                    <div key={s} className="flex items-center gap-2 pl-9">
                      <span className={`h-1.5 w-1.5 rounded-full flex-shrink-0 ${c.dot}`} />
                      <span className="text-xs font-mono text-gray-300">{s}</span>
                    </div>
                  ))}
                </div>
              )}

              {tier.servers === null && (
                <p className="text-[11px] italic text-gray-600 px-4 pb-3 pl-[52px]">
                  See "Upstream DNS Servers" above to configure
                </p>
              )}
            </div>

            {idx < tiers.length - 1 && (
              <div className="flex justify-center py-0.5">
                <ArrowDown size={13} className="text-gray-700" />
              </div>
            )}
          </div>
        )
      })}
    </div>
  )
}

function SettingsInner({ api }) {
  const [cfg, setCfg] = useState(null)
  const [saving, setSaving] = useState(false)
  const [saveResult, setSaveResult] = useState(null)
  const [startMsg, setStartMsg] = useState('Connecting to DNS server…')
  const [newUp, setNewUp] = useState('')

  useEffect(() => {
    let mounted = true
    let timer
    let attempts = 0

    const messages = [
      'Connecting to DNS server…',
      'Starting DNS server…',
      'Waiting for DNS server…',
      'Almost ready…',
    ]

    const tryLoad = async () => {
      try {
        const data = await api.get('/config')
        if (mounted) setCfg(data)
      } catch {
        if (!mounted) return
        attempts++
        setStartMsg(messages[Math.min(attempts - 1, messages.length - 1)])
        timer = setTimeout(tryLoad, 600)
      }
    }

    tryLoad()
    return () => { mounted = false; clearTimeout(timer) }
  }, [api])

  const set = (path, value) => {
    setCfg(prev => {
      if (prev == null) return prev
      const next = JSON.parse(JSON.stringify(prev))
      const keys = path.split('.')
      let cur = next
      for (let i = 0; i < keys.length - 1; i++) {
        if (cur[keys[i]] == null || typeof cur[keys[i]] !== 'object') {
          cur[keys[i]] = {}
        }
        cur = cur[keys[i]]
      }
      cur[keys[keys.length - 1]] = value
      return next
    })
  }

  const save = async () => {
    setSaving(true)
    try {
      await api.post('/config', cfg)
      setSaveResult('ok')
    } catch {
      setSaveResult('err')
    }
    setSaving(false)
    setTimeout(() => setSaveResult(null), 4000)
  }

  const addUpstream = () => {
    const s = newUp.trim()
    if (!s || !cfg) return
    set('upstream', [...(cfg.upstream ?? []), s])
    setNewUp('')
  }

  const removeUpstream = i =>
    set('upstream', (cfg.upstream ?? []).filter((_, idx) => idx !== i))

  if (!cfg) {
    return (
      <div className="flex flex-col items-center justify-center h-full gap-4">
        <div className="w-10 h-10 rounded-full border-2 border-blue-500 border-t-transparent animate-spin" />
        <div className="text-center">
          <p className="text-sm font-medium text-gray-300">{startMsg}</p>
          <p className="mt-1 text-xs text-gray-600">
            Port 53 requires root — run: <code className="text-gray-500">sudo selfdns-app</code>
          </p>
        </div>
      </div>
    )
  }

  const rl = cfg.rate_limit ?? {}
  const ca = cfg.cache ?? {}
  const whitelistIPs = rl.whitelist_ips ?? []
  const listenPort = (cfg.listen ?? '127.0.0.1:53').split(':').pop()

  return (
    <div className="flex flex-col h-full overflow-hidden">

      <div className="flex-shrink-0 flex items-center justify-between
                      px-6 py-4 border-b border-gray-800/80 bg-gray-950/60 backdrop-blur-sm">
        <div>
          <h1 className="text-lg font-bold text-white">Settings</h1>
          <p className="text-xs text-gray-500 mt-0.5">Changes apply after Save + Restart</p>
        </div>

        <div className="flex items-center gap-3">
          {saveResult === 'ok' && (
            <span className="flex items-center gap-1.5 text-sm text-green-400 animate-in fade-in">
              <CheckCircle size={14} /> Saved &amp; restarted
            </span>
          )}
          {saveResult === 'err' && (
            <span className="flex items-center gap-1.5 text-sm text-red-400">
              <AlertCircle size={14} /> Save failed
            </span>
          )}
          <button
            onClick={save}
            disabled={saving}
            className="btn-primary flex items-center gap-2 min-w-[140px] justify-center"
          >
            {saving
              ? <><RefreshCw size={13} className="animate-spin" /> Saving…</>
              : <><Save size={13} /> Save + Restart</>
            }
          </button>
        </div>
      </div>

      <div className="flex-1 overflow-y-auto px-6 py-5 space-y-5">

        <Section title="Network" icon={Network}>
          <Row label="DNS Listen Port" desc="Locked to 127.0.0.1 loopback for security.">
            <div className="flex items-center gap-2">
              <span className="px-2.5 py-1 rounded-lg bg-gray-700 text-xs font-mono text-gray-400
                               border border-gray-600 select-none">
                127.0.0.1
              </span>
              <span className="text-gray-600 text-sm">:</span>
              <input
                className="w-20 rounded-lg border border-gray-600 bg-gray-700 px-2 py-1 text-center
                           text-sm font-mono text-gray-100 focus:outline-none focus:ring-2
                           focus:ring-blue-500/50"
                value={listenPort}
                onChange={e => set('listen', `127.0.0.1:${e.target.value}`)}
              />
            </div>
          </Row>

          <div className="px-5 py-3.5 space-y-3">
            <div>
              <p className="text-sm font-medium text-gray-100">Upstream DNS Servers</p>
              <p className="mt-0.5 text-xs text-gray-500">
                Raced concurrently, ordered by live latency. System DNS is the final fallback.
              </p>
            </div>

            <div className="space-y-2">
              {(cfg.upstream ?? []).length === 0 && (
                <p className="py-2 text-xs italic text-gray-600">No upstreams — add one below.</p>
              )}
              {(cfg.upstream ?? []).map((s, i) => (
                <div key={i} className="flex items-center gap-2 group">
                  <span className="flex h-6 w-6 flex-shrink-0 items-center justify-center
                                   rounded bg-gray-700 text-[10px] font-mono text-gray-500
                                   border border-gray-600">
                    {i + 1}
                  </span>
                  <input
                    value={s}
                    onChange={e => {
                      const up = [...(cfg.upstream ?? [])]
                      up[i] = e.target.value
                      set('upstream', up)
                    }}
                    className="flex-1 rounded-lg border border-gray-600 bg-gray-700 px-3 py-1.5
                               text-xs font-mono text-gray-200 focus:outline-none focus:ring-1
                               focus:ring-blue-500/50"
                  />
                  <button
                    onClick={() => removeUpstream(i)}
                    className="flex-shrink-0 text-gray-600 opacity-0 group-hover:opacity-100
                               hover:text-red-400 transition-all"
                  >
                    <Trash2 size={14} />
                  </button>
                </div>
              ))}
            </div>

            <div className="flex gap-2 pt-1">
              <input
                placeholder="9.9.9.9:853 (DoT) or 8.8.8.8:53 (plain)"
                value={newUp}
                onChange={e => setNewUp(e.target.value)}
                onKeyDown={e => e.key === 'Enter' && addUpstream()}
                className="flex-1 rounded-lg border border-dashed border-gray-600 bg-gray-700/60
                           px-3 py-1.5 text-xs font-mono text-gray-300 placeholder-gray-600
                           focus:outline-none focus:ring-1 focus:ring-blue-500/50 focus:border-blue-700"
              />
              <button
                onClick={addUpstream}
                className="flex items-center gap-1.5 rounded-lg border border-gray-600 bg-gray-700
                           px-3 py-1.5 text-xs font-medium text-gray-300 hover:bg-gray-600 transition-colors"
              >
                <Plus size={11} /> Add
              </button>
            </div>
          </div>

          <div className="divide-y divide-gray-700/40">
            <div className="px-5 py-2">
              <p className="text-[10px] font-semibold uppercase tracking-widest text-gray-500">Resolution Order</p>
            </div>
            <ResolutionOrder api={api} />
          </div>
        </Section>

        <Section title="Security" icon={Shield}>
          <Row
            label="DNS-over-TLS"
            desc="Encrypts all upstream queries. Uses TLS 1.2+ with session resumption on port 853."
          >
            <Toggle value={!!cfg.use_tls} onChange={v => set('use_tls', v)} />
          </Row>

          <Row
            label="DNS Rebinding Protection"
            desc="Blocks private-range IPs returned for public domains."
          >
            <Toggle
              value={!!cfg.dns_rebinding_protection}
              onChange={v => set('dns_rebinding_protection', v)}
            />
          </Row>

          <Row label="DNSSEC" desc="Sets the DO bit on upstream queries.">
            <Toggle value={!!cfg.dnssec} onChange={v => set('dnssec', v)} />
          </Row>
        </Section>

        <Section title="Rate Limiting" icon={Gauge}>
          <Row
            label="Enable Rate Limiting"
            desc="Protects against local query floods and rogue processes."
          >
            <Toggle
              value={!!rl.enabled}
              onChange={v => set('rate_limit.enabled', v)}
            />
          </Row>

          {rl.enabled && (<>
            <SliderRow
              label="Max Queries / Second (per IP)"
              desc="Sustained token-refill rate. Burst can briefly exceed this."
              value={rl.max_rps ?? 200}
              min={10} max={2000} step={10} unit=" req/s"
              onChange={v => set('rate_limit.max_rps', v)}
              indented
            />

            <SliderRow
              label={`Burst Multiplier  →  ${((rl.burst_multiplier ?? 3) * (rl.max_rps ?? 200)).toLocaleString()} req burst`}
              desc="How many extra queries an idle client can fire instantly. Higher = more bursty but still fair over time."
              value={rl.burst_multiplier ?? 3}
              min={1} max={10} step={1} unit="×"
              onChange={v => set('rate_limit.burst_multiplier', v)}
              indented
            />

            <SliderRow
              label="Per-Domain Cap"
              desc="Max queries/sec to any single domain regardless of source IP. Set to 0 to disable."
              value={rl.per_domain_max_rps ?? 0}
              min={0} max={500} step={5} unit=" req/s"
              onChange={v => set('rate_limit.per_domain_max_rps', v)}
              indented
            />

            {(rl.per_domain_max_rps ?? 0) === 0 && (
              <div className="px-5 pl-11 py-2 bg-gray-900/40 border-l-2 border-blue-800/30">
                <p className="text-[11px] text-gray-600 italic">
                  Per-domain cap is disabled (0). Raise the slider to limit individual domains.
                </p>
              </div>
            )}

            <IPListEditor
              ips={whitelistIPs}
              onAdd={ip => set('rate_limit.whitelist_ips', [...whitelistIPs, ip])}
              onRemove={i => set('rate_limit.whitelist_ips', whitelistIPs.filter((_, idx) => idx !== i))}
            />
          </>)}
        </Section>

        <Section title="Cache" icon={Database}>
          <Row
            label="Enable Cache"
            desc="Stores DNS responses in memory for sub-millisecond repeat lookups."
          >
            <Toggle value={!!ca.enabled} onChange={v => set('cache.enabled', v)} />
          </Row>

          {ca.enabled && (<>
            <SliderRow
              label="Max Cache Size"
              desc="Total number of entries across all 16 shards."
              value={ca.max_size ?? 10000}
              min={100} max={100000} step={100} unit=" entries"
              onChange={v => set('cache.max_size', v)}
              indented
            />
            <SliderRow
              label="Minimum TTL Floor"
              desc="Prevents zero-TTL entries from flooding upstream. Recommended: 30–120 s."
              value={ca.min_ttl ?? 60}
              min={0} max={3600} step={5} unit="s"
              onChange={v => set('cache.min_ttl', v)}
              indented
            />
            <Row
              label={
                <span className="flex items-center">
                  Stale-While-Revalidate
                  <Hint text="Serve the just-expired cache entry immediately, then refresh it in the background. Eliminates latency spikes at TTL boundaries." />
                </span>
              }
              desc="Zero-latency responses even when TTL has just expired."
              indented
            >
              <Toggle
                value={!!ca.stale_while_revalidate}
                onChange={v => set('cache.stale_while_revalidate', v)}
              />
            </Row>
            <CacheTools api={api} />
          </>)}
        </Section>

        <Section title="Privacy" icon={Eye}>
          <Row
            label="Privacy Mode"
            desc="Anonymises domain names in the query log — only TLD is recorded."
          >
            <Toggle value={!cfg.log_queries} onChange={v => set('log_queries', !v)} />
          </Row>
        </Section>

        <div className="h-4" />
      </div>
    </div>
  )
}

export default function Settings({ api }) {
  return (
    <ErrorBoundary>
      <SettingsInner api={api} />
    </ErrorBoundary>
  )
}
