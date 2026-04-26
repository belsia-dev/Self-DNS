import { useState, useEffect } from 'react'
import Sidebar from './components/Sidebar.jsx'
import Dashboard from './pages/Dashboard.jsx'
import QueryLog from './pages/QueryLog.jsx'
import Security from './pages/Security.jsx'
import Blocklist from './pages/Blocklist.jsx'
import Hosts from './pages/Hosts.jsx'
import Settings from './pages/Settings.jsx'

import logoImg from './assets/logo.png'

const API_BASE = 'http://127.0.0.1:5380/api'

async function readJSON(response) {
  const text = await response.text()
  if (!text) return null
  try { return JSON.parse(text) } catch { return null }
}

async function request(path, options) {
  const response = await fetch(`${API_BASE}${path}`, options)
  const data = await readJSON(response)
  if (!response.ok) throw new Error(data?.error || response.statusText)
  return data
}

export const api = {
  get: path => request(path),
  post: (path, body) => request(path, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  }),
}

const PAGES = {
  dashboard: Dashboard,
  querylog:  QueryLog,
  security:  Security,
  blocklist: Blocklist,
  hosts:     Hosts,
  settings:  Settings,
}

const KEY_MAP = {
  '1': 'dashboard',
  '2': 'querylog',
  '3': 'blocklist',
  '4': 'hosts',
  '5': 'security',
  '6': 'settings',
}

export default function App() {
  const [page, setPage]       = useState('dashboard')
  const [running, setRunning] = useState(false)
  const [error, setError]     = useState('')
  const [ready, setReady]     = useState(false)

  useEffect(() => {
    let mounted = true
    const poll = async () => {
      try {
        const s = await api.get('/status')
        if (mounted) {
          setRunning(!!s.running)
          setReady(true)
          setError('')
        }
      } catch {
        if (mounted) {
          setRunning(false)
          try {
            const err = await window.go?.main?.App?.GetStartError?.()
            if (err) setError(err)
          } catch {}
        }
      }
    }
    poll()
    const id = setInterval(poll, 2000)
    return () => { mounted = false; clearInterval(id) }
  }, [])

  useEffect(() => {
    const handler = (e) => {
      if (e.metaKey || e.ctrlKey || e.altKey) return
      const tag = e.target?.tagName
      if (tag === 'INPUT' || tag === 'TEXTAREA' || e.target?.isContentEditable) return
      const next = KEY_MAP[e.key]
      if (next) { e.preventDefault(); setPage(next) }
    }
    window.addEventListener('keydown', handler)
    return () => window.removeEventListener('keydown', handler)
  }, [])

  const Page = PAGES[page] ?? Dashboard

  return (
    <div className="flex h-screen bg-ink-50 overflow-hidden text-text-primary">
      <div className="titlebar-drag pointer-events-none absolute inset-x-0 top-0 h-8 z-50" />

      <Sidebar current={page} onNavigate={setPage} running={running} ready={ready} />

      <main className="relative flex-1 overflow-hidden">
        {!ready && <Boot error={error} />}
        <div className="h-full overflow-y-auto animate-fade-in">
          <Page api={api} />
        </div>
      </main>
    </div>
  )
}

function Boot({ error }) {
  return (
    <div className="absolute inset-0 z-40 bg-ink-50/95 backdrop-blur-sm
                    flex flex-col items-center justify-center gap-6 animate-fade-in">
      <div className="flex flex-col items-center gap-3">
        <div className="relative">
          <img src={logoImg} alt="" className="h-12 w-12 rounded-xl" />
          {!error && (
            <span className="absolute inset-0 rounded-xl ring-1 ring-accent/40 animate-breath" />
          )}
        </div>
        <div className="text-center">
          <p className="font-display text-base font-semibold tracking-tight text-text-primary">
            SelfDNS
          </p>
          <p className="text-2xs uppercase tracking-widest text-text-dim mt-1">
            Bringing DNS up
          </p>
        </div>
      </div>

      {error ? (
        <div className="w-full max-w-sm flex flex-col items-center gap-3 px-6">
          <div className="w-full p-3 panel ring-bad/30 bg-bad/5">
            <p className="text-xs font-semibold text-bad mb-1">Startup failed</p>
            <p className="text-2xs text-text-secondary leading-relaxed whitespace-pre-wrap font-mono">
              {error}
            </p>
          </div>
          <div className="flex gap-2">
            <button onClick={() => window.location.reload()} className="btn-ghost">
              Retry
            </button>
            <button onClick={() => window.go?.main?.App?.Elevate?.()} className="btn-primary">
              Elevate &amp; Restart
            </button>
          </div>
        </div>
      ) : (
        <div className="flex items-center gap-2 text-text-muted text-xs">
          <span className="dot-good animate-pulse-soft" />
          <span className="font-mono uppercase tracking-widest text-2xs">
            Resolving
          </span>
        </div>
      )}
    </div>
  )
}
