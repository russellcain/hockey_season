import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { useAuth, API_BASE, authHeaders } from '../context/AuthContext'

interface MatchupTeam {
  id: number
  name: string
  manager: string
}

interface Matchup {
  id: number
  weekNumber: number
  homeTeamId: number
  homeTeam: MatchupTeam
  awayTeamId: number
  awayTeam: MatchupTeam
  homeScore: number
  awayScore: number
  homePoints: number
  awayPoints: number
}

function currentWeekNumber(): number {
  // Approximate week within NHL season (Oct = week 1)
  const now = new Date()
  const seasonStart = new Date(now.getFullYear(), 9, 1) // Oct 1
  if (now < seasonStart) return 1
  const diffDays = Math.floor((now.getTime() - seasonStart.getTime()) / (1000 * 60 * 60 * 24))
  return Math.floor(diffDays / 7) + 1
}

export function SchedulePage() {
  const { token, leagueId } = useAuth()
  const currentWeek = currentWeekNumber()
  const [openWeek, setOpenWeek] = useState<number>(currentWeek)

  const { data: matchups = [], isLoading, error } = useQuery<Matchup[]>({
    queryKey: ['schedule', leagueId],
    queryFn: () =>
      fetch(`${API_BASE}/api/leagues/${leagueId}/schedule`, {
        headers: authHeaders(token),
      }).then(r => {
        if (!r.ok) throw new Error(`HTTP ${r.status}`)
        return r.json()
      }),
    enabled: !!token,
  })

  if (isLoading) {
    return (
      <div className="flex h-64 items-center justify-center">
        <div className="text-slate-500 text-sm">Loading…</div>
      </div>
    )
  }
  if (error) {
    return (
      <div className="flex h-64 items-center justify-center">
        <div className="text-red-400 text-sm">{String(error)}</div>
      </div>
    )
  }

  // Group by week
  const byWeek = matchups.reduce<Record<number, Matchup[]>>((acc, m) => {
    if (!acc[m.weekNumber]) acc[m.weekNumber] = []
    acc[m.weekNumber].push(m)
    return acc
  }, {})

  const weeks = Object.keys(byWeek)
    .map(Number)
    .sort((a, b) => a - b)

  return (
    <div className="p-6 max-w-3xl mx-auto">
      <h1 className="text-xl font-bold text-slate-100 mb-4">Schedule</h1>

      {weeks.length === 0 && (
        <div className="text-slate-500 text-sm text-center py-12">
          Schedule not generated yet.
        </div>
      )}

      <div className="space-y-2">
        {weeks.map(week => {
          const isOpen = openWeek === week
          const isCurrent = week === currentWeek
          const weekMatchups = byWeek[week]

          return (
            <div key={week} className="bg-slate-900 border border-slate-800 rounded-xl overflow-hidden">
              <button
                onClick={() => setOpenWeek(isOpen ? -1 : week)}
                className="w-full flex items-center justify-between px-4 py-3 text-left hover:bg-slate-800/50 transition-colors"
              >
                <div className="flex items-center gap-3">
                  <span className="font-semibold text-slate-100 text-sm">Week {week}</span>
                  {isCurrent && (
                    <span className="text-xs px-2 py-0.5 rounded-full bg-blue-500/20 border border-blue-500/30 text-blue-400">
                      Current
                    </span>
                  )}
                </div>
                <svg
                  className={`w-4 h-4 text-slate-500 transition-transform ${isOpen ? 'rotate-180' : ''}`}
                  fill="none" stroke="currentColor" viewBox="0 0 24 24"
                >
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
                </svg>
              </button>

              {isOpen && (
                <div className="border-t border-slate-800">
                  {weekMatchups.map(m => (
                    <div key={m.id} className="flex items-center justify-between px-4 py-3 border-b border-slate-800/50 last:border-0">
                      <div className="flex-1 text-right">
                        <div className="font-semibold text-slate-100 text-sm">{m.homeTeam.name}</div>
                        <div className="text-xs text-slate-500">{m.homeTeam.manager}</div>
                      </div>

                      <div className="flex items-center gap-3 px-6">
                        {m.homePoints > 0 || m.awayPoints > 0 ? (
                          <span className="text-slate-200 font-mono text-sm">
                            {m.homeScore.toFixed(1)} – {m.awayScore.toFixed(1)}
                          </span>
                        ) : (
                          <span className="text-slate-600 text-xs">TBD</span>
                        )}
                      </div>

                      <div className="flex-1">
                        <div className="font-semibold text-slate-100 text-sm">{m.awayTeam.name}</div>
                        <div className="text-xs text-slate-500">{m.awayTeam.manager}</div>
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </div>
          )
        })}
      </div>
    </div>
  )
}
