import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { useAuth, API_BASE, authHeaders } from '../context/AuthContext'

interface TxnRecord {
  id: number
  teamId: number
  teamName: string
  droppedPlayer: string
  addedPlayer: string
  txnType: string
  createdAt: string
}

const txnTypeLabel: Record<string, string> = {
  elective: 'Drop/Add',
  injury_sub: 'Injury Sub',
  cut: 'Cut',
}

function fmtDate(iso: string) {
  try {
    return new Date(iso).toLocaleString(undefined, {
      month: 'short', day: 'numeric', hour: 'numeric', minute: '2-digit',
    })
  } catch {
    return iso
  }
}

interface Props {
  /** The team whose transactions to show. */
  targetTeamId: number
  /** Total transactions used (shown in header). */
  transactionsUsed?: number
}

export function TeamTransactionHistory({ targetTeamId, transactionsUsed }: Props) {
  const { token, leagueId } = useAuth()
  const [open, setOpen] = useState(false)

  const { data: allTxns = [] } = useQuery<TxnRecord[]>({
    queryKey: ['txn-log', leagueId],
    queryFn: () =>
      fetch(`${API_BASE}/api/leagues/${leagueId}/transactions`, {
        headers: authHeaders(token),
      }).then(r => r.json()),
    enabled: !!token && open,
  })

  const txns = allTxns.filter(t => t.teamId === targetTeamId)

  return (
    <div className="bg-slate-900 border border-slate-800 rounded-xl overflow-hidden">
      <button
        onClick={() => setOpen(o => !o)}
        className="w-full flex items-center justify-between px-4 py-3 hover:bg-slate-800/40 transition-colors"
      >
        <div className="flex items-center gap-2">
          <span className="text-xs font-semibold text-slate-400 uppercase tracking-wider">Transaction History</span>
          {transactionsUsed != null && (
            <span className="text-xs text-slate-600">{transactionsUsed} / 15 used</span>
          )}
        </div>
        <svg
          className={`w-4 h-4 text-slate-600 transition-transform ${open ? 'rotate-180' : ''}`}
          fill="none" stroke="currentColor" viewBox="0 0 24 24"
        >
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
        </svg>
      </button>

      {open && (
        txns.length === 0 ? (
          <p className="px-4 pb-4 text-xs text-slate-600">No transactions yet.</p>
        ) : (
          <table className="w-full text-sm border-t border-slate-800">
            <thead>
              <tr className="border-b border-slate-800">
                <th className="text-left px-4 py-2 text-xs font-semibold text-slate-500 uppercase tracking-wider">Dropped</th>
                <th className="text-left px-4 py-2 text-xs font-semibold text-slate-500 uppercase tracking-wider">Added</th>
                <th className="text-left px-4 py-2 text-xs font-semibold text-slate-500 uppercase tracking-wider hidden sm:table-cell">Type</th>
                <th className="text-right px-4 py-2 text-xs font-semibold text-slate-500 uppercase tracking-wider">Date</th>
              </tr>
            </thead>
            <tbody>
              {txns.map(t => (
                <tr key={t.id} className="border-b border-slate-800/50 hover:bg-slate-800/20">
                  <td className="px-4 py-2.5 text-red-400 text-xs">↓ {t.droppedPlayer}</td>
                  <td className="px-4 py-2.5 text-green-400 text-xs">↑ {t.addedPlayer}</td>
                  <td className="px-4 py-2.5 text-slate-500 text-xs hidden sm:table-cell">
                    {txnTypeLabel[t.txnType] ?? t.txnType}
                  </td>
                  <td className="px-4 py-2.5 text-right text-slate-500 text-xs whitespace-nowrap">
                    {fmtDate(t.createdAt)}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )
      )}
    </div>
  )
}
