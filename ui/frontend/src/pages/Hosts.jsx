import { useState, useEffect, useMemo } from 'react'
import {
  Plus, Trash2, Search, Globe, Upload,
  CheckCircle2, AlertTriangle, ArrowRight,
} from 'lucide-react'

export default function Hosts({ api }) {
  const [hosts, setHosts]         = useState([])
  const [domain, setDomain]       = useState('')
  const [ip, setIp]               = useState('')
  const [search, setSearch]       = useState('')
  const [testHost, setTestHost]   = useState('')
  const [testResult, setTestResult] = useState(null)
  const [msg, setMsg]             = useState('')

  const load = async () => {
    try { setHosts(await api.get('/hosts') ?? []) } catch {}
  }
  useEffect(() => { load() }, [api])

  const toast = m => { setMsg(m); setTimeout(() => setMsg(''), 2500) }

  const add = async () => {
    if (!domain.trim() || !ip.trim()) return
    try {
      await api.post('/hosts/add', { domain: domain.trim(), ip: ip.trim() })
      setDomain(''); setIp('')
      await load()
      toast('Host added')
    } catch (e) { toast('Error: ' + e.message) }
  }

  const remove = async d => {
    try { await api.post('/hosts/remove', { domain: d }); await load() } catch {}
  }

  const test = () => {
    if (!testHost.trim()) return
    const m = hosts.find(h => h.domain.toLowerCase() === testHost.trim().toLowerCase())
    setTestResult(m
      ? { domain: testHost.trim(), ip: m.ip, found: true }
      : { domain: testHost.trim(), found: false }
    )
  }

  const filtered = useMemo(() =>
    hosts.filter(h =>
      !search || h.domain.toLowerCase().includes(search.toLowerCase()) || h.ip.includes(search)
    ),
    [hosts, search]
  )

  return (
    <div className="px-7 py-6 space-y-6 max-w-[1400px]">
      <header className="flex items-end justify-between pt-2">
        <div>
          <h1 className="h-page">Hosts</h1>
          <p className="mt-1 text-xs text-text-muted">
            <span className="font-mono">{hosts.length}</span> custom override
            {hosts.length === 1 ? '' : 's'}
          </p>
        </div>
        <button
          className="btn-ghost"
          onClick={() => toast('Use selfdns --import-hosts to import /etc/hosts entries')}
        >
          <Upload size={12} /> Import /etc/hosts
        </button>
      </header>

      {msg && (
        <div className="panel ring-info/30 bg-info/5 px-3 py-2 text-2xs font-mono text-info uppercase tracking-widest animate-fade-in">
          {msg}
        </div>
      )}

      <div className="grid stable-grid-cols-2 gap-3">
        <Section title="Add host override">
          <div className="p-2.5 space-y-2">
            <div className="flex gap-2">
              <input
                className="input-mono flex-1"
                placeholder="myapp.local"
                value={domain}
                onChange={e => setDomain(e.target.value)}
                onKeyDown={e => e.key === 'Enter' && ip && add()}
              />
              <ArrowRight size={12} className="self-center text-text-dim" />
              <input
                className="input-mono w-36"
                placeholder="192.168.1.10"
                value={ip}
                onChange={e => setIp(e.target.value)}
                onKeyDown={e => e.key === 'Enter' && domain && add()}
              />
              <button className="btn-primary" onClick={add}>
                <Plus size={11} /> Add
              </button>
            </div>
            <p className="text-2xs text-text-dim font-mono">
              resolves the domain locally · bypasses upstream
            </p>
          </div>
        </Section>

        <Section title="Test resolution">
          <div className="p-2.5 space-y-2">
            <div className="flex gap-2">
              <input
                className="input-mono"
                placeholder="myapp.local"
                value={testHost}
                onChange={e => { setTestHost(e.target.value); setTestResult(null) }}
                onKeyDown={e => e.key === 'Enter' && test()}
              />
              <button className="btn-ghost" onClick={test}>Resolve</button>
            </div>
            {testResult && (
              <div className={`flex items-center gap-2 text-2xs font-mono ${
                testResult.found ? 'text-accent' : 'text-warn'
              }`}>
                {testResult.found
                  ? <CheckCircle2 size={11} />
                  : <AlertTriangle size={11} />}
                <span>{testResult.domain}</span>
                {testResult.found ? (
                  <>
                    <ArrowRight size={10} className="text-text-dim" />
                    <span>{testResult.ip}</span>
                  </>
                ) : (
                  <span className="text-text-dim uppercase tracking-widest">no override</span>
                )}
              </div>
            )}
          </div>
        </Section>
      </div>

      <Section
        title={`Overrides · ${filtered.length}`}
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
        {filtered.length === 0 ? (
          <div className="py-12 flex flex-col items-center gap-2">
            <Globe size={20} className="text-text-dim" />
            <p className="text-2xs font-mono text-text-dim uppercase tracking-widest">
              no overrides yet
            </p>
            <p className="text-2xs text-text-dim">
              add one above to get started
            </p>
          </div>
        ) : (
          <table className="table-base">
            <thead>
              <tr>
                <th>Domain</th>
                <th>IP address</th>
                <th className="w-10" />
              </tr>
            </thead>
            <tbody>
              {filtered.map(h => (
                <tr key={h.domain} className="group">
                  <td className="font-mono text-text-primary">{h.domain}</td>
                  <td className="font-mono text-info">{h.ip}</td>
                  <td className="!text-right">
                    <button
                      className="text-text-dim hover:text-bad opacity-0 group-hover:opacity-100 transition-all"
                      onClick={() => remove(h.domain)}
                    >
                      <Trash2 size={12} />
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
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
      <div className="panel overflow-hidden">{children}</div>
    </div>
  )
}
