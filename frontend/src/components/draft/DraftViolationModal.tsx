import { formatSalary, type Player, type Position } from '../../data/mockDraft'
import { POS_COLORS } from './shared'

export type Violation =
  | { kind: 'over-cap';       player: Player; shortfall: number; capRemaining: number }
  | { kind: 'position-full';  player: Player; slotsFilled: number; slotsTarget: number }

const POSITION_LABEL: Record<Position, string> = {
  F: 'Forward',
  D: 'Defence',
  G: 'Goalie',
}

function Row({ label, value, highlight }: { label: string; value: string; highlight?: boolean }) {
  return (
    <div className="flex justify-between items-baseline py-1">
      <span className="stat-text">{label}</span>
      <span className={['text-sm font-mono font-semibold', highlight ? 'text-red-400' : 'text-slate-300'].join(' ')}>
        {value}
      </span>
    </div>
  )
}

function PlayerChip({ player }: { player: Player }) {
  return (
    <div className="flex items-center gap-2 mb-4">
      <span className={`pos-pill ${POS_COLORS[player.position]}`}>{player.position}</span>
      <span className="text-sm font-semibold text-slate-100">{player.name}</span>
      <span className="stat-text">{player.team}</span>
    </div>
  )
}

export function DraftViolationModal({ violation, onDismiss }: {
  violation: Violation
  onDismiss: () => void
}) {
  const isOverCap = violation.kind === 'over-cap'

  return (
    <div
      className="fixed inset-0 bg-black/70 z-50 flex items-center justify-center p-4"
      onClick={onDismiss}
    >
      <div
        className="bg-slate-900 border border-slate-700/80 rounded-2xl p-6 max-w-xs w-full shadow-2xl"
        onClick={e => e.stopPropagation()}
      >
        <div className="flex items-center gap-3 mb-4">
          <div className={[
            'w-8 h-8 rounded-lg flex items-center justify-center shrink-0',
            isOverCap ? 'bg-red-500/20' : 'bg-amber-500/20',
          ].join(' ')}>
            {isOverCap ? (
              <svg className="w-4 h-4 text-red-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v3m0 3h.01M10.29 3.86L1.82 18a2 2 0 001.71 3h16.94a2 2 0 001.71-3L13.71 3.86a2 2 0 00-3.42 0z" />
              </svg>
            ) : (
              <svg className="w-4 h-4 text-amber-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M18.364 18.364A9 9 0 005.636 5.636m12.728 12.728A9 9 0 015.636 5.636m12.728 12.728L5.636 5.636" />
              </svg>
            )}
          </div>
          <div>
            <h2 className="text-sm font-bold text-slate-100">
              {isOverCap ? 'Over Cap Limit' : `${POSITION_LABEL[violation.player.position]} Slots Full`}
            </h2>
            <p className="stat-text">
              {isOverCap ? 'This pick exceeds your remaining cap space' : 'No room left at this position'}
            </p>
          </div>
        </div>

        <PlayerChip player={violation.player} />

        {violation.kind === 'over-cap' && (
          <div className="bg-slate-800/60 rounded-xl px-3 py-1 divide-y divide-slate-700/60">
            <Row label="Cap remaining" value={formatSalary(violation.capRemaining)} />
            <Row label="Player cost"   value={formatSalary(violation.player.salary)} />
            <Row label="Shortfall"     value={formatSalary(violation.shortfall)} highlight />
          </div>
        )}

        {violation.kind === 'position-full' && (
          <div className="bg-slate-800/60 rounded-xl px-3 py-2 text-xs text-slate-400 leading-relaxed">
            You've used all{' '}
            <span className="text-slate-200 font-semibold">{violation.slotsTarget}</span>{' '}
            {POSITION_LABEL[violation.player.position].toLowerCase()} slot{violation.slotsTarget !== 1 ? 's' : ''}.
            Pick a player at a position you still have room to fill.
          </div>
        )}

        <button
          onClick={onDismiss}
          className="mt-4 w-full bg-slate-800 hover:bg-slate-700 text-slate-200 text-sm font-semibold py-2.5 rounded-xl transition-colors"
        >
          Got it
        </button>
      </div>
    </div>
  )
}
