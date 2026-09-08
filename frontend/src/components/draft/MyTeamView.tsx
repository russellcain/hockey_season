import { currentPickingTeamIndex, formatSalary, type Player } from '../../data/mockDraft'
import { useDraft } from '../../context/DraftContext'
import { POS_COLORS } from './shared'

const SLOT_CONFIG = [
  { pos: 'F' as const, label: 'Forwards' },
  { pos: 'D' as const, label: 'Defence'  },
  { pos: 'G' as const, label: 'Goalies'  },
]

function FilledSlot({ player }: { player: Player }) {
  return (
    <div className="flex items-center gap-2.5 py-2 border-b border-slate-800/50">
      <span className={`pos-pill ${POS_COLORS[player.position]}`}>{player.position}</span>
      <div className="flex-1 min-w-0">
        <div className="player-name">{player.name}</div>
        <div className="stat-text">
          {player.team}
          {player.position !== 'G'
            ? ` · ${player.stats.goals}G ${player.stats.assists}A ${player.stats.goals + player.stats.assists}Pts`
            : ` · ${player.stats.wins ?? 0}W ${player.stats.gaa?.toFixed(2) ?? '—'} GAA`}
        </div>
      </div>
      <div className="text-xs font-mono text-slate-400 shrink-0">{formatSalary(player.salary)}</div>
    </div>
  )
}

function EmptySlot({ pos }: { pos: string }) {
  return (
    <div className="flex items-center gap-2.5 py-2 border-b border-slate-800/30">
      <span className="pos-pill bg-slate-800 text-slate-600 border border-slate-700/50">{pos}</span>
      <div className="flex-1 h-px border-t border-dashed border-slate-800" />
      <div className="text-xs font-mono text-slate-700">—</div>
    </div>
  )
}

export function MyTeamView() {
  const { draftState, teams, capLimit, slotTargets } = useDraft()

  const me = teams.find(t => t.isMe)
  if (!me || !draftState) return null

  const picks = me.picks.filter((p): p is Player => p !== null)
  const isMyTurn = teams[currentPickingTeamIndex(draftState)]?.isMe ?? false

  const capUsed = me.capUsed
  const capRemaining = capLimit - capUsed
  const capPct = capUsed / capLimit
  const totalPicks = draftState.totalRounds
  const picksMade = picks.length

  return (
    <div className="flex flex-col h-full">
      {isMyTurn && (
        <div className="your-turn-banner mb-3 shrink-0">
          <span className="w-2 h-2 rounded-full bg-amber-400 animate-pulse shrink-0" />
          It's your pick! Switch to Players to draft.
        </div>
      )}

      <div className="shrink-0 mb-4 p-3 bg-slate-900/60 rounded-xl border border-slate-800">
        <div className="flex items-baseline justify-between mb-2">
          <span className="section-label text-slate-400">Cap Usage</span>
          <span className={[
            'text-lg font-mono font-bold',
            capPct > 0.9 ? 'text-red-400' : capPct > 0.75 ? 'text-amber-400' : 'text-emerald-400',
          ].join(' ')}>
            {Math.round(capPct * 100)}%
          </span>
        </div>

        <div className="h-3 rounded-full bg-slate-800 overflow-hidden mb-2">
          <div
            className={[
              'h-full rounded-full transition-all',
              capPct > 0.9 ? 'bg-red-500' : capPct > 0.75 ? 'bg-amber-500' : 'bg-emerald-500',
            ].join(' ')}
            style={{ width: `${capPct * 100}%` }}
          />
        </div>

        <div className="grid grid-cols-3 gap-2 text-center">
          <div>
            <div className="stat-text mb-0.5">Used</div>
            <div className="text-sm font-mono font-semibold text-slate-200">{formatSalary(capUsed)}</div>
          </div>
          <div className="border-x border-slate-800">
            <div className="stat-text mb-0.5">Remaining</div>
            <div className={['text-sm font-mono font-semibold', capRemaining < 10_000_000 ? 'text-red-400' : 'text-emerald-400'].join(' ')}>
              {formatSalary(capRemaining)}
            </div>
          </div>
          <div>
            <div className="stat-text mb-0.5">Total</div>
            <div className="text-sm font-mono font-semibold text-slate-400">{formatSalary(capLimit)}</div>
          </div>
        </div>
      </div>

      <div className="shrink-0 flex items-center gap-2 mb-4 px-1">
        <div className="flex gap-0.5 flex-1">
          {Array.from({ length: totalPicks }, (_, i) => (
            <div
              key={i}
              className={['h-1.5 flex-1 rounded-full', i < picksMade ? 'bg-amber-400' : 'bg-slate-800'].join(' ')}
            />
          ))}
        </div>
        <span className="stat-text shrink-0 font-mono">{picksMade}/{totalPicks} picks</span>
      </div>

      <div className="flex-1 overflow-y-auto space-y-4 pr-0.5">
        {SLOT_CONFIG.map(({ pos, label }) => {
          const target = slotTargets[pos] ?? 0
          const filled = picks.filter(p => p.position === pos)
          const empty = Math.max(0, target - filled.length)

          return (
            <div key={pos}>
              <div className="flex items-center gap-2 mb-1">
                <span className={`pos-pill ${POS_COLORS[pos]}`}>{pos}</span>
                <span className="section-label text-slate-400">{label}</span>
                <span className="ml-auto stat-text text-slate-600 font-mono">{filled.length}/{target}</span>
              </div>
              {filled.map(p => <FilledSlot key={p.id} player={p} />)}
              {Array.from({ length: empty }, (_, i) => <EmptySlot key={i} pos={pos} />)}
              {filled.length > target && (
                <div className="text-xs text-amber-400 px-1 mt-0.5 italic">+{filled.length - target} over slot target</div>
              )}
            </div>
          )
        })}
      </div>
    </div>
  )
}
