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

export const api = {
  get: path => fetch(`${API_BASE}${path}`).then(r => { if (!r.ok) throw new Error(r.statusText); return r.json() }),
  post: (path, body) => fetch(`${API_BASE}${path}`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  }).then(r => { if (!r.ok) throw new Error(r.statusText); return r.json() }),
}

const PAGES = {
  dashboard: Dashboard,
  querylog:  QueryLog,
  security:  Security,
  blocklist: Blocklist,
  hosts:     Hosts,
  settings:  Settings,
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

  const Page = PAGES[page] ?? Dashboard

  return (
    <div className="flex h-screen bg-gray-950 overflow-hidden text-gray-100">
      <div className="titlebar-drag pointer-events-none absolute inset-x-0 top-0 h-8 z-50" />

      <Sidebar current={page} onNavigate={setPage} running={running} ready={ready} />

      <main className="relative flex-1 overflow-hidden">
        {!ready && (
          <div className="absolute inset-0 z-40 flex flex-col items-center justify-center bg-gray-950 gap-4">
            <div className="flex items-center gap-3">
              <img src={logoImg} alt="SelfDNS" className="w-9 h-9 rounded-xl shadow-lg shadow-black/20" />
              <div>
                <p className="text-white font-semibold text-lg leading-tight">SelfDNS</p>
                <p className="text-gray-500 text-xs">Control Center</p>
              </div>
            </div>

            {error ? (
              <div className="flex flex-col items-center gap-4 animate-in fade-in zoom-in duration-300">
                <div className="p-4 bg-red-500/10 border border-red-500/20 rounded-xl max-w-sm">
                  <p className="text-red-400 text-sm font-medium mb-1">Startup failed</p>
                  <p className="text-red-300/70 text-xs leading-relaxed whitespace-pre-wrap">{error}</p>
                </div>
                <div className="flex gap-2">
                  <button
                    onClick={() => window.location.reload()}
                    className="px-4 py-2 bg-gray-900 hover:bg-gray-800 text-white text-xs rounded-lg border border-white/5 transition-colors"
                  >
                    Retry
                  </button>
                  <button
                    onClick={() => window.go?.main?.App?.Elevate?.()}
                    className="px-4 py-2 bg-blue-600 hover:bg-blue-500 text-white text-xs rounded-lg font-medium transition-colors"
                  >
                    Elevate &amp; Restart
                  </button>
                </div>
              </div>
            ) : (
              <div className="flex items-center gap-2 text-gray-500 text-sm">
                <div className="w-4 h-4 border-2 border-blue-500 border-t-transparent rounded-full animate-spin" />
                Starting DNS server…
              </div>
            )}
          </div>
        )}

        <div className="h-full overflow-y-auto">
          <Page api={api} />
        </div>
      </main>
    </div>
  )
}
