import { useState, useEffect } from 'react'
import { CheckCircle, XCircle, RefreshCw, ShieldCheck, Wrench } from 'lucide-react'

function ScoreRing({ score }) {
  const r = 36
  const circ = 2 * Math.PI * r
  const offset = circ - (score / 100) * circ
  const color = score >= 80 ? '#22c55e' : score >= 50 ? '#f59e0b' : '#ef4444'

  return (
    <div className="flex flex-col items-center">
      <svg width={96} height={96} className="-rotate-90">
        <circle cx={48} cy={48} r={r} fill="none" stroke="#1f2937" strokeWidth={8} />
        <circle
          cx={48} cy={48} r={r} fill="none"
          stroke={color} strokeWidth={8}
          strokeDasharray={circ}
          strokeDashoffset={offset}
          strokeLinecap="round"
          style={{ transition: 'stroke-dashoffset 0.6s ease' }}
        />
      </svg>
      <div className="-mt-[72px] flex flex-col items-center">
        <span className="text-2xl font-bold text-white">{score}</span>
        <span className="text-[10px] text-gray-500">/ 100</span>
      </div>
      <p className="mt-8 text-sm text-gray-400">Security Score</p>
    </div>
  )
}

export default function Security({ api }) {
  const [report, setReport] = useState(null)
  const [loading, setLoading] = useState(true)
  const [fixMsg, setFixMsg] = useState({})

  const load = async () => {
    setLoading(true)
    try {
      const r = await api.get('/security')
      setReport(r)
    } catch { }
    setLoading(false)
  }

  useEffect(() => { load() }, [api])

  const fix = async (check) => {
    if (check.id === 'dot_enabled' || check.id === 'rate_limit' ||
      check.id === 'rebinding' || check.id === 'dnssec' ||
      check.id === 'privacy') {
      try {
        const cfg = await api.get('/config')
        const patch = { ...cfg }
        if (check.id === 'dot_enabled') patch.use_tls = true
        if (check.id === 'rate_limit') patch.rate_limit = { ...patch.rate_limit, enabled: true }
        if (check.id === 'rebinding') patch.dns_rebinding_protection = true
        if (check.id === 'dnssec') patch.dnssec = true
        if (check.id === 'privacy') patch.log_queries = false
        await api.post('/config', patch)
        setFixMsg(prev => ({ ...prev, [check.id]: 'Fixed! Reloading…' }))
        setTimeout(() => load(), 1500)
      } catch (e) {
        setFixMsg(prev => ({ ...prev, [check.id]: 'Fix failed: ' + e.message }))
      }
    } else if (check.id === 'config_perms') {
      setFixMsg(prev => ({ ...prev, [check.id]: 'Run: chmod 600 ~/.config/selfdns/config.yaml' }))
    }
  }

  if (loading) {
    return (
      <div className="flex items-center justify-center h-full">
        <RefreshCw size={24} className="animate-spin text-blue-400" />
      </div>
    )
  }

  const checks = report?.checks ?? []
  const score = report?.score ?? 0

  return (
    <div className="p-6 space-y-6">
      <div className="flex items-center justify-between pt-2">
        <div>
          <h1 className="text-xl font-bold text-white">Security</h1>
          <p className="text-sm text-gray-400">Automated configuration audit</p>
        </div>
        <button className="btn-ghost flex items-center gap-2" onClick={load}>
          <RefreshCw size={14} /> Refresh
        </button>
      </div>

      <div className="card flex items-center gap-8">
        <ScoreRing score={score} />
        <div className="flex-1">
          <p className="text-lg font-semibold text-white mb-1">
            {score >= 80 ? 'Great security posture' : score >= 50 ? 'Some improvements needed' : 'Action required'}
          </p>
          <p className="text-sm text-gray-400 mb-3">
            {checks.filter(c => c.pass).length} of {checks.length} checks passing
          </p>
          <div className="flex gap-4 text-sm">
            <span className="flex items-center gap-1 text-green-400">
              <CheckCircle size={14} />
              {checks.filter(c => c.pass).length} passed
            </span>
            <span className="flex items-center gap-1 text-red-400">
              <XCircle size={14} />
              {checks.filter(c => !c.pass).length} failed
            </span>
          </div>
        </div>
      </div>

      <div className="space-y-2">
        {checks.map(check => (
          <div key={check.id}
            className={`card flex items-start gap-4 border ${check.pass ? 'border-green-800/30 bg-green-900/10' : 'border-red-800/30 bg-red-900/10'
              }`}
          >
            {check.pass
              ? <CheckCircle size={18} className="text-green-400 mt-0.5 flex-shrink-0" />
              : <XCircle size={18} className="text-red-400   mt-0.5 flex-shrink-0" />
            }
            <div className="flex-1 min-w-0">
              <p className={`text-sm font-medium ${check.pass ? 'text-green-300' : 'text-red-300'}`}>
                {check.name}
              </p>
              <p className="text-xs text-gray-400 mt-0.5">{check.message}</p>
              {fixMsg[check.id] && (
                <p className="text-xs text-blue-400 mt-1 font-mono">{fixMsg[check.id]}</p>
              )}
            </div>
            {!check.pass && check.can_fix && (
              <button
                className="flex items-center gap-1.5 text-xs px-3 py-1.5 bg-blue-700/30 hover:bg-blue-700/50
                           text-blue-300 border border-blue-700/40 rounded-lg transition-colors flex-shrink-0"
                onClick={() => fix(check)}
              >
                <Wrench size={12} /> Fix it
              </button>
            )}
          </div>
        ))}
      </div>
    </div>
  )
}
