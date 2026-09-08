import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
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
  playerId: number
  player: Player
  slotType: string
}

interface RosterData {
  slots: RosterSlot[]
  capUsed: number
  salaryCap: number
  transactionsUsed: number
}

interface TxnRecord {
  id: number
  teamId: number
  teamName: string
  droppedPlayer: string
  addedPlayer: string
  txnType: string
  createdAt: string
}

const MAX_TRANSACTIONS = 15

function fmt$(n: number) {
  return '$' + (n / 1_000_000).toFixed(2) + 'M'
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

const txnTypeLabel: Record<string, string> = {
  elective: 'Drop/Add',
  injury_sub: 'Injury Sub',
  cut: 'Cut',
}

export function TransactionsPage() {
  const { token, leagueId, teamId } = useAuth()
  const qc = useQueryClient()
  const [step, setStep] = useState<'drop' | 'add'>('drop')
  const [dropPlayer, setDropPlayer] = useState<Player | null>(null)
  const [searchQ, setSearchQ] = useState('')
  const [error, setError] = useState<string | null>(null)

  const { data: rosterData, isLoading } = useQuery<RosterData>({
    queryKey: ['roster', leagueId, teamId],
    queryFn: () =>
      fetch(`${API_BASE}/api/leagues/${leagueId}/teams/${teamId}/roster`, {
        headers: authHeaders(token),
      }).then(r => r.json()),
    enabled: !!token && !!teamId,
  })

  const { data: available = [] } = useQuery<Player[]>({
    queryKey: ['available', leagueId, dropPlayer?.position, searchQ],
    queryFn: () => {
      const params = new URLSearchParams({ available: 'true' })
      if (dropPlayer?.position) params.set('position', dropPlayer.position)
      if (searchQ) params.set('q', searchQ)
      return fetch(`${API_BASE}/api/leagues/${leagueId}/players?${params}`, {
        headers: authHeaders(token),
      }).then(r => r.json())
    },
    enabled: !!token && step === 'add' && !!dropPlayer,
  })

  const { data: txnLog = [] } = useQuery<TxnRecord[]>({
    queryKey: ['txn-log', leagueId],
    queryFn: () =>
      fetch(`${API_BASE}/api/leagues/${leagueId}/transactions`, {
        headers: authHeaders(token),
      }).then(r => r.json()),
    enabled: !!token,
  })

  const mutation = useMutation({
    mutationFn: (addPlayer: Player) =>
      fetch(`${API_BASE}/api/leagues/${leagueId}/teams/${teamId}/transactions`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', ...authHeaders(token) },
        body: JSON.stringify({ dropPlayerId: dropPlayer!.id, addPlayerId: addPlayer.id }),
      }).then(r => r.json()),
    onSuccess: async (data) => {
      if (data.success) {
        await qc.refetchQueries({ queryKey: ['roster'] })
        qc.invalidateQueries({ queryKey: ['available'] })
        qc.invalidateQueries({ queryKey: ['txn-log'] })
        setStep('drop')
        setDropPlayer(null)
        setError(null)
      } else {
        setError(data.error ?? 'Transaction failed')
      }
    },
    onError: () => setError('Network error'),
  })

  if (isLoading) {
    return <div className="flex h-64 items-center justify-center"><div className="text-slate-500 text-sm">Loading…</div></div>
  }

  const txnUsed = rosterData?.transactionsUsed ?? 0
  const capUsed = rosterData?.capUsed ?? 0
  const cap = rosterData?.salaryCap ?? 104_000_000
  const exhausted = txnUsed >= MAX_TRANSACTIONS
  const capAfterDrop = dropPlayer ? capUsed - dropPlayer.salary : capUsed
  const capRemaining = cap - capAfterDrop

  return (
    <div className="p-6 max-w-4xl mx-auto space-y-8">

      {/* ── Make a transaction ── */}
      <section>
        <div className="flex items-center justify-between mb-4">
          <h1 className="text-xl font-bold text-slate-100">Transactions</h1>
          <span className={`text-sm ${exhausted ? 'text-red-400' : 'text-slate-400'}`}>
            {txnUsed} / {MAX_TRANSACTIONS} used this season
          </span>
        </div>

        {error && (
          <div className="mb-4 p-3 rounded-lg bg-red-500/10 border border-red-500/20 text-red-400 text-sm">{error}</div>
        )}
        {exhausted && (
          <div className="mb-4 p-3 rounded-lg bg-orange-500/10 border border-orange-500/20 text-orange-400 text-sm">
            You have used all {MAX_TRANSACTIONS} transactions this season.
          </div>
        )}

        {/* Step indicator */}
        <div className="flex gap-2 mb-5">
          {(['drop', 'add'] as const).map((s, i) => (
            <div key={s} className="flex items-center gap-2">
              <div className={`w-6 h-6 rounded-full flex items-center justify-center text-xs font-bold
                ${step === s ? 'bg-blue-600 text-white' : 'bg-slate-800 text-slate-500'}`}>
                {i + 1}
              </div>
              <span className={`text-sm ${step === s ? 'text-slate-100' : 'text-slate-600'}`}>
                {s === 'drop' ? 'Select player to drop' : 'Select player to add'}
              </span>
              {i === 0 && <span className="text-slate-700 mx-1">→</span>}
            </div>
          ))}
        </div>

        {step === 'drop' && (
          <div className="bg-slate-900 border border-slate-800 rounded-xl overflow-hidden">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-slate-800">
                  <th className="text-left px-4 py-3 text-xs font-semibold text-slate-400 uppercase tracking-wider">Player</th>
                  <th className="text-left px-4 py-3 text-xs font-semibold text-slate-400 uppercase tracking-wider">Pos</th>
                  <th className="text-right px-4 py-3 text-xs font-semibold text-slate-400 uppercase tracking-wider">Salary</th>
                  <th className="px-4 py-3" />
                </tr>
              </thead>
              <tbody>
                {(rosterData?.slots ?? [])
                  .filter(s => s.slotType === 'active')
                  .map(slot => (
                    <tr key={slot.id} className="border-b border-slate-800/50 hover:bg-slate-800/30">
                      <td className="px-4 py-3 font-medium text-slate-100">{slot.player.name}</td>
                      <td className="px-4 py-3 text-slate-400 text-xs">{slot.player.position}</td>
                      <td className="px-4 py-3 text-right font-mono text-slate-300">{fmt$(slot.player.salary)}</td>
                      <td className="px-4 py-3">
                        <button
                          disabled={exhausted}
                          onClick={() => { setDropPlayer(slot.player); setStep('add') }}
                          className="px-3 py-1 text-xs font-semibold bg-red-600/20 hover:bg-red-600/30 text-red-400 rounded-md transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
                        >
                          Drop
                        </button>
                      </td>
                    </tr>
                  ))}
              </tbody>
            </table>
          </div>
        )}

        {step === 'add' && dropPlayer && (
          <div className="space-y-4">
            <div className="flex items-center justify-between">
              <div className="text-sm text-slate-400">
                Dropping: <span className="font-semibold text-slate-100">{dropPlayer.name}</span>
                <span className="ml-2 text-slate-500">({dropPlayer.position} · {fmt$(dropPlayer.salary)})</span>
              </div>
              <div className="text-sm text-slate-400">
                Cap available after drop: <span className="font-semibold text-blue-400">{fmt$(capRemaining)}</span>
              </div>
            </div>
            <button
              onClick={() => { setStep('drop'); setDropPlayer(null) }}
              className="text-xs text-slate-500 underline"
            >
              ← Back
            </button>
            <input
              type="text"
              placeholder="Search players…"
              value={searchQ}
              onChange={e => setSearchQ(e.target.value)}
              className="w-full px-3 py-2.5 text-sm bg-slate-900 border border-slate-800 rounded-lg text-slate-100 placeholder-slate-600 focus:outline-none focus:border-slate-600"
            />
            <div className="bg-slate-900 border border-slate-800 rounded-xl overflow-hidden">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b border-slate-800">
                    <th className="text-left px-4 py-3 text-xs font-semibold text-slate-400 uppercase tracking-wider">Player</th>
                    <th className="text-left px-4 py-3 text-xs font-semibold text-slate-400 uppercase tracking-wider">NHL Team</th>
                    <th className="text-right px-4 py-3 text-xs font-semibold text-slate-400 uppercase tracking-wider">Salary</th>
                    <th className="text-right px-4 py-3 text-xs font-semibold text-slate-400 uppercase tracking-wider">New Cap</th>
                    <th className="px-4 py-3" />
                  </tr>
                </thead>
                <tbody>
                  {available.map(p => {
                    const newCap = capAfterDrop + p.salary
                    const overCap = newCap > cap
                    return (
                      <tr key={p.id} className={`border-b border-slate-800/50 ${overCap ? 'opacity-40' : 'hover:bg-slate-800/30'}`}>
                        <td className="px-4 py-3 font-medium text-slate-100">{p.name}</td>
                        <td className="px-4 py-3 text-slate-400 text-xs">{p.nhlTeam}</td>
                        <td className="px-4 py-3 text-right font-mono text-slate-300">{fmt$(p.salary)}</td>
                        <td className={`px-4 py-3 text-right font-mono text-xs ${overCap ? 'text-red-400' : 'text-green-400'}`}>
                          {fmt$(newCap)}
                        </td>
                        <td className="px-4 py-3">
                          <button
                            disabled={overCap || mutation.isPending}
                            onClick={() => mutation.mutate(p)}
                            className="px-3 py-1 text-xs font-semibold bg-green-600/20 hover:bg-green-600/30 text-green-400 rounded-md transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
                          >
                            Add
                          </button>
                        </td>
                      </tr>
                    )
                  })}
                  {available.length === 0 && (
                    <tr>
                      <td colSpan={5} className="px-4 py-8 text-center text-slate-600 text-sm">No available players matching filters.</td>
                    </tr>
                  )}
                </tbody>
              </table>
            </div>
          </div>
        )}
      </section>

      {/* ── League-wide transaction log ── */}
      <section>
        <h2 className="text-sm font-semibold text-slate-400 uppercase tracking-wider mb-3">
          League Activity
        </h2>
        {txnLog.length === 0 ? (
          <p className="text-slate-600 text-sm">No transactions yet.</p>
        ) : (
          <div className="bg-slate-900 border border-slate-800 rounded-xl overflow-hidden">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-slate-800">
                  <th className="text-left px-4 py-3 text-xs font-semibold text-slate-400 uppercase tracking-wider">Team</th>
                  <th className="text-left px-4 py-3 text-xs font-semibold text-slate-400 uppercase tracking-wider">Dropped</th>
                  <th className="text-left px-4 py-3 text-xs font-semibold text-slate-400 uppercase tracking-wider">Added</th>
                  <th className="text-left px-4 py-3 text-xs font-semibold text-slate-400 uppercase tracking-wider hidden sm:table-cell">Type</th>
                  <th className="text-right px-4 py-3 text-xs font-semibold text-slate-400 uppercase tracking-wider">Date</th>
                </tr>
              </thead>
              <tbody>
                {txnLog.map(t => (
                  <tr key={t.id} className="border-b border-slate-800/50 hover:bg-slate-800/20">
                    <td className="px-4 py-3">
                      <span className={`font-medium ${t.teamId === teamId ? 'text-blue-400' : 'text-slate-100'}`}>
                        {t.teamName}
                        {t.teamId === teamId && <span className="ml-1 text-xs text-blue-500">(you)</span>}
                      </span>
                    </td>
                    <td className="px-4 py-3 text-red-400 text-xs">
                      ↓ {t.droppedPlayer}
                    </td>
                    <td className="px-4 py-3 text-green-400 text-xs">
                      ↑ {t.addedPlayer}
                    </td>
                    <td className="px-4 py-3 text-slate-500 text-xs hidden sm:table-cell">
                      {txnTypeLabel[t.txnType] ?? t.txnType}
                    </td>
                    <td className="px-4 py-3 text-right text-slate-500 text-xs whitespace-nowrap">
                      {fmtDate(t.createdAt)}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </section>
    </div>
  )
}
