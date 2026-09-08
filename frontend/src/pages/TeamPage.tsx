import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useAuth, API_BASE, authHeaders } from '../context/AuthContext'
import { TeamTransactionHistory } from '../components/TeamTransactionHistory'

interface Stats {
  goals: number
  assists: number
  wins: number
  otl: number
  shutouts: number
  gamesPlayed: number
  fp: number
}

interface Player {
  id: number
  name: string
  nhlTeam: string
  nhlTeamCode: string
  position: string
  salary: number
  stats: Stats
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

function StatCell({ player }: { player: Player }) {
  if (player.position === 'G') {
    return (
      <span className="font-mono text-xs text-slate-400">
        {player.stats.wins}W {player.stats.otl}OTL {player.stats.shutouts}SO
        <span className="text-blue-400 ml-2">{player.stats.fp.toFixed(0)}fp</span>
      </span>
    )
  }
  return (
    <span className="font-mono text-xs text-slate-400">
      {player.stats.goals}G {player.stats.assists}A
      <span className="text-blue-400 ml-2">{player.stats.fp.toFixed(0)}fp</span>
    </span>
  )
}

// ── Inline drop+replace flow ──────────────────────────────────────────────────

interface DropFlowProps {
  dropping: Player
  leagueId: string
  token: string | null
  teamId: number | null
  capUsed: number
  salaryCap: number
  onCancel: () => void
  onSuccess: () => void
}

function DropFlow({ dropping, leagueId, token, teamId, capUsed, salaryCap, onCancel, onSuccess }: DropFlowProps) {
  const [search, setSearch] = useState('')
  const [selected, setSelected] = useState<Player | null>(null)
  const [txError, setTxError] = useState<string | null>(null)
  const qc = useQueryClient()

  // Cap budget after dropping the player being released.
  const maxSalary = salaryCap - capUsed + dropping.salary

  const { data: available = [], isLoading } = useQuery<Player[]>({
    queryKey: ['available', leagueId, dropping.position, search],
    queryFn: () => {
      const p = new URLSearchParams({ available: 'true', position: dropping.position })
      p.set('max_salary', String(maxSalary))
      if (search) p.set('q', search)
      return fetch(`${API_BASE}/api/leagues/${leagueId}/players?${p}`, {
        headers: authHeaders(token),
      }).then(r => r.json())
    },
    enabled: !!token,
  })

  const transact = useMutation({
    mutationFn: () =>
      fetch(`${API_BASE}/api/leagues/${leagueId}/teams/${teamId}/transactions`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', ...authHeaders(token) },
        body: JSON.stringify({ dropPlayerId: dropping.id, addPlayerId: selected!.id }),
      }).then(r => r.json()),
    onSuccess: async data => {
      if (data.success) {
        await qc.refetchQueries({ queryKey: ['roster'] })
        qc.invalidateQueries({ queryKey: ['available'] })
        qc.invalidateQueries({ queryKey: ['txn-log'] })
        onSuccess()
      } else {
        setTxError(data.error ?? 'Transaction failed')
      }
    },
  })

  return (
    <div className="mt-2 bg-slate-800/60 border border-slate-700 rounded-xl p-4 space-y-3">
      <div className="flex items-center justify-between">
        <div className="text-sm font-semibold text-slate-100">
          Replace <span className="text-red-400">{dropping.name}</span>
        </div>
        <button onClick={onCancel} className="text-xs text-slate-500 hover:text-slate-300">Cancel</button>
      </div>
      <p className="text-xs text-slate-500">
        Pick a {dropping.position} — budget after drop: <span className="text-slate-300 font-mono">{fmt$(maxSalary)}</span>
      </p>

      <input
        autoFocus
        type="text"
        placeholder="Search free agents…"
        value={search}
        onChange={e => { setSearch(e.target.value); setSelected(null) }}
        className="w-full px-3 py-2 text-sm bg-slate-900 border border-slate-700 rounded-lg text-slate-100 placeholder-slate-600 focus:outline-none focus:ring-1 focus:ring-blue-500"
      />

      {isLoading ? (
        <div className="text-xs text-slate-500">Loading…</div>
      ) : available.length === 0 ? (
        <div className="text-xs text-slate-500">No eligible free agents.</div>
      ) : (
        <div className="max-h-48 overflow-y-auto space-y-1 pr-1">
          {available.map(p => (
            <button
              key={p.id}
              onClick={() => setSelected(p)}
              className={[
                'w-full flex items-center justify-between px-3 py-2 rounded-lg text-left transition-colors',
                selected?.id === p.id
                  ? 'bg-blue-600/20 border border-blue-500/30'
                  : 'hover:bg-slate-700/50 border border-transparent',
              ].join(' ')}
            >
              <div>
                <span className="text-sm text-slate-100">{p.name}</span>
                <span className="text-xs text-slate-500 ml-2">{p.nhlTeam}</span>
              </div>
              <span className="text-xs font-mono text-slate-400 shrink-0">{fmt$(p.salary)}</span>
            </button>
          ))}
        </div>
      )}

      {txError && <p className="text-xs text-red-400">{txError}</p>}

      <button
        disabled={!selected || transact.isPending}
        onClick={() => transact.mutate()}
        className="w-full py-2 text-sm font-semibold bg-red-600 hover:bg-red-500 disabled:opacity-30 text-white rounded-lg transition-colors"
      >
        {transact.isPending
          ? 'Processing…'
          : selected
          ? `Confirm: drop ${dropping.name} → add ${selected.name}`
          : 'Select a replacement above'}
      </button>
    </div>
  )
}

// ── Main page ─────────────────────────────────────────────────────────────────

