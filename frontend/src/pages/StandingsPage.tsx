import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { useAuth, API_BASE, authHeaders } from '../context/AuthContext'

interface TeamStanding {
  team: { id: number; name: string; manager: string }
  totalPoints: number
  goals: number
  h2hPoints: number
  h2hWins: number
  h2hTies: number
  h2hLosses: number
}

interface StandingsData {
  aggregate: TeamStanding[]
  h2h: TeamStanding[]
}

type Tab = 'aggregate' | 'h2h'

export function StandingsPage() {
  const { token, leagueId } = useAuth()
  const [tab, setTab] = useState<Tab>('aggregate')

  const { data, isLoading, error } = useQuery<StandingsData>({
    queryKey: ['standings', leagueId],
    queryFn: () =>
      fetch(`${API_BASE}/api/leagues/${leagueId}/standings`, {
        headers: authHeaders(token),
      }).then(r => {
        if (!r.ok) throw new Error(`HTTP ${r.status}`)
        return r.json()
      }),
    enabled: !!token,
  })

  if (isLoading) return <PageLoading />
  if (error) return <PageError message={String(error)} />

  const rows = tab === 'aggregate' ? (data?.aggregate ?? []) : (data?.h2h ?? [])

  return (
    <div className="p-6 max-w-4xl mx-auto">
      <h1 className="text-xl font-bold text-slate-100 mb-4">Standings</h1>

      <div className="flex gap-1 mb-4 bg-slate-900 rounded-lg p-1 self-start inline-flex">
        {(['aggregate', 'h2h'] as Tab[]).map(t => (
          <button
            key={t}
            onClick={() => setTab(t)}
            className={[
              'px-4 py-1.5 rounded-md text-xs font-semibold transition-all',
              tab === t ? 'bg-slate-700 text-slate-100 shadow' : 'text-slate-500 hover:text-slate-300',
            ].join(' ')}
          >
            {t === 'aggregate' ? 'Aggregate' : 'Head-to-Head'}
          </button>
        ))}
      </div>

      <div className="bg-slate-900 border border-slate-800 rounded-xl overflow-hidden">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-slate-800">
              <th className="text-left px-4 py-3 text-xs font-semibold text-slate-400 uppercase tracking-wider">Rank</th>
              <th className="text-left px-4 py-3 text-xs font-semibold text-slate-400 uppercase tracking-wider">Team</th>
              {tab === 'aggregate' ? (
                <>
                  <th className="text-right px-4 py-3 text-xs font-semibold text-slate-400 uppercase tracking-wider">Pts</th>
                  <th className="text-right px-4 py-3 text-xs font-semibold text-slate-400 uppercase tracking-wider">Goals</th>
                </>
              ) : (
                <>
                  <th className="text-right px-4 py-3 text-xs font-semibold text-slate-400 uppercase tracking-wider">H2H Pts</th>
                  <th className="text-center px-4 py-3 text-xs font-semibold text-slate-400 uppercase tracking-wider">W-T-L</th>
                </>
              )}
            </tr>
          </thead>
          <tbody>
            {rows.map((row, i) => (
              <tr key={row.team.id} className="border-b border-slate-800/50 hover:bg-slate-800/30">
                <td className="px-4 py-3 text-slate-500 text-xs">{i + 1}</td>
                <td className="px-4 py-3">
                  <div className="font-semibold text-slate-100">{row.team.name}</div>
                  <div className="text-xs text-slate-500">{row.team.manager}</div>
                </td>
                {tab === 'aggregate' ? (
                  <>
                    <td className="px-4 py-3 text-right font-mono text-slate-200">{row.totalPoints.toFixed(1)}</td>
                    <td className="px-4 py-3 text-right font-mono text-slate-400">{row.goals}</td>
                  </>
                ) : (
                  <>
                    <td className="px-4 py-3 text-right font-mono text-slate-200">{row.h2hPoints}</td>
                    <td className="px-4 py-3 text-center font-mono text-slate-400">
                      {row.h2hWins}-{row.h2hTies}-{row.h2hLosses}
                    </td>
                  </>
                )}
              </tr>
            ))}
            {rows.length === 0 && (
              <tr>
                <td colSpan={4} className="px-4 py-8 text-center text-slate-600 text-sm">
                  No standings data yet.
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>
    </div>
  )
}

function PageLoading() {
  return (
    <div className="flex h-64 items-center justify-center">
      <div className="text-slate-500 text-sm">Loading…</div>
    </div>
  )
}

function PageError({ message }: { message: string }) {
  return (
    <div className="flex h-64 items-center justify-center">
      <div className="text-red-400 text-sm">{message}</div>
    </div>
  )
}
