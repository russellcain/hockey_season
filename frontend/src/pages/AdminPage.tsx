import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useAuth, API_BASE, authHeaders } from '../context/AuthContext'

interface League {
  id: number
  name: string
  status: string
  salaryCap: number
}

interface TradeDetail {
  id: number
  status: string
  submittedByTeam: { id: number; name: string }
  legs: Array<{ fromTeam: { name: string }; toTeam: { name: string }; player: { name: string; position: string } }>
  notes?: string
}

const SEASON_STATUSES = ['setup', 'draft_ready', 'drafting', 'in_season', 'complete'] as const
type SeasonStatus = typeof SEASON_STATUSES[number]

const STATUS_LABELS: Record<SeasonStatus, string> = {
  setup: 'Setup',
  draft_ready: 'Draft Ready',
  drafting: 'Drafting',
  in_season: 'In Season',
  complete: 'Complete',
}

function nextStatus(current: string): SeasonStatus | null {
  const idx = SEASON_STATUSES.indexOf(current as SeasonStatus)
  if (idx < 0 || idx >= SEASON_STATUSES.length - 1) return null
  return SEASON_STATUSES[idx + 1]
}

export function AdminPage() {
  const { token, leagueId } = useAuth()
  const qc = useQueryClient()

  const { data: league } = useQuery<League>({
    queryKey: ['league', leagueId],
    queryFn: () =>
      fetch(`${API_BASE}/api/leagues/${leagueId}`, { headers: authHeaders(token) })
        .then(r => r.json()),
    enabled: !!token,
  })

  const { data: trades = [] } = useQuery<TradeDetail[]>({
    queryKey: ['trades', leagueId],
    queryFn: () =>
      fetch(`${API_BASE}/api/leagues/${leagueId}/trades`, { headers: authHeaders(token) })
        .then(r => r.json()),
    enabled: !!token,
  })

  const advanceStatus = useMutation({
    mutationFn: (status: string) =>
      fetch(`${API_BASE}/api/leagues/${leagueId}/status`, {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json', ...authHeaders(token) },
        body: JSON.stringify({ status }),
      }).then(r => r.json()),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['league'] }),
  })

  const generateSchedule = useMutation({
    mutationFn: () =>
      fetch(`${API_BASE}/api/leagues/${leagueId}/schedule/generate`, {
        method: 'POST',
        headers: authHeaders(token),
      }).then(r => r.json()),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['schedule'] }),
  })

  const reviewTrade = useMutation({
    mutationFn: ({ tradeId, decision, notes }: { tradeId: number; decision: string; notes?: string }) =>
      fetch(`${API_BASE}/api/trades/${tradeId}/review`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', ...authHeaders(token) },
        body: JSON.stringify({ decision, notes }),
      }).then(r => r.json()),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['trades'] }),
  })

  const next = league ? nextStatus(league.status) : null
  const pendingTrades = trades.filter(t => t.status === 'pending')

  return (
    <div className="p-6 max-w-4xl mx-auto space-y-8">
      <h1 className="text-xl font-bold text-slate-100">Admin</h1>

      {/* Season status */}
      <section>
        <h2 className="text-sm font-semibold text-slate-400 uppercase tracking-wider mb-3">Season Status</h2>
        <div className="bg-slate-900 border border-slate-800 rounded-xl p-5">
          {league ? (
            <div className="flex items-center justify-between">
              <div>
                <div className="text-lg font-semibold text-slate-100">{league.name}</div>
                <div className="text-sm text-slate-400 mt-1">
                  Status: <span className="text-blue-400 font-medium">{STATUS_LABELS[league.status as SeasonStatus] ?? league.status}</span>
                </div>
              </div>
              {next && (
                <button
                  disabled={advanceStatus.isPending}
                  onClick={() => advanceStatus.mutate(next)}
                  className="px-4 py-2 text-sm font-semibold bg-blue-600 hover:bg-blue-500 disabled:opacity-40 text-white rounded-lg transition-colors"
                >
                  Advance to {STATUS_LABELS[next]}
                </button>
              )}
            </div>
          ) : (
            <div className="text-slate-500 text-sm">Loading league…</div>
          )}
        </div>
      </section>

      {/* Schedule */}
      <section>
        <h2 className="text-sm font-semibold text-slate-400 uppercase tracking-wider mb-3">Schedule</h2>
        <div className="bg-slate-900 border border-slate-800 rounded-xl p-5">
          <div className="flex items-center justify-between">
            <p className="text-sm text-slate-400">
              Generate the H2H schedule after the draft completes.
            </p>
            <button
              disabled={generateSchedule.isPending}
              onClick={() => generateSchedule.mutate()}
              className="px-4 py-2 text-sm font-semibold bg-slate-700 hover:bg-slate-600 disabled:opacity-40 text-slate-100 rounded-lg transition-colors"
            >
              {generateSchedule.isPending ? 'Generating…' : 'Generate Schedule'}
            </button>
          </div>
          {generateSchedule.isSuccess && (
            <div className="mt-3 text-xs text-green-400">Schedule generated successfully.</div>
          )}
          {generateSchedule.isError && (
            <div className="mt-3 text-xs text-red-400">Failed to generate schedule.</div>
          )}
        </div>
      </section>

      {/* Pending trades */}
      <section>
        <h2 className="text-sm font-semibold text-slate-400 uppercase tracking-wider mb-3">
          Pending Trades ({pendingTrades.length})
        </h2>
        {pendingTrades.length === 0 ? (
          <p className="text-slate-600 text-sm">No pending trades.</p>
        ) : (
          <div className="space-y-3">
            {pendingTrades.map(trade => (
              <div key={trade.id} className="bg-slate-900 border border-slate-800 rounded-xl p-4">
                <div className="font-semibold text-slate-100 mb-2 text-sm">
                  Trade #{trade.id} from {trade.submittedByTeam.name}
                </div>
                <div className="space-y-1 mb-4">
                  {(trade.legs ?? []).map((leg, i) => (
                    <div key={i} className="text-xs text-slate-400">
                      {leg.player.name} ({leg.player.position}) · {leg.fromTeam.name} → {leg.toTeam.name}
                    </div>
                  ))}
                </div>
                <div className="flex gap-2">
                  <button
                    onClick={() => reviewTrade.mutate({ tradeId: trade.id, decision: 'approved' })}
                    disabled={reviewTrade.isPending}
                    className="px-3 py-1.5 text-xs font-semibold bg-green-600/20 hover:bg-green-600/30 text-green-400 rounded-md transition-colors disabled:opacity-40"
                  >
                    Approve
                  </button>
                  <button
                    onClick={() => reviewTrade.mutate({ tradeId: trade.id, decision: 'rejected', notes: 'Rejected by commissioner' })}
                    disabled={reviewTrade.isPending}
                    className="px-3 py-1.5 text-xs font-semibold bg-red-600/20 hover:bg-red-600/30 text-red-400 rounded-md transition-colors disabled:opacity-40"
                  >
                    Reject
                  </button>
                </div>
              </div>
            ))}
          </div>
        )}
      </section>
    </div>
  )
}
