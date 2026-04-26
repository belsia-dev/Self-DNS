export default function StatCard({ label, value, sub, accent = 'accent', icon: Icon }) {
  const accentClass = {
    accent: 'text-accent',
    info:   'text-info',
    bad:    'text-bad',
    warn:   'text-warn',
    violet: 'text-violet',
    mute:   'text-text-secondary',
  }[accent]

  return (
    <div className="panel p-3.5 flex flex-col gap-3 group">
      <div className="flex items-center justify-between">
        <p className="h-card">{label}</p>
        {Icon && (
          <Icon size={12} className={`${accentClass} opacity-70 group-hover:opacity-100 transition-opacity`} />
        )}
      </div>

      <p className={`num-display text-[28px] leading-none ${accentClass}`}>
        {value ?? '—'}
      </p>

      {sub && (
        <p className="text-2xs font-mono text-text-muted truncate">{sub}</p>
      )}
    </div>
  )
}
