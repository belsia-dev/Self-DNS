import { useState, useEffect } from 'react'
import { Plus, Search, Download, ToggleLeft, ToggleRight, X, FileText, ShieldOff, ShieldCheck } from 'lucide-react'

const POPULAR = [
  { name: 'StevenBlack Unified', url: 'https://raw.githubusercontent.com/StevenBlack/hosts/master/hosts' },
  { name: 'OISD Big', url: 'https://big.oisd.nl/domainswild' },
  { name: 'AdGuard DNS', url: 'https://adguardteam.github.io/AdGuardSDNSFilter/Filters/filter.txt' },
  { name: 'Hagezi Multi-Normal', url: 'https://raw.githubusercontent.com/hagezi/dns-blocklists/main/hosts/multi.txt' },
]

export default function Blocklist({ api }) {
  const [data, setData] = useState({ enabled: false, domains: [], files: [], count: 0 })
  const [search, setSearch] = useState('')
  const [newDomain, setNewDomain] = useState('')
  const [testDomain, setTestDomain] = useState('')
  const [testResult, setTestResult] = useState(null)
  const [msg, setMsg] = useState('')

  const load = async () => {
    try { setData(await api.get('/blocklist')) } catch { }
  }
  useEffect(() => { load() }, [api])

  const toast = (m) => { setMsg(m); setTimeout(() => setMsg(''), 3000) }

  const toggle = async () => {
    try {
      const res = await api.post('/blocklist/toggle', {})
      await load()
      toast('Blocking ' + (res.enabled ? 'enabled' : 'disabled'))
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

  const removeDomain = async (d) => {
    try {
      await api.post('/blocklist/remove', { domain: d })
      await load()
    } catch { }
  }

  const testDom = async () => {
    if (!testDomain.trim()) return
    const blocked = data.domains.includes(testDomain.trim().toLowerCase())
    setTestResult({ domain: testDomain.trim(), blocked })
  }

  const downloadList = (url, name) => {
    toast(`To add "${name}", download and add the file via the file picker (not yet wired for live download).`)
  }

  const filtered = (data.domains ?? []).filter(d =>
    !search || d.toLowerCase().includes(search.toLowerCase())
  )

  return (
    <div className="p-6 space-y-6">
      <div className="flex items-center justify-between pt-2">
        <div>
          <h1 className="text-xl font-bold text-white">Blocklist</h1>
          <p className="text-sm text-gray-400">{data.count.toLocaleString()} domains blocked</p>
        </div>
        <button
          onClick={toggle}
          className={`flex items-center gap-2 px-4 py-2 rounded-lg text-sm font-medium transition-colors ${data.enabled
            ? 'bg-green-900/30 text-green-400 border border-green-700/40 hover:bg-green-900/50'
            : 'bg-gray-700 text-gray-300 hover:bg-gray-600'
            }`}
        >
          {data.enabled ? <ToggleRight size={16} /> : <ToggleLeft size={16} />}
          Blocking {data.enabled ? 'ON' : 'OFF'}
        </button>
      </div>

      {msg && <div className="card border-blue-700/30 bg-blue-900/10 text-blue-300 text-sm">{msg}</div>}

      {data.files?.length > 0 && (
        <div className="card">
          <p className="text-sm font-semibold text-gray-200 mb-3">Loaded Files</p>
          <div className="space-y-2">
            {data.files.map(f => (
              <div key={f.path} className="flex items-center justify-between py-1.5">
                <div className="flex items-center gap-2 min-w-0">
                  <FileText size={14} className="text-gray-500 flex-shrink-0" />
                  <span className="text-xs text-gray-300 font-mono truncate">{f.path}</span>
                </div>
                <div className="flex items-center gap-2 flex-shrink-0 ml-2">
                  <span className="badge-gray">{f.count?.toLocaleString()} entries</span>
                  <span className={f.loaded ? 'badge-green' : 'badge-red'}>
                    {f.loaded ? 'loaded' : 'error'}
                  </span>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      <div className="card">
        <p className="text-sm font-semibold text-gray-200 mb-3">Popular Blocklists</p>
        <div className="grid grid-cols-2 gap-2">
          {POPULAR.map(p => (
            <button key={p.name}
              className="flex items-center gap-2 text-left px-3 py-2 rounded-lg bg-gray-700/50
                         hover:bg-gray-700 text-sm text-gray-300 hover:text-white transition-colors"
              onClick={() => downloadList(p.url, p.name)}
            >
              <Download size={13} className="text-blue-400 flex-shrink-0" />
              <span className="truncate">{p.name}</span>
            </button>
          ))}
        </div>
      </div>

      <div className="card">
        <p className="text-sm font-semibold text-gray-200 mb-3">Add Domain Manually</p>
        <div className="flex gap-2">
          <input className="input" placeholder="ads.example.com"
            value={newDomain} onChange={e => setNewDomain(e.target.value)}
            onKeyDown={e => e.key === 'Enter' && addDomain()} />
          <button className="btn-primary flex items-center gap-1.5" onClick={addDomain}>
            <Plus size={14} /> Add
          </button>
        </div>
      </div>

      <div className="card">
        <p className="text-sm font-semibold text-gray-200 mb-3">Test a Domain</p>
        <div className="flex gap-2">
          <input className="input" placeholder="example.com"
            value={testDomain} onChange={e => { setTestDomain(e.target.value); setTestResult(null) }}
            onKeyDown={e => e.key === 'Enter' && testDom()} />
          <button className="btn-ghost" onClick={testDom}>Test</button>
        </div>
        {testResult && (
          <p className={`mt-2 flex items-center gap-1.5 text-sm ${testResult.blocked ? 'text-red-400' : 'text-green-400'}`}>
            {testResult.blocked ? <ShieldOff size={13} /> : <ShieldCheck size={13} />}
            <span className="font-mono">{testResult.domain}</span>
            <span>is {testResult.blocked ? 'blocked' : 'not blocked'}</span>
          </p>
        )}
      </div>

      <div className="card">
        <div className="flex items-center gap-2 mb-3">
          <p className="text-sm font-semibold text-gray-200 flex-1">Blocked Domains ({filtered.length})</p>
          <div className="relative">
            <Search size={13} className="absolute left-2.5 top-1/2 -translate-y-1/2 text-gray-500" />
            <input className="input pl-8 w-52 text-xs" placeholder="Search…"
              value={search} onChange={e => setSearch(e.target.value)} />
          </div>
        </div>
        <div className="max-h-64 overflow-y-auto space-y-1">
          {filtered.length === 0
            ? <p className="text-xs text-gray-500 py-4 text-center">No domains</p>
            : filtered.map(d => (
              <div key={d} className="flex items-center justify-between py-1 px-2 rounded-lg hover:bg-gray-700/50 group">
                <span className="text-xs font-mono text-gray-300">{d}</span>
                <button className="text-gray-600 hover:text-red-400 opacity-0 group-hover:opacity-100 transition-all"
                  onClick={() => removeDomain(d)}>
                  <X size={13} />
                </button>
              </div>
            ))
          }
        </div>
      </div>
    </div>
  )
}
