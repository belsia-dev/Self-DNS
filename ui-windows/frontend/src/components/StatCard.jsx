export default function StatCard({ label, value, sub, color = 'blue', icon: Icon }) {
  const colors = {
    blue:   'text-blue-400   bg-blue-900/20   border-blue-800/30',
    green:  'text-green-400  bg-green-900/20  border-green-800/30',
    red:    'text-red-400    bg-red-900/20    border-red-800/30',
    orange: 'text-orange-400 bg-orange-900/20 border-orange-800/30',
    purple: 'text-purple-400 bg-purple-900/20 border-purple-800/30',
  }

  return (
    <div className={`card border ${colors[color]} flex flex-col gap-1`}>
      <div className="flex items-center justify-between">
        <p className="text-xs font-medium text-gray-400 uppercase tracking-wide">{label}</p>
        {Icon && <Icon size={14} className={colors[color].split(' ')[0]} />}
      </div>
      <p className={`text-2xl font-bold ${colors[color].split(' ')[0]}`}>{value ?? '—'}</p>
      {sub && <p className="text-xs text-gray-500">{sub}</p>}
    </div>
  )
}
