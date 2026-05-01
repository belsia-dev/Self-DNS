import { useState, useEffect } from 'react'
import { Check, X as XIcon, RefreshCw, Wrench, Shield } from 'lucide-react'

function ScoreRing({ score }) {
  const r = 38
  const circ = 2 * Math.PI * r
  const offset = circ - (score / 100) * circ
  const tone =
    score >= 80 ? '#a3e635' :
    score >= 50 ? '#fbbf24' :
                  '#f87171'

  return (
    <div className="relative flex items-center justify-center">
      <svg width={104} height={104} className="-rotate-90">
        <circle cx={52} cy={52} r={r} fill="none" stroke="#1f1f23" strokeWidth={6} />
        <circle
          cx={52} cy={52} r={r}
          fill="none" stroke={tone} strokeWidth={6}
          strokeDasharray={circ}
          strokeDashoffset={offset}
          strokeLinecap="round"
          style={{ transition: 'stroke-dashoffset 0.6s ease' }}
        />
      </svg>
      <div className="absolute inset-0 flex flex-col items-center justify-center">
        <span className="num-display text-[28px] leading-none" style={{ color: tone }}>
          {score}
        </span>
        <span className="text-2xs font-mono text-text-dim mt-1">of 100</span>
      </div>
    </div>
  )
}

export default function Security({ api }) {
  const [report, setReport]   = useState(null)
  const [loading, setLoading] = useState(true)
  const [fixMsg, setFixMsg]   = useState({})

  const load = async () => {
    setLoading(true)
    try { setReport(await api.get('/security')) } catch (err) { console.error('[Security]:', err) }
    setLoading(false)
  }
  useEffect(() => { load() }, [api])

  const fix = async check => {
    const togglable = ['dot_enabled', 'rate_limit', 'rebinding', 'dnssec', 'privacy']
    if (togglable.includes(check.id)) {
      try {
        const cfg = await api.get('/config')
        const patch = { ...cfg }
        if (check.id === 'dot_enabled') patch.use_tls = true
        if (check.id === 'rate_limit')  patch.rate_limit = { ...patch.rate_limit, enabled: true }
        if (check.id === 'rebinding')   patch.dns_rebinding_protection = true
        if (check.id === 'dnssec')      patch.dnssec = true
        if (check.id === 'privacy')     patch.log_queries = false
        await api.post('/config', patch)
        setFixMsg(p => ({ ...p, [check.id]: 'fixed · reloading' }))
        setTimeout(load, 1500)
      } catch (e) {
        setFixMsg(p => ({ ...p, [check.id]: 'fix failed: ' + e.message }))
      }
    } else if (check.id === 'config_perms') {
      setFixMsg(p => ({ ...p, [check.id]: 'run: chmod 600 ~/.config/selfdns/config.yaml' }))
    }
  }

  if (loading) {
    return (
      <div className="flex items-center justify-center h-full text-text-muted">
        <RefreshCw size={18} className="animate-spin text-accent" />
      </div>
    )
  }

  const checks = report?.checks ?? []
  const score  = report?.score ?? 0
  const passed = checks.filter(c => c.pass).length
  const failed = checks.length - passed

  const headline =
    score >= 80 ? 'Strong configuration'
    : score >= 50 ? 'Some gaps to address'
    : 'Action required'

  return (
    <div className="px-7 py-6 space-y-6 max-w-[1100px]">
      <header className="flex items-end justify-between pt-2">
        <div>
          <h1 className="h-page">Security</h1>
          <p className="mt-1 text-xs text-text-muted">
            <span className="font-mono">{checks.length}</span> automated checks
          </p>
        </div>
        <button className="btn-ghost" onClick={load}>
          <RefreshCw size={12} /> Re-run
        </button>
      </header>

      <section className="panel p-5 flex items-center gap-6">
        <ScoreRing score={score} />
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2">
            <Shield size={13} className="text-accent" />
            <p className="font-display text-base font-semibold tracking-tight text-text-primary">
              {headline}
            </p>
          </div>
          <p className="mt-1 text-xs text-text-muted">
            <span className="font-mono">{passed}</span> of <span className="font-mono">{checks.length}</span> checks passing
          </p>

          <div className="mt-4 flex gap-2">
            <span className="chip-good">
              <Check size={10} /> {passed} passed
            </span>
            <span className={failed > 0 ? 'chip-bad' : 'chip-mute'}>
              <XIcon size={10} /> {failed} failed
            </span>
          </div>

          <div className="mt-4 h-1.5 w-full rounded-full bg-ink-300 overflow-hidden">
            <div
              className="h-full rounded-full transition-all duration-700"
              style={{
                width: `${score}%`,
                background: score >= 80 ? '#a3e635'
                          : score >= 50 ? '#fbbf24'
                          : '#f87171',
              }}
            />
          </div>
        </div>
      </section>

      <section className="space-y-1.5">
        {checks.map(check => (
          <div
            key={check.id}
            className={`panel ring-line/40 px-4 py-3 flex items-start gap-3 transition-colors ${
              check.pass ? '' : 'bg-bad/[0.04] ring-bad/15'
            }`}
          >
            <div className={`mt-0.5 h-5 w-5 rounded flex items-center justify-center ring-1
                            ${check.pass
                              ? 'bg-accent/15 ring-accent/30 text-accent'
                              : 'bg-bad/15 ring-bad/30 text-bad'}`}>
              {check.pass ? <Check size={12} /> : <XIcon size={12} />}
            </div>
            <div className="flex-1 min-w-0">
              <p className={`text-xs font-semibold tracking-tight ${
                check.pass ? 'text-text-primary' : 'text-bad'
              }`}>
                {check.name}
              </p>
              <p className="mt-0.5 text-2xs text-text-muted leading-relaxed">
                {check.message}
              </p>
              {fixMsg[check.id] && (
                <p className="mt-1.5 text-2xs font-mono text-info uppercase tracking-widest">
                  {fixMsg[check.id]}
                </p>
              )}
            </div>
            {!check.pass && check.can_fix && (
              <button
                onClick={() => fix(check)}
                className="btn-ghost flex-shrink-0"
              >
                <Wrench size={11} /> Fix
              </button>
            )}
          </div>
        ))}
      </section>
    </div>
  )
}
