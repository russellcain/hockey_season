import { snakePickOrder, type Player } from '../../data/mockDraft'
import { useDraft } from '../../context/DraftContext'

const POS_BG: Record<string, string> = {
  F: 'bg-blue-500/20 text-blue-300',
  D: 'bg-emerald-500/20 text-emerald-300',
  G: 'bg-amber-500/20 text-amber-300',
}

function PickCell({ player, isCurrent, isPast }: {
  player: Player | null | undefined
  isCurrent: boolean
  isPast: boolean
}) {
  if (player) {
    return (
      <div className="h-14 px-2 py-1.5 bg-slate-800/80 rounded flex flex-col justify-center gap-0.5">
        <div className="flex items-center gap-1.5">
          <span className={['text-xs px-1 rounded font-semibold shrink-0', POS_BG[player.position]].join(' ')}>
            {player.position}
          </span>
          <span className="text-xs text-slate-200 truncate leading-tight font-medium">{player.name.split(' ').pop()}</span>
        </div>
        <div className="text-xs text-slate-500 truncate pl-0.5">{player.team}</div>
      </div>
    )
  }
  if (isCurrent) {
    return (
      <div className="h-14 px-2 py-1.5 rounded flex items-center justify-center bg-blue-600/20 ring-2 ring-blue-500/60 ring-inset">
        <span className="w-2 h-2 rounded-full bg-blue-400 animate-pulse" />
      </div>
    )
  }
  return (
    <div className={['h-14 rounded', isPast ? 'bg-slate-800/20' : 'bg-slate-800/40 border border-dashed border-slate-700/50'].join(' ')} />
  )
}

export function DraftBoard() {
  const { draftState, teams } = useDraft()

  if (!draftState || teams.length === 0) return null

  const { totalRounds, totalTeams, currentRound, currentPick } = draftState

  return (
    <div className="overflow-x-auto">
      <div style={{ minWidth: `${totalTeams * 120 + 60}px` }}>
        <div className="flex gap-1 mb-1 pl-14">
          {teams.map(team => (
            <div key={team.id} className={[
              'flex-1 text-center text-xs font-medium truncate px-1',
              team.isMe ? 'text-amber-300' : 'text-slate-400',
            ].join(' ')}>
              {team.name.split(' ')[0]}
            </div>
          ))}
        </div>

        <div className="space-y-1 max-h-52 overflow-y-auto pr-0.5">
          {Array.from({ length: totalRounds }, (_, roundIdx) => {
            const round = roundIdx + 1
            const order = snakePickOrder(round, totalTeams)
            const isCurrentRound = round === currentRound
            const isPastRound = round < currentRound

            return (
              <div key={round} className={[
                'flex gap-1 items-center',
                !isCurrentRound && !isPastRound ? 'opacity-40' : '',
              ].join(' ')}>
                <div className={[
                  'w-12 shrink-0 text-xs font-mono text-right pr-1.5',
                  isCurrentRound ? 'text-blue-400 font-bold' : 'text-slate-600',
                ].join(' ')}>
                  R{round}
                </div>
                {teams.map((_, teamIdx) => {
                  const pickPos = order.indexOf(teamIdx)
                  const team = teams[teamIdx]
                  const pickNumber = pickPos + 1
                  const player = team.picks[roundIdx] ?? null

                  const isCurrent = isCurrentRound && pickNumber === currentPick
                  const isPastPick = isPastRound || (isCurrentRound && pickNumber < currentPick)

                  return (
                    <div key={teamIdx} className="flex-1 min-w-0">
                      <PickCell player={player} isCurrent={isCurrent} isPast={isPastPick} />
                    </div>
                  )
                })}
              </div>
            )
          })}
        </div>
      </div>
    </div>
  )
}
