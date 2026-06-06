import { useMemo, useState } from 'react'
import { AVAILABLE_PLAYERS, TEAMS, DRAFT_STATE, currentPickingTeamIndex, CAP_LIMIT, SLOT_TARGETS, formatSalary, type Player, type Position } from '../../data/mockDraft'
import { DraftViolationModal, type Violation } from './DraftViolationModal'
import { POS_COLORS } from './shared'

const initialDraftedBy = new Map<string, string>()
for (const team of TEAMS) {
  for (const pick of team.picks) {
    if (pick) initialDraftedBy.set(pick.id, team.name)
  }
}

function BlockedLabel({ overCap, positionFull }: { overCap: boolean; positionFull: boolean }) {
  return <div className="text-xs text-red-500">{positionFull ? 'Position Filled' : 'Over Cap'}</div>
}

function CheckIcon() {
  return (
    <svg className="w-2.5 h-2.5 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={3} d="M5 13l4 4L19 7" />
    </svg>
  )
}

function ToggleButton({ active, onClick, accentActive, label, count }: {
  active: boolean
  onClick: () => void
  accentActive: string
  label: string
  count?: number
}) {
  return (
    <button
      onClick={onClick}
      className={[
        'filter-btn flex-1 flex items-center gap-2 border',
        active ? accentActive : 'bg-slate-800 border-slate-700 text-slate-400 hover:bg-slate-700/60 hover:text-slate-300',
      ].join(' ')}
    >
      <div className={[
        'w-4 h-4 rounded border-2 flex items-center justify-center shrink-0 transition-colors',
        active
          ? accentActive.includes('blue') ? 'bg-blue-500 border-blue-500' : 'bg-emerald-500 border-emerald-500'
          : 'border-slate-500',
      ].join(' ')}>
        {active && <CheckIcon />}
      </div>
      {label}
      {active && count !== undefined && (
        <span className={['ml-auto font-normal', accentActive.includes('blue') ? 'text-blue-400/70' : 'text-emerald-400/70'].join(' ')}>
          {count} shown
        </span>
      )}
    </button>
  )
}

function PlayerRow({ player, isMyTurn, capRemaining, positionFull, draftedBy, onAttemptDraft }: {
  player: Player
  isMyTurn: boolean
  capRemaining: number
  positionFull: boolean
  draftedBy: string | undefined
  onAttemptDraft: (p: Player) => void
}) {
  const isDrafted = draftedBy !== undefined
  const overCap = player.salary > capRemaining
  const blocked = overCap || positionFull
  const clickable = isMyTurn && !isDrafted

  return (
    <div
      className={[
        'player-row group',
        isDrafted ? 'opacity-40 cursor-default' :
        clickable  ? 'hover:bg-slate-700/60 cursor-pointer' :
                     'opacity-60 cursor-default',
      ].join(' ')}
      onClick={() => clickable && onAttemptDraft(player)}
    >
      <span className={`pos-pill ${POS_COLORS[player.position]}`}>{player.position}</span>

      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-2">
          <span className="player-name">{player.name}</span>
          <span className="stat-text text-slate-600 shrink-0">{player.age}y</span>
        </div>
        <div className="stat-text">
          {player.team}
          {player.position !== 'G'
            ? ` · ${player.stats.goals}G ${player.stats.assists}A ${player.stats.goals + player.stats.assists}Pts`
            : ` · ${player.stats.wins}W ${player.stats.gaa?.toFixed(2)} GAA`}
        </div>
        {isDrafted && <div className="stat-text italic">Drafted by {draftedBy}</div>}
      </div>

      <div className="text-right shrink-0">
        <div className={`salary-text ${blocked && !isDrafted ? 'text-red-400' : 'text-slate-300'}`}>
          {formatSalary(player.salary)}
        </div>
        {!isDrafted && blocked && <BlockedLabel overCap={overCap} positionFull={positionFull} />}
      </div>

      {clickable && !blocked && (
        <button
          onClick={e => { e.stopPropagation(); onAttemptDraft(player) }}
          className="shrink-0 opacity-0 group-hover:opacity-100 bg-blue-600 hover:bg-blue-500 text-white text-xs font-semibold px-2.5 py-1.5 rounded-lg transition-all"
        >
          Draft
        </button>
      )}
    </div>
  )
}

