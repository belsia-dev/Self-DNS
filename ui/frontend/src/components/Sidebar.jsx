import {
  LayoutDashboard, ScrollText, ShieldCheck,
  Ban, Globe, Settings, Wifi, WifiOff, Loader,
} from 'lucide-react'

import logoImg from '../assets/logo.png'

const NAV = [
  { id: 'dashboard', label: 'Dashboard', icon: LayoutDashboard },
  { id: 'querylog', label: 'Query Log', icon: ScrollText },
  { id: 'security', label: 'Security', icon: ShieldCheck },
  { id: 'blocklist', label: 'Blocklist', icon: Ban },
  { id: 'hosts', label: 'Hosts', icon: Globe },
  { id: 'settings', label: 'Settings', icon: Settings },
]

export default function Sidebar({ current, onNavigate, running, ready }) {
  return (
    <aside className="w-56 flex-shrink-0 flex flex-col bg-gray-900 border-r border-gray-800/80 pt-10">

      <div className="px-4 pb-5">
        <div className="flex items-center gap-2.5">
          <img src={logoImg} alt="SelfDNS Logo" className="h-9 w-9 rounded-xl shadow-lg shadow-black/20" />
          <div>
            <h2 className="text-sm font-bold tracking-tight text-gray-100">SelfDNS</h2>
            <p className="text-[10px] uppercase tracking-wider text-gray-500">Control Center</p>
          </div>
        </div>
      </div>

      <div className="px-4 pb-4">
        {!ready ? (
          <div className="flex items-center gap-2 rounded-lg border border-gray-700/40
                          bg-gray-800/50 px-3 py-1.5 text-xs font-medium text-gray-500">
            <Loader size={11} className="animate-spin" />
            Starting…
          </div>
        ) : running ? (
          <div className="flex items-center gap-2 rounded-lg border border-green-800/40
                          bg-green-900/20 px-3 py-1.5 text-xs font-medium text-green-400">
            <span className="h-2 w-2 rounded-full bg-green-400 animate-pulse" />
            <Wifi size={11} />
            DNS RUNNING
          </div>
        ) : (
          <div className="flex items-center gap-2 rounded-lg border border-red-800/40
                          bg-red-900/20 px-3 py-1.5 text-xs font-medium text-red-400">
            <span className="h-2 w-2 rounded-full bg-red-400" />
            <WifiOff size={11} />
            DNS STOPPED
          </div>
        )}
      </div>

      <nav className="flex-1 space-y-0.5 px-2">
        {NAV.map(({ id, label, icon: Icon }) => {
          const active = current === id
          return (
            <button
              key={id}
              onClick={() => onNavigate(id)}
              className={`flex w-full items-center gap-3 rounded-lg px-3 py-2.5 text-left
                          text-sm font-medium transition-colors
                          ${active
                  ? 'bg-blue-600/15 text-blue-400 border border-blue-700/30'
                  : 'text-gray-400 hover:bg-gray-800 hover:text-gray-100 border border-transparent'
                }`}
            >
              <Icon size={15} className={active ? 'text-blue-400' : 'text-gray-500'} />
              {label}
            </button>
          )
        })}
      </nav>

      <div className="px-4 py-4 border-t border-gray-800/60">
        <p className="text-[10px] text-gray-600">v1.0.0 · localhost:5380</p>
      </div>
    </aside>
  )
}
