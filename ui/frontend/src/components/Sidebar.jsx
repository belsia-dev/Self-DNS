import {
  LayoutDashboard, ScrollText, ShieldCheck,
  Ban, Globe, Settings as Cog,
} from 'lucide-react'

import logoImg from '../assets/logo.png'

const SECTIONS = [
  {
    label: 'Overview',
    items: [
      { id: 'dashboard', label: 'Dashboard', icon: LayoutDashboard, kbd: '1' },
      { id: 'querylog',  label: 'Query Log', icon: ScrollText,      kbd: '2' },
    ],
  },
  {
    label: 'Policy',
    items: [
      { id: 'blocklist', label: 'Blocklist', icon: Ban,    kbd: '3' },
      { id: 'hosts',     label: 'Hosts',     icon: Globe,  kbd: '4' },
    ],
  },
  {
    label: 'System',
    items: [
      { id: 'security', label: 'Security', icon: ShieldCheck, kbd: '5' },
      { id: 'settings', label: 'Settings', icon: Cog,         kbd: '6' },
    ],
  },
]

export default function Sidebar({ current, onNavigate, running, ready }) {
  const statusLabel = !ready ? 'starting' : running ? 'live' : 'offline'
  const statusTone  = !ready ? 'mute'     : running ? 'good' : 'bad'

  return (
    <aside className="w-[208px] flex-shrink-0 flex flex-col bg-ink-100/60
                      border-r border-line/60 pt-9 select-none">
      <div className="px-4 pb-3 flex items-center gap-2.5">
        <div className="relative">
          <img src={logoImg} alt="" className="h-7 w-7 rounded-md" />
          <span className="absolute -inset-1 rounded-lg ring-1 ring-line/40" />
        </div>
        <div className="leading-none">
          <p className="font-display text-[14px] font-semibold tracking-tight text-text-primary">
            SelfDNS
          </p>
          <p className="mt-1 text-[9.5px] uppercase tracking-widest text-text-dim">
            Operator
          </p>
        </div>
      </div>

      <div className="mx-3 mb-4 mt-1 flex items-center justify-between rounded-md
                      bg-ink-200/60 ring-1 ring-line/60 px-2.5 py-1.5">
        <div className="flex items-center gap-2">
          <StatusDot tone={statusTone} />
          <span className="text-2xs uppercase tracking-widest text-text-secondary">
            DNS · {statusLabel}
          </span>
        </div>
        <span className="text-2xs font-mono text-text-dim">53</span>
      </div>

      <nav className="flex-1 overflow-y-auto px-2 space-y-4" aria-label="Main navigation">
        {SECTIONS.map(section => (
          <div key={section.label}>
            <p className="px-2.5 mb-1 text-[9.5px] font-semibold uppercase
                          tracking-widest text-text-dim">
              {section.label}
            </p>
            <div className="space-y-0.5">
              {section.items.map(({ id, label, icon: Icon, kbd }) => {
                const active = current === id
                return (
                  <button
                    key={id}
                    aria-current={active ? 'page' : undefined}
                    onClick={() => onNavigate(id)}
                    className={`group nav-item ${active ? 'nav-item-active' : 'hover:bg-ink-200/60 hover:text-text-primary'}`}
                  >
                    <Icon size={14} className="nav-item-icon" />
                    <span className="flex-1 text-left">{label}</span>
                    <span className="kbd opacity-0 group-hover:opacity-100 transition-opacity">
                      {kbd}
                    </span>
                  </button>
                )
              })}
            </div>
          </div>
        ))}
      </nav>

      <div className="px-3 py-3 border-t border-line/60 flex items-center justify-between">
        <span className="text-2xs font-mono text-text-dim">v1.0.0</span>
        <span className="text-2xs font-mono text-text-dim">:5380</span>
      </div>
    </aside>
  )
}

function StatusDot({ tone }) {
  if (tone === 'good') {
    return (
      <span className="relative inline-flex h-2 w-2">
        <span className="absolute inset-0 rounded-full bg-accent animate-pulse-soft" />
        <span className="absolute inset-0 rounded-full bg-accent blur-[3px] opacity-70" />
        <span className="relative inline-flex rounded-full h-2 w-2 bg-accent" />
      </span>
    )
  }
  return (
    <span className={`h-2 w-2 rounded-full ${
      tone === 'bad' ? 'bg-bad' :
      tone === 'warn' ? 'bg-warn' :
      'bg-text-dim'
    }`} />
  )
}