export function TeamPage() {
  const { token, leagueId, teamId } = useAuth()
  const [dropping, setDropping] = useState<Player | null>(null)

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

  if (isLoading) return <div className="flex h-64 items-center justify-center"><div className="text-slate-500 text-sm">Loading…</div></div>
  if (error || !data) return <div className="flex h-64 items-center justify-center"><div className="text-red-400 text-sm">{String(error ?? 'No data')}</div></div>

  const capPct = data.salaryCap > 0 ? (data.capUsed / data.salaryCap) * 100 : 0
  const byPos: Record<string, RosterSlot[]> = { F: [], D: [], G: [] }
  for (const s of data.slots) {
    const key = s.player.position === 'D' ? 'D' : s.player.position === 'G' ? 'G' : 'F'
    byPos[key].push(s)
  }

  return (
    <div className="p-6 max-w-5xl mx-auto">
      <h1 className="text-xl font-bold text-slate-100 mb-4">My Team</h1>

      {/* Cap + counters */}
      <div className="grid grid-cols-3 gap-4 mb-6">
        <div className="bg-slate-900 border border-slate-800 rounded-xl p-4">
          <div className="text-xs text-slate-500 uppercase tracking-wider mb-2">Cap Usage</div>
          <div className="text-lg font-mono font-bold text-slate-100">
            {fmt$(data.capUsed)} <span className="text-slate-500 text-sm font-normal">/ {fmt$(data.salaryCap)}</span>
          </div>
          <div className="mt-2 h-1.5 bg-slate-800 rounded-full overflow-hidden">
            <div className={`h-full rounded-full transition-all ${capPct > 95 ? 'bg-red-500' : 'bg-blue-500'}`}
              style={{ width: `${Math.min(capPct, 100)}%` }} />
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

      {/* Roster by position */}
      {teamId != null && (
        <div className="mb-6">
          <TeamTransactionHistory targetTeamId={teamId} transactionsUsed={data.transactionsUsed} />
        </div>
      )}
      {(['F', 'D', 'G'] as const).map(pos => {
        const label = pos === 'F' ? 'Forwards' : pos === 'D' ? 'Defensemen' : 'Goalies'
        return (
          <section key={pos} className="mb-6">
            <h2 className="text-xs font-semibold text-slate-400 uppercase tracking-wider mb-2">{label}</h2>
            <div className="bg-slate-900 border border-slate-800 rounded-xl overflow-hidden">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b border-slate-800">
                    <th className="text-left px-4 py-2 text-xs font-semibold text-slate-500 uppercase tracking-wider w-8">Pos</th>
                    <th className="text-left px-4 py-2 text-xs font-semibold text-slate-500 uppercase tracking-wider">Player</th>
                    <th className="text-left px-4 py-2 text-xs font-semibold text-slate-500 uppercase tracking-wider hidden sm:table-cell">NHL Team</th>
                    <th className="text-left px-4 py-2 text-xs font-semibold text-slate-500 uppercase tracking-wider">Season Stats</th>
                    <th className="text-right px-4 py-2 text-xs font-semibold text-slate-500 uppercase tracking-wider hidden md:table-cell">Salary</th>
                    <th className="text-center px-4 py-2 text-xs font-semibold text-slate-500 uppercase tracking-wider">Status</th>
                    <th className="px-4 py-2 w-16" />
                  </tr>
                </thead>
                <tbody>
                  {byPos[pos].map(slot => (
                    <>
                      <tr key={slot.id} className={`border-b border-slate-800/50 ${dropping?.id === slot.player.id ? 'bg-red-500/5' : 'hover:bg-slate-800/30'}`}>
                        <td className="px-4 py-3">
                          <span className="text-xs font-bold text-slate-400">{slot.player.position}</span>
                        </td>
                        <td className="px-4 py-3 font-medium text-slate-100">{slot.player.name}</td>
                        <td className="px-4 py-3 text-slate-400 text-xs hidden sm:table-cell">{slot.player.nhlTeam}</td>
                        <td className="px-4 py-3"><StatCell player={slot.player} /></td>
                        <td className="px-4 py-3 text-right font-mono text-slate-300 text-xs hidden md:table-cell">{fmt$(slot.player.salary)}</td>
                        <td className="px-4 py-3 text-center">
                          <span className={`text-xs px-2 py-0.5 rounded-full ${slotBadge[slot.slotType] ?? ''}`}>
                            {slot.slotType}
                          </span>
                        </td>
                        <td className="px-4 py-3 text-right">
                          {slot.slotType !== 'substitute' && (
                            dropping?.id === slot.player.id ? (
                              <button
                                onClick={() => setDropping(null)}
                                className="text-xs text-slate-500 hover:text-slate-300"
                              >
                                Cancel
                              </button>
                            ) : (
                              <button
                                onClick={() => setDropping(slot.player)}
                                className="text-xs font-semibold text-red-400 hover:text-red-300 transition-colors"
                              >
                                Drop
                              </button>
                            )
                          )}
                        </td>
                      </tr>
                      {dropping?.id === slot.player.id && (
                        <tr key={`${slot.id}-drop`} className="bg-slate-900">
                          <td colSpan={7} className="px-4 pb-4">
                            <DropFlow
                              dropping={dropping}
                              leagueId={leagueId}
                              token={token}
                              teamId={teamId}
                              capUsed={data.capUsed}
                              salaryCap={data.salaryCap}
                              onCancel={() => setDropping(null)}
                              onSuccess={() => setDropping(null)}
                            />
                          </td>
                        </tr>
                      )}
                    </>
                  ))}
                  {byPos[pos].length === 0 && (
                    <tr><td colSpan={7} className="px-4 py-6 text-center text-slate-600 text-xs">No {label.toLowerCase()} on roster.</td></tr>
                  )}
                </tbody>
              </table>
            </div>
          </section>
        )
      })}
    </div>
  )
}
