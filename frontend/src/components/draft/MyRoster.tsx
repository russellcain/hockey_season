import { formatSalary, type Player } from '../../data/mockDraft'
import { useDraft } from '../../context/DraftContext'
import { POS_COLORS } from './shared'

function StatLine({ player }: { player: Player }) {
  if (player.position === 'G') {
    return <span className="stat-text">{player.stats.wins}W · {player.stats.gaa?.toFixed(2)} GAA</span>
  }
  return <span className="stat-text">{player.stats.goals}G · {player.stats.assists}A</span>
}

export function MyRoster() {
  const { teams, capLimit } = useDraft()
  const me = teams.find(t => t.isMe)
  if (!me) return null

  const roster = me.picks.filter((p): p is Player => p !== null)
  const capPct = me.capUsed / capLimit
  const capRemaining = capLimit - me.capUsed

  const forwards = roster.filter(p => p.position === 'F')
  const defence  = roster.filter(p => p.position === 'D')
  const goalies  = roster.filter(p => p.position === 'G')

  const slotsFilled = roster.length
  const slotsTotal = 20

  return (
    <div className="flex flex-col h-full">
      <div className="flex items-center justify-between mb-3">
        <div>
          <h3 className="text-sm font-semibold text-amber-300">{me.name}</h3>
          <p className="stat-text">{slotsFilled} of {slotsTotal} picks made</p>
        </div>
        <div className="text-right">
          <div className="stat-text">Cap remaining</div>
          <div className={['text-sm font-mono font-bold', capRemaining < 10_000_000 ? 'text-red-400' : 'text-emerald-400'].join(' ')}>
            {formatSalary(capRemaining)}
          </div>
        </div>
      </div>

      <div className="mb-3">
        <div className="flex justify-between stat-text mb-1">
          <span>Cap used: {formatSalary(me.capUsed)}</span>
          <span>{Math.round(capPct * 100)}%</span>
        </div>
        <div className="h-2 rounded-full bg-slate-800 overflow-hidden">
          <div
            className={[
              'h-full rounded-full transition-all',
              capPct > 0.9 ? 'bg-red-500' : capPct > 0.75 ? 'bg-amber-500' : 'bg-emerald-500',
            ].join(' ')}
            style={{ width: `${capPct * 100}%` }}
          />
        </div>
        <div className="flex justify-between text-xs text-slate-600 mt-0.5">
          <span>$0</span>
          <span>{formatSalary(capLimit)}</span>
        </div>
      </div>

      <div className="flex-1 overflow-y-auto space-y-3 pr-0.5">
        {[
          { label: 'Forwards', players: forwards, max: 12 },
          { label: 'Defence',  players: defence,  max: 6  },
          { label: 'Goalies',  players: goalies,  max: 2  },
        ].map(({ label, players, max }) => (
          <div key={label}>
            <div className="section-label text-slate-500 mb-1 flex items-center gap-2">
              {label}
              <span className="text-slate-600">{players.length}/{max}</span>
            </div>
            {players.map(player => (
              <div key={player.id} className="flex items-center gap-2 py-1.5 border-b border-slate-800/60">
                <span className={`pos-pill ${POS_COLORS[player.position]}`}>{player.position}</span>
                <div className="min-w-0 flex-1">
                  <div className="player-name">{player.name}</div>
                  <StatLine player={player} />
                </div>
                <div className="text-xs font-mono text-slate-400 shrink-0">{formatSalary(player.salary)}</div>
              </div>
            ))}
            {players.length === 0 && (
              <div className="stat-text italic py-1">No {label.toLowerCase()} yet</div>
            )}
          </div>
        ))}
      </div>
    </div>
  )
}
