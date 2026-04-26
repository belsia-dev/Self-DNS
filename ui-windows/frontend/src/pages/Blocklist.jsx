import { useState, useEffect, useMemo } from 'react'
import {
  Plus, Search, Download, X, FileText,
  ShieldOff, ShieldCheck, Power,
} from 'lucide-react'

const POPULAR = [
  { name: 'StevenBlack Unified',  url: 'https://raw.githubusercontent.com/StevenBlack/hosts/master/hosts' },
  { name: 'OISD Big',              url: 'https://big.oisd.nl/domainswild' },
  { name: 'AdGuard DNS',           url: 'https://adguardteam.github.io/AdGuardSDNSFilter/Filters/filter.txt' },
  { name: 'Hagezi Multi-Normal',   url: 'https://raw.githubusercontent.com/hagezi/dns-blocklists/main/hosts/multi.txt' },
]

export default function Blocklist({ api }) {
  const [data, setData]           = useState({ enabled: false, domains: [], files: [], count: 0 })
  const [search, setSearch]       = useState('')
  const [newDomain, setNewDomain] = useState('')
  const [testDomain, setTestDomain] = useState('')
  const [testResult, setTestResult] = useState(null)
  const [msg, setMsg]             = useState('')

  const load = async () => {
    try { setData(await api.get('/blocklist')) } catch {}
  }
  useEffect(() => { load() }, [api])

  const toast = m => { setMsg(m); setTimeout(() => setMsg(''), 2500) }

  const toggle = async () => {
    try {
      const res = await api.post('/blocklist/toggle', {})
      await load()
      toast(`Blocking · ${res.enabled ? 'on' : 'off'}`)
    } catch (e) { toast('Error: ' + e.message) }
  }

  const addDomain = async () => {
    if (!newDomain.trim()) return
    try {
      await api.post('/blocklist/add', { domain: newDomain.trim() })
      setNewDomain('')
      await load()
      toast('Domain added')
    } catch (e) { toast('Error: ' + e.message) }
  }

  const removeDomain = async d => {
    try { await api.post('/blocklist/remove', { domain: d }); await load() } catch {}
  }

  const testDom = () => {
    if (!testDomain.trim()) return
    const c = testDomain.trim().toLowerCase()
    const blocked = (data.domains ?? []).some(d =>
      c === d.toLowerCase() || c.endsWith(`.${d.toLowerCase()}`)
    )
    setTestResult({ domain: testDomain.trim(), blocked })
  }

  const downloadList = (_, name) =>
    toast(`Download "${name}" via the file picker (live download not yet wired).`)

  const filtered = useMemo(() =>
    (data.domains ?? []).filter(d =>
      !search || d.toLowerCase().includes(search.toLowerCase())
    ),
    [data.domains, search]
  )

  return (
    <div className="px-7 py-6 space-y-6 max-w-[1400px]">
      {/* Header */}
      <header className="flex items-end justify-between pt-2">
        <div>
          <h1 className="h-page">Blocklist</h1>
          <p className="mt-1 text-xs text-text-muted">
            <span className="font-mono">{data.count.toLocaleString()}</span> domains
            <span className="mx-2 text-text-dim">·</span>
            <span className={`font-mono ${data.enabled ? 'text-accent' : 'text-text-muted'}`}>
              {data.enabled ? 'enforcement on' : 'enforcement off'}
            </span>
          </p>
        </div>

        <button
          onClick={toggle}
          className={`inline-flex items-center gap-2 h-8 rounded-md px-3 text-xs font-semibold tracking-tight transition-colors ${
            data.enabled
              ? 'bg-accent/15 text-accent ring-1 ring-accent/40 hover:bg-accent/25'
              : 'bg-ink-300 text-text-secondary ring-1 ring-line/60 hover:bg-ink-400'
          }`}
        >
          <Power size={12} />
          {data.enabled ? 'Blocking · ON' : 'Blocking · OFF'}
        </button>
      </header>

      {msg && (
        <div className="panel ring-accent/30 bg-accent/5 px-3 py-2 text-2xs font-mono text-accent uppercase tracking-widest animate-fade-in">
          {msg}
        </div>
      )}

      {/* Loaded files */}
      {data.files?.length > 0 && (
        <Section title="Loaded blocklist files">
          <div className="divide-line">
            {data.files.map(f => (
              <div key={f.path} className="flex items-center justify-between px-3 h-9">
                <div className="flex items-center gap-2 min-w-0">
                  <FileText size={11} className="text-text-dim flex-shrink-0" />
                  <span className="text-2xs font-mono text-text-secondary truncate">{f.path}</span>
                </div>
                <div className="flex items-center gap-2 flex-shrink-0">
                  <span className="text-2xs font-mono text-text-muted tabular-nums">
                    {f.count?.toLocaleString()}
                  </span>
                  <span className={f.loaded ? 'chip-good' : 'chip-bad'}>
                    {f.loaded ? 'loaded' : 'error'}
                  </span>
                </div>
              </div>
            ))}
          </div>
        </Section>
      )}

      {/* Two-column: popular + add */}
      <div className="grid stable-grid-cols-2 gap-3">
        <Section title="Popular blocklists">
          <div className="grid grid-cols-2 gap-1.5 p-2.5">
            {POPULAR.map(p => (
              <button
                key={p.name}
                onClick={() => downloadList(p.url, p.name)}
                className="flex items-center gap-2 text-left px-2.5 h-8 rounded-md
                           bg-ink-300/40 hover:bg-ink-300 ring-1 ring-line/40
                           text-2xs text-text-secondary hover:text-text-primary
                           transition-colors"
              >
                <Download size={11} className="text-info flex-shrink-0" />
                <span className="truncate">{p.name}</span>
              </button>
            ))}
          </div>
        </Section>

        <Section title="Add domain">
          <div className="p-2.5 space-y-2">
            <div className="flex gap-2">
              <input
                className="input-mono"
                placeholder="ads.example.com"
                value={newDomain}
                onChange={e => setNewDomain(e.target.value)}
                onKeyDown={e => e.key === 'Enter' && addDomain()}
              />
              <button className="btn-primary" onClick={addDomain}>
                <Plus size={11} /> Add
              </button>
            </div>
            <p className="text-2xs text-text-dim font-mono">
              also blocks all subdomains
            </p>
          </div>
        </Section>
      </div>

      {/* Test */}
      <Section title="Test a domain">
        <div className="p-2.5 space-y-2">
          <div className="flex gap-2">
            <input
              className="input-mono"
              placeholder="example.com"
              value={testDomain}
              onChange={e => { setTestDomain(e.target.value); setTestResult(null) }}
              onKeyDown={e => e.key === 'Enter' && testDom()}
            />
            <button className="btn-ghost" onClick={testDom}>
              Test
            </button>
          </div>
          {testResult && (
            <div className={`flex items-center gap-2 text-2xs font-mono ${
              testResult.blocked ? 'text-bad' : 'text-accent'
            }`}>
              {testResult.blocked
                ? <ShieldOff size={11} />
                : <ShieldCheck size={11} />}
              <span>{testResult.domain}</span>
              <span className="text-text-dim">·</span>
              <span className="uppercase tracking-widest">
                {testResult.blocked ? 'blocked' : 'allowed'}
              </span>
            </div>
          )}
        </div>
      </Section>

      {/* Domain list */}
      <Section
        title={`Blocked domains · ${filtered.length}`}
        right={
          <div className="relative">
            <Search size={11} className="absolute left-2 top-1/2 -translate-y-1/2 text-text-dim" />
            <input
              className="input-mono pl-7 w-52 h-7 text-2xs"
              placeholder="search…"
              value={search}
              onChange={e => setSearch(e.target.value)}
            />
          </div>
        }
      >
        <div className="max-h-72 overflow-y-auto">
          {filtered.length === 0 ? (
            <p className="py-10 text-center text-2xs font-mono text-text-dim uppercase tracking-widest">
              no domains
            </p>
          ) : (
            filtered.map(d => (
              <div
                key={d}
                className="group flex items-center justify-between px-3 h-7 row-hover"
              >
                <span className="text-2xs font-mono text-text-secondary">{d}</span>
                <button
                  className="text-text-dim hover:text-bad opacity-0 group-hover:opacity-100 transition-all"
                  onClick={() => removeDomain(d)}
                >
                  <X size={12} />
                </button>
              </div>
            ))
          )}
        </div>
      </Section>
    </div>
  )
}

function Section({ title, right, children }) {
  return (
    <div>
      <div className="flex items-center justify-between mb-1.5 px-1">
        <p className="h-section">{title}</p>
        {right}
      </div>
      <div className="panel">{children}</div>
    </div>
  )
}
