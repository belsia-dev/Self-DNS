import { useState, useEffect } from 'react'
import { Plus, Trash2, Search, Globe, Upload, CheckCircle, AlertTriangle } from 'lucide-react'

export default function Hosts({ api }) {
  const [hosts, setHosts] = useState([])
  const [domain, setDomain] = useState('')
  const [ip, setIp] = useState('')
  const [search, setSearch] = useState('')
  const [testHost, setTestHost] = useState('')
  const [testResult, setTestResult] = useState(null)
  const [msg, setMsg] = useState('')

  const load = async () => {
    try { setHosts(await api.get('/hosts') ?? []) } catch { }
  }
  useEffect(() => { load() }, [api])

  const toast = (m) => { setMsg(m); setTimeout(() => setMsg(''), 3000) }

  const add = async () => {
    if (!domain.trim() || !ip.trim()) return
    try {
      await api.post('/hosts/add', { domain: domain.trim(), ip: ip.trim() })
      setDomain(''); setIp('')
      await load()
      toast('Host added')
    } catch (e) { toast('Error: ' + e.message) }
  }

  const remove = async (d) => {
    try {
      await api.post('/hosts/remove', { domain: d })
      await load()
    } catch { }
  }

  const test = async () => {
    if (!testHost.trim()) return
    const match = hosts.find(h => h.domain.toLowerCase() === testHost.trim().toLowerCase())
    if (match) {
      setTestResult({ domain: testHost.trim(), ip: match.ip, found: true })
    } else {
      setTestResult({ domain: testHost.trim(), found: false })
    }
  }

  const importHosts = () => {
    toast('Import from /etc/hosts: use the command-line tool "selfdns --import-hosts" or paste entries above.')
  }

  const filtered = hosts.filter(h =>
    !search || h.domain.toLowerCase().includes(search.toLowerCase()) || h.ip.includes(search)
  )

  return (
    <div className="p-6 space-y-6">
      <div className="flex items-center justify-between pt-2">
        <div>
          <h1 className="text-xl font-bold text-white">Custom Hosts</h1>
          <p className="text-sm text-gray-400">{hosts.length} custom domain overrides</p>
        </div>
        <button className="btn-ghost flex items-center gap-2" onClick={importHosts}>
          <Upload size={14} /> Import /etc/hosts
        </button>
      </div>

      {msg && <div className="card border-blue-700/30 bg-blue-900/10 text-blue-300 text-sm">{msg}</div>}

      <div className="card">
        <p className="text-sm font-semibold text-gray-200 mb-3">Add Host Override</p>
        <div className="flex gap-2">
          <input className="input" placeholder="myapp.local"
            value={domain} onChange={e => setDomain(e.target.value)}
            onKeyDown={e => e.key === 'Enter' && ip && add()} />
          <input className="input w-40" placeholder="192.168.1.10"
            value={ip} onChange={e => setIp(e.target.value)}
            onKeyDown={e => e.key === 'Enter' && domain && add()} />
          <button className="btn-primary flex items-center gap-1.5" onClick={add}>
            <Plus size={14} /> Add
          </button>
        </div>
        <p className="text-xs text-gray-500 mt-2">
          Resolves <code className="text-gray-400">myapp.local</code> → <code className="text-gray-400">127.0.0.1</code> locally, bypassing upstream DNS.
        </p>
      </div>

      <div className="card">
        <p className="text-sm font-semibold text-gray-200 mb-3">Test a Host</p>
        <div className="flex gap-2">
          <input className="input" placeholder="myapp.local"
            value={testHost} onChange={e => { setTestHost(e.target.value); setTestResult(null) }}
            onKeyDown={e => e.key === 'Enter' && test()} />
          <button className="btn-ghost" onClick={test}>Resolve</button>
        </div>
        {testResult && (
          <p className={`mt-2 flex items-center gap-1.5 text-sm ${testResult.found ? 'text-green-400' : 'text-yellow-400'}`}>
            {testResult.found ? <CheckCircle size={13} /> : <AlertTriangle size={13} />}
            {testResult.found
              ? <><span className="font-mono">{testResult.domain}</span> → <span className="font-mono">{testResult.ip}</span></>
              : <><span className="font-mono">{testResult.domain}</span> has no custom override</>
            }
          </p>
        )}
      </div>

      <div className="card">
        <div className="flex items-center gap-2 mb-3">
          <p className="text-sm font-semibold text-gray-200 flex-1">Host Overrides</p>
          <div className="relative">
            <Search size={13} className="absolute left-2.5 top-1/2 -translate-y-1/2 text-gray-500" />
            <input className="input pl-8 w-48 text-xs" placeholder="Search…"
              value={search} onChange={e => setSearch(e.target.value)} />
          </div>
        </div>

        {filtered.length === 0
          ? (
            <div className="text-center py-10 text-gray-500">
              <Globe size={32} className="mx-auto mb-2 opacity-30" />
              <p className="text-sm">No host overrides yet</p>
              <p className="text-xs mt-1">Add one above to get started</p>
            </div>
          )
          : (
            <div className="overflow-auto rounded-lg border border-gray-700">
              <table className="w-full text-sm">
                <thead className="bg-gray-900">
                  <tr className="text-xs text-gray-500 uppercase tracking-wide">
                    <th className="text-left px-4 py-2.5 font-medium">Domain</th>
                    <th className="text-left px-4 py-2.5 font-medium">IP Address</th>
                    <th className="w-10 px-4 py-2.5" />
                  </tr>
                </thead>
                <tbody className="divide-y divide-gray-800">
                  {filtered.map(h => (
                    <tr key={h.domain} className="hover:bg-gray-800/50 group">
                      <td className="px-4 py-2.5 font-mono text-sm text-gray-200">{h.domain}</td>
                      <td className="px-4 py-2.5 font-mono text-sm text-blue-400">{h.ip}</td>
                      <td className="px-4 py-2.5">
                        <button
                          className="text-gray-600 hover:text-red-400 opacity-0 group-hover:opacity-100 transition-all"
                          onClick={() => remove(h.domain)}
                        >
                          <Trash2 size={14} />
                        </button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )
        }
      </div>
    </div>
  )
}
