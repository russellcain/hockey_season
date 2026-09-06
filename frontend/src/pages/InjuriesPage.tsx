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

interface InjuryInfo {
  injuredPlayer: Player
  teamId: number
  substitutePlayer?: Player
  capCeiling: number
}

function fmt$(n: number) {
  return '$' + (n / 1_000_000).toFixed(2) + 'M'
}

export function InjuriesPage() {
  const { token, leagueId, teamId } = useAuth()
  const qc = useQueryClient()
  const [expandedInjury, setExpandedInjury] = useState<number | null>(null)
  const [subError, setSubError] = useState<string | null>(null)

  const { data: injuries = [], isLoading } = useQuery<InjuryInfo[]>({
    queryKey: ['injuries', leagueId],
    queryFn: () =>
      fetch(`${API_BASE}/api/leagues/${leagueId}/injuries`, {
        headers: authHeaders(token),
      }).then(r => r.json()),
    enabled: !!token,
  })

  const { data: eligibleSubs = [] } = useQuery<Player[]>({
    queryKey: ['eligible-subs', leagueId, teamId, expandedInjury],
    queryFn: () =>
      fetch(
        `${API_BASE}/api/leagues/${leagueId}/teams/${teamId}/injury-subs/available?injuredPlayerId=${expandedInjury}`,
        { headers: authHeaders(token) }
      ).then(r => r.json()),
    enabled: !!token && !!expandedInjury && !!teamId,
  })

  const subMutation = useMutation({
    mutationFn: ({ injuredId, subId }: { injuredId: number; subId: number }) =>
      fetch(`${API_BASE}/api/leagues/${leagueId}/teams/${teamId}/injury-subs`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', ...authHeaders(token) },
        body: JSON.stringify({ injuredPlayerId: injuredId, subPlayerId: subId }),
      }).then(r => r.json()),
    onSuccess: data => {
      if (data.success) {
        qc.invalidateQueries({ queryKey: ['injuries'] })
        qc.invalidateQueries({ queryKey: ['roster'] })
        setExpandedInjury(null)
        setSubError(null)
      } else {
        setSubError(data.error ?? 'Sub failed')
      }
    },
    onError: () => setSubError('Network error'),
  })

  if (isLoading) {
    return (
      <div className="flex h-64 items-center justify-center">
        <div className="text-slate-500 text-sm">Loading…</div>
      </div>
    )
  }

  const myInjuries = injuries.filter(i => i.teamId === teamId)
  const otherInjuries = injuries.filter(i => i.teamId !== teamId)

  return (
    <div className="p-6 max-w-4xl mx-auto">
      <h1 className="text-xl font-bold text-slate-100 mb-6">Injuries</h1>

      {subError && (
        <div className="mb-4 p-3 rounded-lg bg-red-500/10 border border-red-500/20 text-red-400 text-sm">
          {subError}
        </div>
      )}

      {/* My team injuries */}
      <section className="mb-8">
        <h2 className="text-sm font-semibold text-slate-400 uppercase tracking-wider mb-3">My Team</h2>
        {myInjuries.length === 0 ? (
          <p className="text-slate-600 text-sm">No injuries on your team.</p>
        ) : (
          <div className="space-y-3">
            {myInjuries.map(inj => (
              <div key={inj.injuredPlayer.id} className="bg-slate-900 border border-slate-800 rounded-xl p-4">
                <div className="flex items-start justify-between">
                  <div>
                    <div className="flex items-center gap-2 mb-1">
                      <span className="font-semibold text-slate-100">{inj.injuredPlayer.name}</span>
                      <span className="text-xs px-2 py-0.5 rounded-full bg-red-500/10 text-red-400">Injured</span>
                      <span className="text-xs text-slate-500">{inj.injuredPlayer.position} · {fmt$(inj.injuredPlayer.salary)}</span>
                    </div>
                    <div className="text-xs text-slate-500">Cap ceiling for sub: {fmt$(inj.capCeiling)}</div>
                    {inj.substitutePlayer && (
                      <div className="mt-2 text-xs text-yellow-400">
                        Sub: {inj.substitutePlayer.name} ({fmt$(inj.substitutePlayer.salary)})
                      </div>
                    )}
                  </div>
                  {!inj.substitutePlayer && (
                    <button
                      onClick={() => setExpandedInjury(
                        expandedInjury === inj.injuredPlayer.id ? null : inj.injuredPlayer.id
                      )}
                      className="px-3 py-1.5 text-xs font-semibold bg-blue-600/20 hover:bg-blue-600/30 text-blue-400 rounded-md transition-colors"
                    >
                      Pick Sub
                    </button>
                  )}
                </div>

                {expandedInjury === inj.injuredPlayer.id && (
                  <div className="mt-4 border-t border-slate-800 pt-4">
                    <div className="text-xs text-slate-500 mb-2">Eligible substitutes (same position, cap ≤ {fmt$(inj.capCeiling)})</div>
                    {eligibleSubs.length === 0 ? (
                      <p className="text-slate-600 text-xs">No eligible players available.</p>
                    ) : (
                      <div className="space-y-1">
                        {eligibleSubs.map(p => (
                          <div key={p.id} className="flex items-center justify-between py-2 px-3 rounded-lg hover:bg-slate-800/50">
                            <span className="text-sm text-slate-100">{p.name}</span>
                            <div className="flex items-center gap-3">
                              <span className="text-xs font-mono text-slate-400">{fmt$(p.salary)}</span>
                              <button
                                disabled={subMutation.isPending}
                                onClick={() => subMutation.mutate({ injuredId: inj.injuredPlayer.id, subId: p.id })}
                                className="px-3 py-1 text-xs font-semibold bg-green-600/20 hover:bg-green-600/30 text-green-400 rounded-md transition-colors disabled:opacity-40"
                              >
                                Select
                              </button>
                            </div>
                          </div>
                        ))}
                      </div>
                    )}
                  </div>
                )}
              </div>
            ))}
          </div>
        )}
      </section>

      {/* League-wide injuries */}
      {otherInjuries.length > 0 && (
        <section>
          <h2 className="text-sm font-semibold text-slate-400 uppercase tracking-wider mb-3">League-Wide</h2>
          <div className="bg-slate-900 border border-slate-800 rounded-xl overflow-hidden">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-slate-800">
                  <th className="text-left px-4 py-3 text-xs font-semibold text-slate-400 uppercase tracking-wider">Player</th>
                  <th className="text-left px-4 py-3 text-xs font-semibold text-slate-400 uppercase tracking-wider">Substitute</th>
                </tr>
              </thead>
              <tbody>
                {otherInjuries.map(inj => (
                  <tr key={inj.injuredPlayer.id} className="border-b border-slate-800/50">
                    <td className="px-4 py-3">
                      <span className="font-medium text-slate-100">{inj.injuredPlayer.name}</span>
                      <span className="ml-2 text-xs text-slate-500">{inj.injuredPlayer.position}</span>
                    </td>
                    <td className="px-4 py-3 text-slate-400 text-sm">
                      {inj.substitutePlayer ? inj.substitutePlayer.name : '—'}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </section>
      )}
    </div>
  )
}
