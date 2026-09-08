import { useParams, Navigate } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
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
  position: string
  salary: number
  stats: Stats
}

interface RosterSlot {
  id: number
  player: Player
  slotType: string
}

interface RosterData {
  slots: RosterSlot[]
  capUsed: number
  salaryCap: number
}

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

export function TeamDetailPage() {
  const { teamId: teamIdParam } = useParams<{ teamId: string }>()
  const { token, leagueId, teamId: myTeamId } = useAuth()
  const targetTeamId = Number(teamIdParam)

  // Redirect to My Team if viewing own team.
  if (myTeamId != null && targetTeamId === myTeamId) {
    return <Navigate to="/team" replace />
  }

  const { data, isLoading, error } = useQuery<RosterData>({
    queryKey: ['roster', leagueId, targetTeamId],
    queryFn: () =>
      fetch(`${API_BASE}/api/leagues/${leagueId}/teams/${targetTeamId}/roster`, {
        headers: authHeaders(token),
      }).then(r => {
        if (!r.ok) throw new Error(`HTTP ${r.status}`)
        return r.json()
      }),
    enabled: !!token && !!targetTeamId,
  })

  if (isLoading) return <div className="flex h-64 items-center justify-center"><div className="text-slate-500 text-sm">Loading…</div></div>
  if (error || !data) return <div className="flex h-64 items-center justify-center"><div className="text-red-400 text-sm">{String(error ?? 'No data')}</div></div>

  const byPos: Record<string, RosterSlot[]> = { F: [], D: [], G: [] }
  for (const s of data.slots) {
    const key = s.player.position === 'D' ? 'D' : s.player.position === 'G' ? 'G' : 'F'
    byPos[key].push(s)
  }

  const capPct = data.salaryCap > 0 ? (data.capUsed / data.salaryCap) * 100 : 0

  return (
    <div className="p-6 max-w-5xl mx-auto">
      <div className="flex items-center gap-3 mb-4">
        <h1 className="text-xl font-bold text-slate-100">Team Roster</h1>
        <span className="text-xs px-2 py-1 rounded-md bg-slate-800 text-slate-400">View only</span>
      </div>

      <div className="grid grid-cols-2 gap-4 mb-6">
        <div className="bg-slate-900 border border-slate-800 rounded-xl p-4">
          <div className="text-xs text-slate-500 uppercase tracking-wider mb-2">Cap Usage</div>
          <div className="text-lg font-mono font-bold text-slate-100">
            {fmt$(data.capUsed)} <span className="text-slate-500 text-sm font-normal">/ {fmt$(data.salaryCap)}</span>
          </div>
          <div className="mt-2 h-1.5 bg-slate-800 rounded-full overflow-hidden">
            <div className={`h-full rounded-full ${capPct > 95 ? 'bg-red-500' : 'bg-blue-500'}`}
              style={{ width: `${Math.min(capPct, 100)}%` }} />
          </div>
        </div>
        <div className="bg-slate-900 border border-slate-800 rounded-xl p-4 flex items-center justify-center">
          <div className="text-center">
            <div className="text-2xl font-bold text-slate-100">{data.slots.length}</div>
            <div className="text-xs text-slate-500 mt-1">Rostered Players</div>
          </div>
        </div>
      </div>

      <div className="mb-6">
        <TeamTransactionHistory targetTeamId={targetTeamId} />
      </div>

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
                  </tr>
                </thead>
                <tbody>
                  {byPos[pos].map(slot => (
                    <tr key={slot.id} className="border-b border-slate-800/50 hover:bg-slate-800/30">
                      <td className="px-4 py-3"><span className="text-xs font-bold text-slate-400">{slot.player.position}</span></td>
                      <td className="px-4 py-3 font-medium text-slate-100">{slot.player.name}</td>
                      <td className="px-4 py-3 text-slate-400 text-xs hidden sm:table-cell">{slot.player.nhlTeam}</td>
                      <td className="px-4 py-3"><StatCell player={slot.player} /></td>
                      <td className="px-4 py-3 text-right font-mono text-slate-300 text-xs hidden md:table-cell">{fmt$(slot.player.salary)}</td>
                      <td className="px-4 py-3 text-center">
                        <span className={`text-xs px-2 py-0.5 rounded-full ${slotBadge[slot.slotType] ?? ''}`}>
                          {slot.slotType}
                        </span>
                      </td>
                    </tr>
                  ))}
                  {byPos[pos].length === 0 && (
                    <tr><td colSpan={6} className="px-4 py-6 text-center text-slate-600 text-xs">No {label.toLowerCase()}.</td></tr>
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
