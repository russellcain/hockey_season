import { useEffect } from 'react'
import { formatSalary, type Player } from '../../data/mockDraft'
import { POS_COLORS } from './shared'

const AUTO_DISMISS_MS = 3000

export function DraftSnackbar({ player, onDismiss }: {
  player: Player
  onDismiss: () => void
}) {
  useEffect(() => {
    const t = setTimeout(onDismiss, AUTO_DISMISS_MS)
    return () => clearTimeout(t)
  }, [onDismiss])

  return (
    <div className="fixed bottom-6 left-1/2 -translate-x-1/2 z-40">
      <div
        role="status"
        className="snackbar-enter flex items-center gap-3 px-4 py-3 bg-slate-900 border border-emerald-500/30 rounded-xl shadow-2xl whitespace-nowrap"
      >
        <div className="w-6 h-6 rounded-full bg-emerald-500/20 flex items-center justify-center shrink-0">
          <svg className="w-3.5 h-3.5 text-emerald-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2.5} d="M5 13l4 4L19 7" />
          </svg>
        </div>
        <span className="stat-text">Drafted</span>
        <span className="text-sm font-semibold text-slate-100">{player.name}</span>
        <span className={`pos-pill ${POS_COLORS[player.position]}`}>{player.position}</span>
        <span className="salary-text text-slate-400">{formatSalary(player.salary)}</span>
        <button
          onClick={onDismiss}
          aria-label="Dismiss"
          className="ml-1 text-slate-600 hover:text-slate-300 transition-colors"
        >
          <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
      </div>
    </div>
  )
}