export function AvailablePlayerList() {
  const [query, setQuery] = useState('')
  const [posFilter, setPosFilter] = useState<Position | 'ALL'>('ALL')
  const [nhlTeamFilter, setNhlTeamFilter] = useState('ALL')
  const [hideTaken, setHideTaken] = useState(false)
  const [showDraftable, setShowDraftable] = useState(false)
  const [draftedBy, setDraftedBy] = useState<Map<string, string>>(initialDraftedBy)
  const [sessionCapUsed, setSessionCapUsed] = useState(() => TEAMS.find(t => t.isMe)!.capUsed)
  const [violation, setViolation] = useState<Violation | null>(null)

  const me = TEAMS.find(t => t.isMe)!
  const pickingIdx = currentPickingTeamIndex(DRAFT_STATE)
  const isMyTurn = TEAMS[pickingIdx].isMe
  const capRemaining = CAP_LIMIT - sessionCapUsed

  const fullPositions = useMemo<Set<Position>>(() => {
    const full = new Set<Position>()
    for (const pos of ['F', 'D', 'G'] as Position[]) {
      const count = AVAILABLE_PLAYERS.filter(
        p => draftedBy.get(p.id) === me.name && p.position === pos
      ).length
      if (count >= SLOT_TARGETS[pos]) full.add(pos)
    }
    return full
  }, [draftedBy, me.name])

  const nhlTeams = useMemo(() =>
    ['ALL', ...Array.from(new Set(AVAILABLE_PLAYERS.map(p => p.team))).sort()],
    []
  )

  const availableCount = useMemo(
    () => AVAILABLE_PLAYERS.filter(p => !draftedBy.has(p.id)).length,
    [draftedBy]
  )

  const draftableCount = useMemo(
    () => AVAILABLE_PLAYERS.filter(p =>
      !draftedBy.has(p.id) && p.salary <= capRemaining && !fullPositions.has(p.position)
    ).length,
    [draftedBy, capRemaining, fullPositions]
  )

  const filtered = AVAILABLE_PLAYERS.filter(p => {
    if (showDraftable) {
      if (draftedBy.has(p.id)) return false
      if (p.salary > capRemaining) return false
      if (fullPositions.has(p.position)) return false
    } else if (hideTaken && draftedBy.has(p.id)) return false
    if (posFilter !== 'ALL' && p.position !== posFilter) return false
    if (nhlTeamFilter !== 'ALL' && p.team !== nhlTeamFilter) return false
    if (query && !p.name.toLowerCase().includes(query.toLowerCase())) return false
    return true
  })

  function handleAttemptDraft(player: Player) {
    const myCountAtPos = AVAILABLE_PLAYERS.filter(
      p => draftedBy.get(p.id) === me.name && p.position === player.position
    ).length
    if (myCountAtPos >= SLOT_TARGETS[player.position]) {
      setViolation({ kind: 'position-full', player, slotsFilled: myCountAtPos, slotsTarget: SLOT_TARGETS[player.position] })
      return
    }
    if (player.salary > capRemaining) {
      setViolation({ kind: 'over-cap', player, shortfall: player.salary - capRemaining, capRemaining })
      return
    }
    setDraftedBy(prev => new Map(prev).set(player.id, TEAMS[pickingIdx].name))
    setSessionCapUsed(prev => prev + player.salary)
  }

  return (
    <>
      {violation && (
        <DraftViolationModal violation={violation} onDismiss={() => setViolation(null)} />
      )}

      <div className="flex flex-col h-full">
        {/* Search + position pills */}
        <div className="flex gap-2 mb-2">
          <div className="relative flex-1">
            <svg className="absolute left-2.5 top-1/2 -translate-y-1/2 w-4 h-4 text-slate-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
            </svg>
            <input
              type="text"
              placeholder="Search players…"
              value={query}
              onChange={e => setQuery(e.target.value)}
              className="filter-control w-full pl-9 pr-3 py-2 text-sm placeholder-slate-500"
            />
          </div>
          <div className="flex gap-1">
            {(['ALL', 'F', 'D', 'G'] as const).map(pos => (
              <button
                key={pos}
                onClick={() => setPosFilter(pos)}
                className={[
                  'filter-btn',
                  posFilter === pos
                    ? pos === 'ALL' ? 'bg-slate-600 text-white'
                      : pos === 'F' ? 'bg-blue-600 text-white'
                      : pos === 'D' ? 'bg-emerald-600 text-white'
                      : 'bg-amber-600 text-white'
                    : 'bg-slate-800 text-slate-400 hover:bg-slate-700',
                ].join(' ')}
              >
                {pos}
              </button>
            ))}
          </div>
        </div>

        {/* NHL team filter */}
        <div className="flex items-center gap-2 mb-2">
          <svg className="w-3.5 h-3.5 text-slate-500 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M3 4h18M7 8h10M10 12h4" />
          </svg>
          <span className="stat-text shrink-0">NHL Team</span>
          <select
            value={nhlTeamFilter}
            onChange={e => setNhlTeamFilter(e.target.value)}
            className="filter-control flex-1 px-2.5 py-1.5 text-xs appearance-none cursor-pointer"
          >
            {nhlTeams.map(team => (
              <option key={team} value={team}>{team === 'ALL' ? 'All Teams' : team}</option>
            ))}
          </select>
        </div>

        {/* Visibility toggles */}
        <div className="flex gap-2 mb-2">
          <ToggleButton
            active={hideTaken}
            onClick={() => setHideTaken(v => !v)}
            accentActive="bg-blue-600/15 border-blue-500/40 text-blue-300"
            label="Hide taken"
            count={availableCount}
          />
          <ToggleButton
            active={showDraftable}
            onClick={() => setShowDraftable(v => !v)}
            accentActive="bg-emerald-600/15 border-emerald-500/40 text-emerald-300"
            label="Draftable only"
            count={draftableCount}
          />
        </div>

        {isMyTurn && (
          <div className="your-turn-banner mb-2">
            <span className="w-2 h-2 rounded-full bg-amber-400 animate-pulse shrink-0" />
            It's your pick! Click any player to draft them.
          </div>
        )}

        <div className="stat-text px-1 mb-1">{availableCount} players available</div>

        <div className="flex-1 overflow-y-auto space-y-0.5 pr-0.5">
          {filtered.length === 0 ? (
            <div className="text-center text-slate-600 py-8 text-sm">No players match your filters</div>
          ) : (
            filtered.map(player => (
              <PlayerRow
                key={player.id}
                player={player}
                isMyTurn={isMyTurn}
                capRemaining={capRemaining}
                positionFull={fullPositions.has(player.position)}
                draftedBy={draftedBy.get(player.id)}
                onAttemptDraft={handleAttemptDraft}
              />
            ))
          )}
        </div>
      </div>
    </>
  )
}
