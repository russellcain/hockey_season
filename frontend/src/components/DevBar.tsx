import { useState } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { useAuth, API_BASE, authHeaders } from '../context/AuthContext'

interface TopScorer {
  name: string
  team: string
  position: string
  points: number
}

interface MatchupResult {
  id: number
  homeTeam: string
  awayTeam: string
  homeScore: number
  awayScore: number
  homePoints: number
  awayPoints: number
  topScorers: TopScorer[]
}

interface AdvanceResp {
  week: number
  weekStart: string
  weekEnd: string
  logsSeeded: number
  matchups: MatchupResult[]
  newInjuries: string[]
  clearedInjuries: string[]
}

export function DevBar() {
  const { token, leagueId } = useAuth()
  const qc = useQueryClient()
  const [loading, setLoading] = useState(false)
  const [result, setResult] = useState<AdvanceResp | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [expanded, setExpanded] = useState(false)

  async function advance() {
    setLoading(true)
    setError(null)
    try {
      const res = await fetch(`${API_BASE}/api/dev/leagues/${leagueId}/advance-week`, {
        method: 'POST',
        headers: authHeaders(token),
      })
      if (!res.ok) {
        const text = await res.text()
        setError(text.trim() || `HTTP ${res.status}`)
        return
      }
      const data: AdvanceResp = await res.json()
      setResult(data)
      setExpanded(true)
      // Invalidate all queries so standings, schedule, injuries refresh automatically.
      qc.invalidateQueries()
    } catch (e) {
      setError(String(e))
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="fixed bottom-0 left-0 right-0 z-50 border-t border-yellow-500/30 bg-slate-950/95 backdrop-blur">
      {/* Main bar */}
      <div className="flex items-center gap-3 px-4 py-2">
        <span className="text-xs font-bold text-yellow-400 uppercase tracking-widest shrink-0">⚙ Dev</span>

        {result && (
          <span className="text-xs text-slate-400 shrink-0">
            Last: <span className="text-slate-200 font-medium">Week {result.week}</span>
            {' '}({result.weekStart} – {result.weekEnd}, {result.logsSeeded} logs)
          </span>
        )}

        <button
          onClick={advance}
          disabled={loading}
          className="ml-auto shrink-0 px-4 py-1.5 text-xs font-bold bg-yellow-500 hover:bg-yellow-400 disabled:opacity-40 text-slate-900 rounded-md transition-colors"
        >
          {loading ? 'Advancing…' : result ? `→ Advance Week ${result.week + 1}` : '→ Advance Week 1'}
        </button>

        {result && (
          <button
            onClick={() => setExpanded(e => !e)}
            className="shrink-0 text-xs text-slate-500 hover:text-slate-300 transition-colors"
          >
            {expanded ? 'Hide ▲' : 'Show ▼'}
          </button>
        )}
      </div>

      {error && (
        <div className="px-4 pb-2 text-xs text-red-400">{error}</div>
      )}

      {/* Results panel */}
      {result && expanded && (
        <div className="border-t border-slate-800 px-4 py-3 max-h-72 overflow-y-auto space-y-4">

          {/* Matchups */}
          <div>
            <div className="text-xs font-semibold text-slate-400 uppercase tracking-wider mb-2">
              Week {result.week} Matchups
            </div>
            <div className="space-y-2">
              {(result.matchups ?? []).map(m => (
                <div key={m.id} className="bg-slate-900 rounded-lg p-3">
                  <div className="flex items-center justify-between mb-2">
                    <div className="text-sm">
                      <span className={m.homePoints > m.awayPoints ? 'text-slate-100 font-semibold' : 'text-slate-400'}>
                        {m.homeTeam}
                      </span>
                      <span className="text-slate-500 mx-2 font-mono text-xs">
                        {m.homeScore.toFixed(1)} – {m.awayScore.toFixed(1)}
                      </span>
                      <span className={m.awayPoints > m.homePoints ? 'text-slate-100 font-semibold' : 'text-slate-400'}>
                        {m.awayTeam}
                      </span>
                    </div>
                    <div className="text-xs text-slate-500 font-mono shrink-0">
                      {m.homePoints}–{m.awayPoints} pts
                    </div>
                  </div>
                  {(m.topScorers ?? []).length > 0 && (
                    <div className="flex flex-wrap gap-1.5">
                      {m.topScorers.map((s, i) => (
                        <span key={i} className="text-[11px] px-2 py-0.5 rounded-full bg-slate-800 text-slate-300">
                          {s.name} <span className="text-slate-500">{s.position} · {s.team.split(' ').pop()}</span>{' '}
                          <span className="text-yellow-400 font-semibold">{s.points}fp</span>
                        </span>
                      ))}
                    </div>
                  )}
                </div>
              ))}
            </div>
          </div>

          {/* Injuries */}
          {((result.newInjuries ?? []).length > 0 || (result.clearedInjuries ?? []).length > 0) && (
            <div>
              <div className="text-xs font-semibold text-slate-400 uppercase tracking-wider mb-2">Injury Changes</div>
              <div className="space-y-1">
                {(result.newInjuries ?? []).map((p, i) => (
                  <div key={i} className="text-xs text-red-400">🚑 {p} flagged as injured</div>
                ))}
                {(result.clearedInjuries ?? []).map((p, i) => (
                  <div key={i} className="text-xs text-green-400">✅ {p} cleared</div>
                ))}
              </div>
            </div>
          )}
        </div>
      )}
    </div>
  )
}
