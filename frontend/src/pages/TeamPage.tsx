import { useQuery } from '@tanstack/react-query'
import { useAuth, API_BASE, authHeaders } from '../context/AuthContext'

interface Player {
  id: number
  name: string
  nhlTeam: string
  position: string
  salary: number
}

interface RosterSlot {
  id: number
  teamId: number
  playerId: number
  player: Player
  slotType: 'active' | 'injured' | 'substitute'
  originalPlayerId?: number
}

interface RosterData {
  slots: RosterSlot[]
  capUsed: number
  salaryCap: number
  transactionsUsed: number
  tradesUsed: number
}

const MAX_TRANSACTIONS = 15
const MAX_TRADES = 3

function fmt$(n: number) {
  return '$' + (n / 1_000_000).toFixed(2) + 'M'
}

const slotBadge: Record<string, string> = {
  active: 'text-green-400 bg-green-500/10',
  injured: 'text-red-400 bg-red-500/10',
  substitute: 'text-yellow-400 bg-yellow-500/10',
}

export function TeamPage() {
  const { token, leagueId, teamId } = useAuth()

  const { data, isLoading, error } = useQuery<RosterData>({
    queryKey: ['roster', leagueId, teamId],
    queryFn: () =>
      fetch(`${API_BASE}/api/leagues/${leagueId}/teams/${teamId}/roster`, {
        headers: authHeaders(token),
      }).then(r => {
        if (!r.ok) throw new Error(`HTTP ${r.status}`)
        return r.json()
      }),
    enabled: !!token && !!teamId,
  })

  if (isLoading) {
    return (
      <div className="flex h-64 items-center justify-center">
        <div className="text-slate-500 text-sm">Loading…</div>
      </div>
    )
  }
  if (error || !data) {
    return (
      <div className="flex h-64 items-center justify-center">
        <div className="text-red-400 text-sm">{String(error ?? 'No data')}</div>
      </div>
    )
  }

  const capPct = data.salaryCap > 0 ? (data.capUsed / data.salaryCap) * 100 : 0

  return (
    <div className="p-6 max-w-4xl mx-auto">
      <h1 className="text-xl font-bold text-slate-100 mb-4">My Team</h1>

      {/* Cap + counters */}
      <div className="grid grid-cols-3 gap-4 mb-6">
        <div className="bg-slate-900 border border-slate-800 rounded-xl p-4">
          <div className="text-xs text-slate-500 uppercase tracking-wider mb-2">Cap Usage</div>
          <div className="text-lg font-mono font-bold text-slate-100">
            {fmt$(data.capUsed)} <span className="text-slate-500 text-sm font-normal">/ {fmt$(data.salaryCap)}</span>
          </div>
          <div className="mt-2 h-1.5 bg-slate-800 rounded-full overflow-hidden">
            <div
              className={`h-full rounded-full transition-all ${capPct > 95 ? 'bg-red-500' : 'bg-blue-500'}`}
              style={{ width: `${Math.min(capPct, 100)}%` }}
            />
          </div>
        </div>
        <div className="bg-slate-900 border border-slate-800 rounded-xl p-4">
          <div className="text-xs text-slate-500 uppercase tracking-wider mb-2">Transactions</div>
          <div className="text-lg font-bold text-slate-100">
            {data.transactionsUsed} <span className="text-slate-500 text-sm font-normal">/ {MAX_TRANSACTIONS}</span>
          </div>
        </div>
        <div className="bg-slate-900 border border-slate-800 rounded-xl p-4">
          <div className="text-xs text-slate-500 uppercase tracking-wider mb-2">Trades</div>
          <div className="text-lg font-bold text-slate-100">
            {data.tradesUsed} <span className="text-slate-500 text-sm font-normal">/ {MAX_TRADES}</span>
          </div>
        </div>
      </div>

      {/* Roster table */}
      <div className="bg-slate-900 border border-slate-800 rounded-xl overflow-hidden">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-slate-800">
              <th className="text-left px-4 py-3 text-xs font-semibold text-slate-400 uppercase tracking-wider">Pos</th>
              <th className="text-left px-4 py-3 text-xs font-semibold text-slate-400 uppercase tracking-wider">Player</th>
              <th className="text-left px-4 py-3 text-xs font-semibold text-slate-400 uppercase tracking-wider">NHL Team</th>
              <th className="text-right px-4 py-3 text-xs font-semibold text-slate-400 uppercase tracking-wider">Salary</th>
              <th className="text-center px-4 py-3 text-xs font-semibold text-slate-400 uppercase tracking-wider">Status</th>
            </tr>
          </thead>
          <tbody>
            {data.slots.map(slot => (
              <tr key={slot.id} className="border-b border-slate-800/50 hover:bg-slate-800/30">
                <td className="px-4 py-3">
                  <span className="text-xs font-bold text-slate-400">{slot.player.position}</span>
                </td>
                <td className="px-4 py-3 font-medium text-slate-100">{slot.player.name}</td>
                <td className="px-4 py-3 text-slate-400 text-xs">{slot.player.nhlTeam}</td>
                <td className="px-4 py-3 text-right font-mono text-slate-300">{fmt$(slot.player.salary)}</td>
                <td className="px-4 py-3 text-center">
                  <span className={`text-xs px-2 py-0.5 rounded-full ${slotBadge[slot.slotType] ?? ''}`}>
                    {slot.slotType}
                  </span>
                </td>
              </tr>
            ))}
            {data.slots.length === 0 && (
              <tr>
                <td colSpan={5} className="px-4 py-8 text-center text-slate-600 text-sm">
                  No players on roster.
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>
    </div>
  )
}
