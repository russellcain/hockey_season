import { TEAMS, DRAFT_STATE, snakePickOrder, formatSalary, CAP_LIMIT } from '../../data/mockDraft'

export function DraftOrder() {
  const order = snakePickOrder(DRAFT_STATE.currentRound, DRAFT_STATE.totalTeams)
  const currentIdx = DRAFT_STATE.currentPick - 1

  return (
    <div className="flex flex-col gap-1 overflow-y-auto">
      <div className="text-xs font-semibold text-slate-400 uppercase tracking-wider px-1 mb-1">
        Round {DRAFT_STATE.currentRound} Order
      </div>
      {order.map((teamIdx, pickIdx) => {
        const team = TEAMS[teamIdx]
        const isPicking = pickIdx === currentIdx
        const hasPicked = pickIdx < currentIdx
        const capPct = team.capUsed / CAP_LIMIT

        return (
          <div
            key={team.id}
            className={[
              'rounded-lg px-3 py-2.5 transition-all',
              isPicking
                ? 'bg-blue-600 ring-2 ring-blue-400 shadow-lg shadow-blue-900/40'
                : hasPicked
                  ? 'bg-slate-800/40 opacity-50'
                  : 'bg-slate-800/60',
              team.isMe && !hasPicked ? 'ring-1 ring-amber-400/40' : '',
            ].filter(Boolean).join(' ')}
          >
            <div className="flex items-center justify-between gap-2">
              <div className="flex items-center gap-2 min-w-0">
                <span className={[
                  'text-xs font-mono w-4 text-right shrink-0',
                  isPicking ? 'text-blue-200' : 'text-slate-500',
                ].join(' ')}>
                  {pickIdx + 1}
                </span>
                <div className="min-w-0">
                  <div className={[
                    'text-sm font-medium truncate',
                    isPicking ? 'text-white' : team.isMe ? 'text-amber-300' : 'text-slate-200',
                  ].join(' ')}>
                    {team.name}
                    {team.isMe && (
                      <span className="ml-1.5 text-xs font-normal text-amber-400/80">(you)</span>
                    )}
                  </div>
                  <div className={[
                    'text-xs',
                    isPicking ? 'text-blue-200' : 'text-slate-500',
                  ].join(' ')}>
                    {formatSalary(team.capUsed)} used
                  </div>
                </div>
              </div>
              {isPicking && (
                <div className="w-2 h-2 rounded-full bg-blue-300 animate-pulse shrink-0" />
              )}
            </div>
            {!hasPicked && (
              <div className="mt-2 h-1 rounded-full bg-slate-700 overflow-hidden">
                <div
                  className={['h-full rounded-full transition-all', isPicking ? 'bg-blue-300' : 'bg-slate-500'].join(' ')}
                  style={{ width: `${capPct * 100}%` }}
                />
              </div>
            )}
          </div>
        )
      })}
    </div>
  )
}
