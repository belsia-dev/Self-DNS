import { Component, useState, useEffect, useCallback, useRef } from 'react'
import {
  Save, RefreshCw, Plus, Trash2, CheckCircle2, AlertCircle,
  Shield, Database, Eye, Network, Gauge, Info, Download, Upload, Zap,
  ArrowDown, Globe, Router, Lock, ShieldCheck,
} from 'lucide-react'

class ErrorBoundary extends Component {
  constructor(props) {
    super(props)
    this.state = { hasError: false, error: null }
  }
  static getDerivedStateFromError(error) { return { hasError: true, error } }
  componentDidCatch(error, info) { console.error('[Settings]', error, info.componentStack) }
  render() {
    if (this.state.hasError) {
      return (
        <div className="flex flex-col items-center justify-center h-full gap-4 px-8">
          <div className="flex items-center gap-2 text-bad">
            <AlertCircle size={16} />
            <p className="text-sm font-semibold">Settings crashed</p>
          </div>
          <pre className="text-2xs font-mono text-text-muted bg-ink-100 rounded-md px-4 py-3 max-w-lg overflow-auto ring-1 ring-line">
            {String(this.state.error)}
          </pre>
          <button className="btn-primary" onClick={() => this.setState({ hasError: false, error: null })}>
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
      className={`relative inline-flex h-5 w-9 flex-shrink-0 cursor-pointer rounded-full
                  border border-line/60 transition-colors focus:outline-none
                  focus:ring-2 focus:ring-accent/40 ${value ? 'bg-accent' : 'bg-ink-300'}`}
    >
      <span
        className={`pointer-events-none inline-block h-3.5 w-3.5 rounded-full
                    bg-ink-50 shadow ring-0 transition-transform duration-150
                    ${value ? 'translate-x-[18px]' : 'translate-x-[2px]'}
                    self-center`}
      />
    </button>
  )
}

function Section({ title, icon: Icon, children }) {
  return (
    <section className="space-y-2">
      <div className="flex items-center gap-1.5 px-1">
        {Icon && <Icon size={11} className="text-text-dim" />}
        <p className="h-section">{title}</p>
      </div>
      <div className="panel divide-y divide-line/40 overflow-hidden">
        {children}
      </div>
    </section>
  )
}

function Row({ label, desc, children, indented = false }) {
  return (
    <div className={`flex items-center justify-between gap-5 px-4 py-3 ${
      indented ? 'pl-8 bg-ink-100/40' : ''
    }`}>
      <div className="flex-1 min-w-0">
        <p className={`text-xs font-medium tracking-tight ${
          indented ? 'text-text-secondary' : 'text-text-primary'
        }`}>{label}</p>
        {desc && <p className="mt-0.5 text-2xs text-text-muted leading-relaxed">{desc}</p>}
      </div>
      <div className="flex-shrink-0">{children}</div>
    </div>
  )
}

function Hint({ text }) {
  return (
    <span className="group relative inline-flex items-center ml-1 cursor-help align-middle">
      <Info size={10} className="text-text-dim hover:text-text-secondary transition-colors" />
      <span className="pointer-events-none absolute left-full ml-2 top-1/2 -translate-y-1/2
                       hidden group-hover:flex z-50 panel px-3 py-2 text-2xs
                       text-text-secondary w-56 leading-relaxed">
        {text}
      </span>
    </span>
  )
}

function Segmented({ value, onChange, options, indented = false }) {
  return (
    <div className={`px-4 py-3 ${indented ? 'pl-8 bg-ink-100/40' : ''}`}>
      <div className="flex flex-wrap gap-1.5">
        {options.map(o => {
          const active = value === o.value
          return (
            <button
              key={o.value}
              onClick={() => onChange(o.value)}
              className={`min-w-[140px] rounded-md px-3 py-2 text-left transition-colors ring-1
                          ${active
                            ? 'ring-accent/50 bg-accent/10 text-accent'
                            : 'ring-line/60 bg-ink-300/40 text-text-secondary hover:bg-ink-300 hover:text-text-primary'}`}
            >
              <p className="text-2xs font-semibold uppercase tracking-widest">{o.label}</p>
              {o.note && (
                <p className={`mt-1 text-2xs leading-relaxed ${
                  active ? 'text-accent/70' : 'text-text-dim'
                }`}>
                  {o.note}
                </p>
              )}
            </button>
          )
        })}
      </div>
    </div>
  )
}

function TextareaBlock({ label, desc, value, onChange, rows = 6, indented = false }) {
  return (
    <div className={`px-4 py-3 ${indented ? 'pl-8 bg-ink-100/40' : ''}`}>
      <div className="mb-2">
        <p className={`text-xs font-medium ${
          indented ? 'text-text-secondary' : 'text-text-primary'
        }`}>{label}</p>
        {desc && <p className="mt-0.5 text-2xs text-text-muted leading-relaxed">{desc}</p>}
      </div>
      <textarea
        rows={rows}
        value={value}
        onChange={e => onChange(e.target.value)}
        spellCheck={false}
        className="w-full rounded-md bg-ink-300/60 ring-1 ring-line px-3 py-2.5
                   text-2xs font-mono text-text-secondary leading-relaxed
                   placeholder:text-text-dim focus:outline-none focus:ring-2 focus:ring-accent/40"
      />
    </div>
  )
}

function Slider({ label, desc, value, onChange, min, max, unit = '', step = 1, indented = false }) {
  return (
    <div className={`px-4 py-3 ${indented ? 'pl-8 bg-ink-100/40' : ''}`}>
      <div className="flex items-center justify-between mb-2">
        <div className="min-w-0">
          <p className={`text-xs font-medium ${
            indented ? 'text-text-secondary' : 'text-text-primary'
          }`}>{label}</p>
          {desc && <p className="mt-0.5 text-2xs text-text-muted">{desc}</p>}
        </div>
        <span className="font-mono text-2xs tabular-nums text-accent bg-accent/10 ring-1 ring-accent/25
                         px-2 h-6 inline-flex items-center rounded min-w-[64px] justify-center">
          {Number(value).toLocaleString()}{unit}
        </span>
      </div>
      <input
        type="range" min={min} max={max} step={step} value={value}
        onChange={e => onChange(Number(e.target.value))}
        className="w-full h-1 appearance-none rounded-full bg-ink-300 accent-accent cursor-pointer"
      />
      <div className="mt-1 flex justify-between text-2xs font-mono text-text-dim">
        <span>{Number(min).toLocaleString()}{unit}</span>
        <span>{Number(max).toLocaleString()}{unit}</span>
      </div>
    </div>
  )
}

function IPListEditor({ ips, onAdd, onRemove }) {
  const [val, setVal] = useState('')
  const add = () => {
    const s = val.trim()
    if (!s) return
    onAdd(s); setVal('')
  }
  return (
    <div className="px-4 py-3 pl-8 bg-ink-100/40 space-y-2">
      <p className="text-2xs font-medium text-text-secondary">
        Whitelisted IPs
        <Hint text="These source IPs are never rate-limited, even during a flood." />
      </p>

      {ips.length === 0 && (
        <p className="text-2xs italic text-text-dim font-mono">no whitelisted IPs</p>
      )}
      {ips.map((ip, i) => (
        <div key={i} className="flex items-center gap-2 group">
          <span className="flex-1 rounded-md bg-ink-300/40 ring-1 ring-line/60 px-3 h-7
                           inline-flex items-center text-2xs font-mono text-text-secondary">
            {ip}
          </span>
          <button
            onClick={() => onRemove(i)}
            className="text-text-dim opacity-0 group-hover:opacity-100 hover:text-bad transition-all"
          >
            <Trash2 size={12} />
          </button>
        </div>
      ))}

      <div className="flex gap-2 pt-1">
        <input
          placeholder="192.168.1.0/24 or 10.0.0.5"
          value={val}
          onChange={e => setVal(e.target.value)}
          onKeyDown={e => e.key === 'Enter' && add()}
          className="input-mono flex-1"
        />
        <button onClick={add} className="btn-ghost">
          <Plus size={11} /> Add
        </button>
      </div>
    </div>
  )
}

function CacheTools({ api }) {
  const [msg, setMsg] = useState('')
  const say = t => { setMsg(t); setTimeout(() => setMsg(''), 2500) }

  const exportCache = async () => {
    try {
      const data = await api.get('/cache/export')
      const blob = new Blob([JSON.stringify(data, null, 2)], { type: 'application/json' })
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url; a.download = 'selfdns-cache.json'; a.click()
      URL.revokeObjectURL(url)
      say(`exported ${data.entries?.length ?? 0} entries`)
    } catch (e) { say('export failed: ' + e.message) }
  }

  const importCache = async file => {
    if (!file) return
    try {
      const text = await file.text()
      const res = await api.post('/cache/import', JSON.parse(text))
      say(`imported ${res.entries} entries`)
    } catch (e) { say('import failed: ' + e.message) }
  }

  const prefetch = async () => {
    try { await api.post('/prefetch/run', {}); say('prefetching hot domains…') }
    catch (e) { say('prefetch failed: ' + e.message) }
  }

  const flush = async () => {
    try { await api.post('/cache/flush', {}); say('cache flushed') }
    catch (e) { say('flush failed: ' + e.message) }
  }

  return (
    <div className="px-4 py-3 pl-8 bg-ink-100/40">
      <div className="flex flex-wrap items-center gap-1.5">
        <button className="btn-ghost" onClick={exportCache}>
          <Download size={11} /> Export
        </button>
        <label className="btn-ghost cursor-pointer">
          <Upload size={11} /> Import
          <input type="file" accept="application/json" className="hidden"
                 onChange={e => importCache(e.target.files?.[0])} />
        </label>
        <button className="btn-ghost" onClick={prefetch}>
          <Zap size={11} /> Prefetch hot
        </button>
        <button className="btn-danger" onClick={flush}>
          <Trash2 size={11} /> Flush
        </button>
        {msg && (
          <span className="text-2xs font-mono text-accent uppercase tracking-widest ml-1 animate-fade-in">
            {msg}
          </span>
        )}
      </div>
    </div>
  )
}

function ResolutionOrder({ api }) {
  const [data, setData] = useState(null)
  const [refreshing, setRefreshing] = useState(false)

  const load = useCallback(async () => {
    try { setData(await api.get('/network-dns')) } catch (err) { console.error('[Settings]:', err) }
  }, [api])
  useEffect(() => { load() }, [load])

  const refresh = async () => {
    setRefreshing(true)
    try { await api.post('/network-dns', {}); setTimeout(load, 800) } catch (err) { console.error('[Settings]:', err) }
    setRefreshing(false)
  }

  const sysDNS = (data?.system_dns ?? []).filter(s => !s.startsWith('127.') && !s.startsWith('[::1]'))
  const netDNS = data?.network_dns ?? []

  const tiers = [
    {
      label: 'System DNS',  sublabel: '/etc/resolv.conf',
      servers: sysDNS, icon: Globe, tone: 'info', badge: 'plain UDP',
      empty: 'none detected',
    },
    {
      label: 'Router · DHCP', sublabel: 'Wi-Fi gateway',
      servers: netDNS, icon: Router, tone: 'warn', badge: 'plain UDP',
      empty: 'none detected',
    },
    {
      label: 'DoT upstreams', sublabel: 'configured · last resort',
      servers: null, icon: Lock, tone: 'violet', badge: 'DNS-over-TLS',
    },
  ]

  return (
    <div className="px-4 py-4 space-y-2">
      <div className="flex items-center justify-between mb-2">
        <p className="text-2xs text-text-muted leading-relaxed">
          Queries resolve top-to-bottom · lower tiers fire only if upper ones fail
        </p>
        <button
          onClick={refresh}
          disabled={refreshing}
          className="text-2xs text-text-muted hover:text-text-primary transition-colors flex items-center gap-1.5 disabled:opacity-40"
        >
          <RefreshCw size={10} className={refreshing ? 'animate-spin' : ''} />
          Re-detect
        </button>
      </div>

      {tiers.map((tier, idx) => {
        const Icon = tier.icon
        const chipClass = {
          info: 'chip-info',
          warn: 'chip-warn',
          violet: 'chip-violet',
        }[tier.tone]
        return (
          <div key={tier.label}>
            <div className="rounded-md ring-1 ring-line/60 bg-ink-300/30 overflow-hidden">
              <div className="flex items-center gap-2.5 px-3 py-2">
                <span className="font-mono text-2xs text-text-dim w-3 text-center">{idx + 1}</span>
                <Icon size={12} className="text-text-secondary" />
                <div className="flex-1 min-w-0">
                  <p className="text-xs font-medium text-text-primary">{tier.label}</p>
                  <p className="text-2xs text-text-dim font-mono">{tier.sublabel}</p>
                </div>
                <span className={chipClass}>{tier.badge}</span>
              </div>

              {tier.servers !== null && (
                <div className="px-3 pb-2 pt-0 space-y-1">
                  {tier.servers.length === 0 ? (
                    <p className="text-2xs italic text-text-dim font-mono pl-7">{tier.empty}</p>
                  ) : tier.servers.map(s => (
                    <div key={s} className="flex items-center gap-2 pl-7">
                      <span className="dot-mute" />
                      <span className="text-2xs font-mono text-text-secondary">{s}</span>
                    </div>
                  ))}
                </div>
              )}

              {tier.servers === null && (
                <p className="text-2xs italic text-text-dim font-mono px-3 pb-2 pl-10">
                  configure above in Upstream DNS Servers
                </p>
              )}
            </div>

            {idx < tiers.length - 1 && (
              <div className="flex justify-center py-0.5">
                <ArrowDown size={11} className="text-text-dim" />
              </div>
            )}
          </div>
        )
      })}
    </div>
  )
}

function listenHostForMode(mode) { return mode === 'local' ? '127.0.0.1' : '0.0.0.0' }
function defaultBindForMode(mode) { return `${listenHostForMode(mode)}:80` }

function blockPagePayload(cfg) {
  const blocklist = cfg?.blocklist ?? {}
  const page = blocklist.block_page ?? {}
  return {
    response_mode: blocklist.response_mode ?? 'nxdomain',
    block_page: {
      bind: page.bind ?? '',
      ipv4: page.ipv4 ?? '',
      ipv6: page.ipv6 ?? '',
      html: page.html ?? '',
      css:  page.css ?? '',
      js:   page.js ?? '',
    },
  }
}

function networkBindingKey(cfg) {
  return JSON.stringify({
    service_mode: cfg?.service_mode ?? 'local',
    listen: cfg?.listen ?? '',
  })
}

function SettingsInner({ api }) {
  const [cfg, setCfg]               = useState(null)
  const [saving, setSaving]         = useState(false)
  const [saveResult, setSaveResult] = useState(null)
  const [liveResult, setLiveResult] = useState(null)
  const [liveError, setLiveError]   = useState('')
  const [startMsg, setStartMsg]     = useState('connecting to DNS server')
  const [newUp, setNewUp]           = useState('')
  const lastLivePayloadRef    = useRef('')
  const lastNetworkBindingRef = useRef('')
  const liveApplyTicketRef    = useRef(0)

  useEffect(() => {
    let mounted = true
    let timer
    let attempts = 0
    const messages = [
      'connecting to DNS server',
      'starting DNS server',
      'waiting for DNS server',
      'almost ready',
    ]
    const tryLoad = async () => {
      try {
        const data = await api.get('/config')
        if (mounted) {
          setCfg(data)
          lastLivePayloadRef.current = JSON.stringify(blockPagePayload(data))
          lastNetworkBindingRef.current = networkBindingKey(data)
        }
      } catch (err) {
        console.error('[Settings]:', err)
        if (!mounted) return
        attempts++
        setStartMsg(messages[Math.min(attempts - 1, messages.length - 1)])
        timer = setTimeout(tryLoad, 600)
      }
    }
    tryLoad()
    return () => { mounted = false; clearTimeout(timer) }
  }, [api])

  useEffect(() => {
    if (!cfg || saving) return

    const payload = blockPagePayload(cfg)
    const payloadKey = JSON.stringify(payload)
    const bindingKey = networkBindingKey(cfg)

    if (!lastLivePayloadRef.current) {
      lastLivePayloadRef.current = payloadKey
      lastNetworkBindingRef.current = bindingKey
      return
    }
    if (payloadKey === lastLivePayloadRef.current) return
    if (bindingKey !== lastNetworkBindingRef.current) return

    const ticket = ++liveApplyTicketRef.current
    setLiveResult('saving')
    setLiveError('')

    const timer = setTimeout(async () => {
      try {
        const res = await api.post('/config/block-page', payload)
        if (liveApplyTicketRef.current !== ticket) return
        const applied = {
          response_mode: res.response_mode ?? payload.response_mode,
          block_page: res.block_page ?? payload.block_page,
        }
        lastLivePayloadRef.current = JSON.stringify(applied)
        setCfg(prev => prev ? ({
          ...prev,
          blocklist: { ...(prev.blocklist ?? {}), response_mode: applied.response_mode, block_page: applied.block_page },
        }) : prev)
        setLiveResult('ok')
        setTimeout(() => {
          if (liveApplyTicketRef.current === ticket) {
            setLiveResult(p => p === 'ok' ? null : p)
          }
        }, 2500)
      } catch (e) {
        if (liveApplyTicketRef.current !== ticket) return
        setLiveResult('err')
        setLiveError(e.message)
      }
    }, 600)

    return () => clearTimeout(timer)
  }, [api, cfg, saving])

  const set = (path, value) => {
    setCfg(prev => {
      if (prev == null) return prev
      const next = JSON.parse(JSON.stringify(prev))
      const keys = path.split('.')
      let cur = next
      for (let i = 0; i < keys.length - 1; i++) {
        if (cur[keys[i]] == null || typeof cur[keys[i]] !== 'object') cur[keys[i]] = {}
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
      lastLivePayloadRef.current = JSON.stringify(blockPagePayload(cfg))
      lastNetworkBindingRef.current = networkBindingKey(cfg)
    } catch (err) {
      console.error('[Settings]:', err)
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
  const removeUpstream = i => set('upstream', (cfg.upstream ?? []).filter((_, idx) => idx !== i))

  if (!cfg) {
    return (
      <div className="flex flex-col items-center justify-center h-full gap-4">
        <div className="w-8 h-8 rounded-full border-2 border-accent border-t-transparent animate-spin" />
        <div className="text-center">
          <p className="text-xs font-mono uppercase tracking-widest text-text-secondary">{startMsg}</p>
          <p className="mt-1 text-2xs text-text-dim font-mono">
            port 53 needs root · sudo selfdns-app
          </p>
        </div>
      </div>
    )
  }

  const rl = cfg.rate_limit ?? {}
  const ca = cfg.cache ?? {}
  const blocklist = cfg.blocklist ?? {}
  const blockPage = blocklist.block_page ?? {}
  const whitelistIPs = rl.whitelist_ips ?? []
  const serviceMode = cfg.service_mode ?? 'local'
  const listenHost = listenHostForMode(serviceMode)
  const listenPort = (cfg.listen ?? `${listenHost}:53`).split(':').pop()
  const responseMode = blocklist.response_mode ?? 'nxdomain'
  const tracksDefaultBind = ['local', 'internal', 'external']
    .map(defaultBindForMode)
    .includes(blockPage.bind ?? '')

  const setServiceMode = mode => {
    set('service_mode', mode)
    set('listen', `${listenHostForMode(mode)}:${listenPort || '53'}`)
    if (!blockPage.bind || tracksDefaultBind) {
      set('blocklist.block_page.bind', defaultBindForMode(mode))
    }
  }
  const setListenPort = port => set('listen', `${listenHost}:${port}`)

  const downloadCA = () => {
    const a = document.createElement('a')
    a.href = 'http://127.0.0.1:5380/api/ca-cert'
    a.download = 'selfdns-ca.crt'
    a.click()
  }

  return (
    <div className="flex flex-col h-full overflow-hidden">
      
      <header className="flex-shrink-0 flex items-center justify-between px-7 py-4
                         border-b border-line/60 bg-ink-50/80 backdrop-blur-sm">
        <div>
          <h1 className="h-page">Settings</h1>
          <p className="mt-0.5 text-2xs text-text-muted font-mono">
            block page changes auto-apply · network changes need save + restart
          </p>
        </div>

        <div className="flex items-center gap-3">
          {liveResult === 'saving' && (
            <span className="flex items-center gap-1.5 text-2xs font-mono text-info uppercase tracking-widest">
              <RefreshCw size={11} className="animate-spin" /> applying
            </span>
          )}
          {liveResult === 'ok' && (
            <span className="flex items-center gap-1.5 text-2xs font-mono text-accent uppercase tracking-widest animate-fade-in">
              <CheckCircle2 size={11} /> live
            </span>
          )}
          {liveResult === 'err' && (
            <span className="flex items-center gap-1.5 text-2xs font-mono text-bad uppercase tracking-widest"
                  title={liveError || 'apply failed'}>
              <AlertCircle size={11} /> apply failed
            </span>
          )}
          {saveResult === 'ok' && (
            <span className="flex items-center gap-1.5 text-2xs font-mono text-accent uppercase tracking-widest animate-fade-in">
              <CheckCircle2 size={11} /> saved · restarted
            </span>
          )}
          {saveResult === 'err' && (
            <span className="flex items-center gap-1.5 text-2xs font-mono text-bad uppercase tracking-widest">
              <AlertCircle size={11} /> save failed
            </span>
          )}
          <button onClick={save} disabled={saving} className="btn-primary min-w-[140px]">
            {saving
              ? <><RefreshCw size={11} className="animate-spin" /> Saving</>
              : <><Save size={11} /> Save + Restart</>}
          </button>
        </div>
      </header>

      <div className="flex-1 overflow-y-auto px-7 py-6 space-y-5 max-w-[1100px]">

        <Section title="Network" icon={Network}>
          <div className="px-4 py-4 space-y-3">
            <div>
              <p className="text-xs font-medium text-text-primary">Public service</p>
              <p className="mt-0.5 text-2xs text-text-muted leading-relaxed">
                Whether SelfDNS stays local, serves your LAN, or is exposed to the internet.
              </p>
            </div>

            <Segmented
              value={serviceMode}
              onChange={setServiceMode}
              options={[
                { value: 'local',    label: 'Local',    note: 'binds to 127.0.0.1' },
                { value: 'internal', label: 'Internal', note: 'binds to 0.0.0.0 · LAN' },
                { value: 'external', label: 'External', note: 'binds to 0.0.0.0 · public' },
              ]}
            />

            <div className="rounded-md ring-1 ring-warn/30 bg-warn/5 px-3 py-2 text-2xs text-warn/80 leading-relaxed">
              <code className="text-warn">internal</code> and <code className="text-warn">external</code>
              {' '}both bind 0.0.0.0 · external means you intend to expose port 53 publicly
            </div>
          </div>

          <Row label="DNS listen port" desc="host derived from service mode">
            <div className="flex items-center gap-1.5">
              <span className="px-2 h-7 inline-flex items-center rounded-md bg-ink-300/40 text-2xs font-mono
                               text-text-secondary ring-1 ring-line/60 select-none">
                {listenHost}
              </span>
              <span className="text-text-dim text-xs">:</span>
              <input
                className="input-mono w-16 text-center"
                value={listenPort}
                onChange={e => setListenPort(e.target.value)}
              />
            </div>
          </Row>

          <div className="px-4 py-4 space-y-3">
            <div>
              <p className="text-xs font-medium text-text-primary">Upstream DNS servers</p>
              <p className="mt-0.5 text-2xs text-text-muted">
                Raced concurrently · ordered by live latency · system DNS is last resort.
              </p>
            </div>

            <div className="space-y-1.5">
              {(cfg.upstream ?? []).length === 0 && (
                <p className="py-2 text-2xs italic text-text-dim font-mono">no upstreams</p>
              )}
              {(cfg.upstream ?? []).map((s, i) => (
                <div key={i} className="flex items-center gap-2 group">
                  <span className="flex h-7 w-7 flex-shrink-0 items-center justify-center
                                   rounded bg-ink-300/40 text-2xs font-mono text-text-dim
                                   ring-1 ring-line/60">
                    {i + 1}
                  </span>
                  <input
                    value={s}
                    onChange={e => {
                      const up = [...(cfg.upstream ?? [])]
                      up[i] = e.target.value
                      set('upstream', up)
                    }}
                    className="input-mono flex-1"
                  />
                  <button
                    onClick={() => removeUpstream(i)}
                    className="text-text-dim opacity-0 group-hover:opacity-100 hover:text-bad transition-all"
                  >
                    <Trash2 size={12} />
                  </button>
                </div>
              ))}
            </div>

            <div className="flex gap-2">
              <input
                placeholder="9.9.9.9:853 (DoT) or 8.8.8.8:53"
                value={newUp}
                onChange={e => setNewUp(e.target.value)}
                onKeyDown={e => e.key === 'Enter' && addUpstream()}
                className="input-mono flex-1"
              />
              <button onClick={addUpstream} className="btn-ghost">
                <Plus size={11} /> Add
              </button>
            </div>
          </div>

          <ResolutionOrder api={api} />
        </Section>

        <Section title="Security" icon={Shield}>
          <Row label="DNS-over-TLS" desc="encrypts upstream queries · TLS 1.2+ on port 853">
            <Toggle value={!!cfg.use_tls} onChange={v => set('use_tls', v)} />
          </Row>
          <Row label="DNS rebinding protection" desc="blocks private-range IPs returned for public domains">
            <Toggle value={!!cfg.dns_rebinding_protection} onChange={v => set('dns_rebinding_protection', v)} />
          </Row>
          <Row label="DNSSEC" desc="sets the DO bit on upstream queries">
            <Toggle value={!!cfg.dnssec} onChange={v => set('dnssec', v)} />
          </Row>
        </Section>

        <Section title="Blocked response" icon={Shield}>
          <div className="px-4 py-4 space-y-3">
            <div>
              <p className="text-xs font-medium text-text-primary">When a blocked domain is queried</p>
              <p className="mt-0.5 text-2xs text-text-muted leading-relaxed">
                Return a normal NXDOMAIN, or redirect to a custom HTML block page. Auto-applies after you stop typing.
              </p>
            </div>

            <Segmented
              value={responseMode}
              onChange={v => set('blocklist.response_mode', v)}
              options={[
                { value: 'nxdomain',   label: 'NXDOMAIN',   note: 'standard DNS name error' },
                { value: 'block_page', label: 'Block page', note: 'redirect to local HTTP page' },
              ]}
            />

            {responseMode === 'block_page' && (
              <>
                <div className="rounded-md ring-1 ring-info/30 bg-info/5 px-3 py-2 text-2xs leading-relaxed text-info/80">
                  HTTPS sites need the local CA cert installed in the system trust store. SelfDNS auto-installs it on launch · restart your browser to pick it up.
                  <button onClick={downloadCA} className="btn-ghost mt-2">
                    <Download size={11} /> Download CA cert
                  </button>
                </div>

                {liveResult === 'err' && (
                  <div className="rounded-md ring-1 ring-bad/30 bg-bad/5 px-3 py-2 text-2xs leading-relaxed text-bad/80 font-mono">
                    apply failed{liveError ? ` · ${liveError}` : ''}
                  </div>
                )}

                {networkBindingKey(cfg) !== lastNetworkBindingRef.current && (
                  <div className="rounded-md ring-1 ring-warn/30 bg-warn/5 px-3 py-2 text-2xs leading-relaxed text-warn/80">
                    pending bind change · save + restart to apply network change first
                  </div>
                )}
              </>
            )}
          </div>

          {responseMode === 'block_page' && (<>
            <Row label="Block page bind" desc="HTTP server hosting the page · use port 80 for browser intercept" indented>
              <input
                value={blockPage.bind ?? defaultBindForMode(serviceMode)}
                onChange={e => set('blocklist.block_page.bind', e.target.value)}
                className="input-mono w-44"
              />
            </Row>
            <Row label="Reply IPv4 override" desc="optional · defaults to interface IP" indented>
              <input
                value={blockPage.ipv4 ?? ''}
                onChange={e => set('blocklist.block_page.ipv4', e.target.value)}
                placeholder="203.0.113.10"
                className="input-mono w-44"
              />
            </Row>
            <Row label="Reply IPv6 override" desc="optional · defaults to interface IPv6" indented>
              <input
                value={blockPage.ipv6 ?? ''}
                onChange={e => set('blocklist.block_page.ipv6', e.target.value)}
                placeholder="2001:db8::10"
                className="input-mono w-56"
              />
            </Row>

            <div className="px-4 pl-8 py-2 bg-ink-100/40">
              <p className="text-2xs text-text-muted leading-relaxed font-mono">
                tokens: {'{{DOMAIN}}'} {'{{HOST}}'} {'{{PATH}}'} {'{{REQUEST_URI}}'} {'{{METHOD}}'} {'{{GENERATED_AT}}'}
              </p>
            </div>

            <TextareaBlock
              label="HTML"
              desc="body markup · tokens are replaced before the page is served"
              value={blockPage.html ?? ''}
              onChange={v => set('blocklist.block_page.html', v)}
              rows={10}
              indented
            />
            <TextareaBlock
              label="CSS"
              desc="injected into the page <style> tag"
              value={blockPage.css ?? ''}
              onChange={v => set('blocklist.block_page.css', v)}
              rows={12}
              indented
            />
            <TextareaBlock
              label="JavaScript"
              desc="runs after the page loads"
              value={blockPage.js ?? ''}
              onChange={v => set('blocklist.block_page.js', v)}
              rows={6}
              indented
            />
          </>)}
        </Section>

        <Section title="Rate limiting" icon={Gauge}>
          <Row label="Enable rate limiting" desc="protects against query floods and rogue processes">
            <Toggle value={!!rl.enabled} onChange={v => set('rate_limit.enabled', v)} />
          </Row>

          {rl.enabled && (<>
            <Slider
              label="Max queries / sec (per IP)"
              desc="sustained refill rate · burst can briefly exceed"
              value={rl.max_rps ?? 200}
              min={10} max={2000} step={10} unit=" req/s"
              onChange={v => set('rate_limit.max_rps', v)}
              indented
            />
            <Slider
              label={`Burst multiplier · ${((rl.burst_multiplier ?? 3) * (rl.max_rps ?? 200)).toLocaleString()} req burst`}
              desc="extra queries an idle client can fire instantly"
              value={rl.burst_multiplier ?? 3}
              min={1} max={10} step={1} unit="×"
              onChange={v => set('rate_limit.burst_multiplier', v)}
              indented
            />
            <Slider
              label="Per-domain cap"
              desc="max queries/sec to any single domain · 0 disables"
              value={rl.per_domain_max_rps ?? 0}
              min={0} max={500} step={5} unit=" req/s"
              onChange={v => set('rate_limit.per_domain_max_rps', v)}
              indented
            />
            {(rl.per_domain_max_rps ?? 0) === 0 && (
              <div className="px-4 pl-8 py-2 bg-ink-100/40">
                <p className="text-2xs text-text-dim italic font-mono">
                  per-domain cap disabled (0)
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
          <Row label="Enable cache" desc="sub-millisecond repeat lookups">
            <Toggle value={!!ca.enabled} onChange={v => set('cache.enabled', v)} />
          </Row>

          {ca.enabled && (<>
            <Slider
              label="Max cache size"
              desc="total entries across 16 shards"
              value={ca.max_size ?? 10000}
              min={100} max={100000} step={100} unit=" entries"
              onChange={v => set('cache.max_size', v)}
              indented
            />
            <Slider
              label="Min TTL floor"
              desc="prevents zero-TTL flooding · 30–120s recommended"
              value={ca.min_ttl ?? 60}
              min={0} max={3600} step={5} unit="s"
              onChange={v => set('cache.min_ttl', v)}
              indented
            />
            <Row
              label={<>Stale-while-revalidate <Hint text="Serve just-expired entry immediately, refresh in background. Eliminates latency spikes at TTL boundaries." /></>}
              desc="zero-latency responses across TTL boundaries"
              indented
            >
              <Toggle value={!!ca.stale_while_revalidate} onChange={v => set('cache.stale_while_revalidate', v)} />
            </Row>
            <CacheTools api={api} />
          </>)}
        </Section>

        <Section title="Privacy" icon={Eye}>
          <Row label="Privacy mode" desc="anonymises domains in the query log · only TLD recorded">
            <Toggle value={!cfg.log_queries} onChange={v => set('log_queries', !v)} />
          </Row>
        </Section>

        <div className="h-2" />
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
