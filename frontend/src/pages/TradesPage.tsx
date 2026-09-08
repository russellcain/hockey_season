import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useAuth, API_BASE, authHeaders } from '../context/AuthContext'

interface Player {
  id: number
  name: string
  position: string
  salary: number
}

interface TradeLeg {
  fromTeam: { id: number; name: string }
  toTeam: { id: number; name: string }
  player: Player
}

interface TradeDetail {
  id: number
  status: 'pending' | 'approved' | 'rejected'
  submittedByTeam: { id: number; name: string; manager: string }
  notes?: string
  legs: TradeLeg[]
  createdAt: string
}

interface RosterSlot {
  player: Player
  slotType: string
}

const statusColor: Record<string, string> = {
  pending: 'text-yellow-400 bg-yellow-500/10 border-yellow-500/20',
  approved: 'text-green-400 bg-green-500/10 border-green-500/20',
  rejected: 'text-red-400 bg-red-500/10 border-red-500/20',
}

function fmt$(n: number) {
  return '$' + (n / 1_000_000).toFixed(2) + 'M'
}

export function TradesPage() {
  const { token, leagueId, teamId } = useAuth()
  const qc = useQueryClient()
  const [showForm, setShowForm] = useState(false)
  const [toTeamId, setToTeamId] = useState('')
  const [fromPlayerIds, setFromPlayerIds] = useState<number[]>([])
  const [toPlayerIds, setToPlayerIds] = useState<number[]>([])
  const [formError, setFormError] = useState<string | null>(null)

  const { data: trades = [], isLoading } = useQuery<TradeDetail[]>({
    queryKey: ['trades', leagueId],
    queryFn: () =>
      fetch(`${API_BASE}/api/leagues/${leagueId}/trades`, { headers: authHeaders(token) })
        .then(r => r.json()),
    enabled: !!token,
  })

  // My roster for the "from" side
  const { data: myRoster } = useQuery<{ slots: RosterSlot[] }>({
    queryKey: ['roster', leagueId, teamId],
    queryFn: () =>
      fetch(`${API_BASE}/api/leagues/${leagueId}/teams/${teamId}/roster`, {
        headers: authHeaders(token),
      }).then(r => r.json()),
    enabled: !!token && !!teamId && showForm,
  })

  // Other team's roster for the "to" side
  const { data: otherRoster } = useQuery<{ slots: RosterSlot[] }>({
    queryKey: ['roster', leagueId, toTeamId],
    queryFn: () =>
      fetch(`${API_BASE}/api/leagues/${leagueId}/teams/${toTeamId}/roster`, {
        headers: authHeaders(token),
      }).then(r => r.json()),
    enabled: !!token && !!toTeamId && showForm,
  })

  const proposeMutation = useMutation({
    mutationFn: () =>
      fetch(`${API_BASE}/api/leagues/${leagueId}/trades`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', ...authHeaders(token) },
        body: JSON.stringify({ toTeamId: Number(toTeamId), fromPlayerIds, toPlayerIds }),
      }).then(r => r.json()),
    onSuccess: data => {
      if (data.success) {
        qc.invalidateQueries({ queryKey: ['trades'] })
        setShowForm(false)
        setFromPlayerIds([])
        setToPlayerIds([])
        setToTeamId('')
        setFormError(null)
      } else {
        setFormError(data.error ?? 'Trade failed')
      }
    },
    onError: () => setFormError('Network error'),
  })

  if (isLoading) {
    return (
      <div className="flex h-64 items-center justify-center">
        <div className="text-slate-500 text-sm">Loading…</div>
      </div>
    )
  }

  function togglePlayer(id: number, side: 'from' | 'to') {
    if (side === 'from') {
      setFromPlayerIds(prev => prev.includes(id) ? prev.filter(x => x !== id) : [...prev, id])
    } else {
      setToPlayerIds(prev => prev.includes(id) ? prev.filter(x => x !== id) : [...prev, id])
    }
  }

  return (
    <div className="p-6 max-w-4xl mx-auto">
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-xl font-bold text-slate-100">Trades</h1>
        <button
          onClick={() => setShowForm(!showForm)}
          className="px-4 py-2 text-sm font-semibold bg-blue-600 hover:bg-blue-500 text-white rounded-lg transition-colors"
        >
          {showForm ? 'Cancel' : 'Propose Trade'}
        </button>
      </div>

      {showForm && (
        <div className="bg-slate-900 border border-slate-800 rounded-xl p-5 mb-6 space-y-4">
          <h2 className="text-sm font-semibold text-slate-200">New Trade Proposal</h2>

          {formError && (
            <div className="p-3 rounded-lg bg-red-500/10 border border-red-500/20 text-red-400 text-sm">
              {formError}
            </div>
          )}

          <div>
            <label className="block text-xs font-semibold text-slate-400 uppercase tracking-wider mb-1.5">
              Other Team ID
            </label>
            <input
              type="number"
              value={toTeamId}
              onChange={e => { setToTeamId(e.target.value); setToPlayerIds([]) }}
              placeholder="Enter team ID"
              className="w-48 px-3 py-2 text-sm bg-slate-800 border border-slate-700 rounded-lg text-slate-100 placeholder-slate-600 focus:outline-none focus:border-slate-500"
            />
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div>
              <div className="text-xs font-semibold text-slate-400 uppercase tracking-wider mb-2">
                Players you send
              </div>
              {(myRoster?.slots ?? [])
                .filter(s => s.slotType === 'active')
                .map(s => (
                  <label key={s.player.id} className="flex items-center gap-2 py-1.5 cursor-pointer hover:bg-slate-800/50 rounded px-2">
                    <input
                      type="checkbox"
                      checked={fromPlayerIds.includes(s.player.id)}
                      onChange={() => togglePlayer(s.player.id, 'from')}
                      className="accent-blue-500"
                    />
                    <span className="text-sm text-slate-100">{s.player.name}</span>
                    <span className="text-xs text-slate-500 ml-auto">{fmt$(s.player.salary)}</span>
                  </label>
                ))}
            </div>

            <div>
              <div className="text-xs font-semibold text-slate-400 uppercase tracking-wider mb-2">
                Players you receive
              </div>
              {!toTeamId ? (
                <p className="text-slate-600 text-xs">Select the other team first.</p>
              ) : (
                (otherRoster?.slots ?? [])
                  .filter(s => s.slotType === 'active')
                  .map(s => (
                    <label key={s.player.id} className="flex items-center gap-2 py-1.5 cursor-pointer hover:bg-slate-800/50 rounded px-2">
                      <input
                        type="checkbox"
                        checked={toPlayerIds.includes(s.player.id)}
                        onChange={() => togglePlayer(s.player.id, 'to')}
                        className="accent-blue-500"
                      />
                      <span className="text-sm text-slate-100">{s.player.name}</span>
                      <span className="text-xs text-slate-500 ml-auto">{fmt$(s.player.salary)}</span>
                    </label>
                  ))
              )}
            </div>
          </div>

          <button
            disabled={!toTeamId || proposeMutation.isPending}
            onClick={() => proposeMutation.mutate()}
            className="px-4 py-2 text-sm font-semibold bg-blue-600 hover:bg-blue-500 disabled:opacity-40 disabled:cursor-not-allowed text-white rounded-lg transition-colors"
          >
            {proposeMutation.isPending ? 'Submitting…' : 'Submit Trade'}
          </button>
        </div>
      )}

      {/* Trade history */}
      <h2 className="text-sm font-semibold text-slate-400 uppercase tracking-wider mb-3">Trade History</h2>
      {trades.length === 0 ? (
        <p className="text-slate-600 text-sm">No trades yet.</p>
      ) : (
        <div className="space-y-3">
          {trades.map(trade => (
            <div key={trade.id} className="bg-slate-900 border border-slate-800 rounded-xl p-4">
              <div className="flex items-start justify-between mb-3">
                <div>
                  <span className="text-sm font-semibold text-slate-100">
                    Trade #{trade.id} — {trade.submittedByTeam.name}
                  </span>
                  <div className="text-xs text-slate-500 mt-0.5">{trade.createdAt.slice(0, 10)}</div>
                </div>
                <span className={`text-xs px-2 py-0.5 rounded-full border ${statusColor[trade.status] ?? ''}`}>
                  {trade.status}
                </span>
              </div>

              <div className="space-y-1">
                {(trade.legs ?? []).map((leg, i) => (
                  <div key={i} className="text-xs text-slate-400">
                    {leg.player.name} ({leg.player.position}) from {leg.fromTeam.name} → {leg.toTeam.name}
                  </div>
                ))}
              </div>

              {trade.notes && (
                <div className="mt-2 text-xs text-slate-500 italic">{trade.notes}</div>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
