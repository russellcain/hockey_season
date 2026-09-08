import { NavLink } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { useAuth, API_BASE, authHeaders } from '../context/AuthContext'

interface InjuryInfo {
  teamId: number
}

interface League {
  status: string
}

export function NavBar() {
  const { token, leagueId, teamId, signOut } = useAuth()

  const { data: injuries = [] } = useQuery<InjuryInfo[]>({
    queryKey: ['injuries', leagueId],
    queryFn: () =>
      fetch(`${API_BASE}/api/leagues/${leagueId}/injuries`, {
        headers: authHeaders(token),
      }).then(r => r.json()),
    enabled: !!token,
    refetchInterval: 5 * 60 * 1000,
  })

  const { data: league } = useQuery<League>({
    queryKey: ['league', leagueId],
    queryFn: () =>
      fetch(`${API_BASE}/api/leagues/${leagueId}`, {
        headers: authHeaders(token),
      }).then(r => r.json()),
    enabled: !!token,
    staleTime: 60_000,
  })

  const myInjuryCount = injuries.filter(i => i.teamId === teamId).length
  const draftComplete = league?.status === 'in_season' || league?.status === 'complete'

  // Core season links — shown prominently while the season is active.
  const seasonLinks = [
    { to: '/standings', label: 'Standings' },
    { to: '/schedule', label: 'Schedule' },
    { to: '/team', label: 'My Team' },
    { to: '/transactions', label: 'Transactions' },
    { to: '/trades', label: 'Trades' },
    { to: '/injuries', label: 'Injuries', badge: myInjuryCount },
    { to: '/admin', label: 'Admin' },
  ]

  // Draft link — primary during drafting, deprioritised once complete.
  const draftLink = { to: '/draft', label: 'Draft', muted: draftComplete }

  // Put draft first when active, last when complete.
  const links = draftComplete
    ? [...seasonLinks, draftLink]
    : [{ to: '/draft', label: 'Draft', muted: false }, ...seasonLinks]

  return (
    <nav className="flex items-center gap-1 px-4 py-2 bg-slate-900 border-b border-slate-800 shrink-0 overflow-x-auto">
      <div className="flex items-center gap-2 mr-4 shrink-0">
        <div className="w-6 h-6 rounded bg-blue-600 flex items-center justify-center">
          <svg className="w-4 h-4 text-white" fill="currentColor" viewBox="0 0 20 20">
            <path d="M10 2a8 8 0 100 16A8 8 0 0010 2zm0 14a6 6 0 110-12 6 6 0 010 12z" />
            <path d="M10 5a1 1 0 011 1v4a1 1 0 01-.293.707l-2 2a1 1 0 01-1.414-1.414L9 9.586V6a1 1 0 011-1z" />
          </svg>
        </div>
        <span className="font-bold text-white tracking-tight text-sm">Draftr</span>
      </div>

      {links.map(({ to, label, badge, muted }) => (
        <NavLink
          key={to}
          to={to}
          className={({ isActive }) =>
            [
              'relative px-3 py-1.5 rounded-md text-xs font-semibold transition-colors shrink-0',
              isActive
                ? 'bg-slate-700 text-slate-100'
                : muted
                ? 'text-slate-600 hover:text-slate-400 hover:bg-slate-800'
                : 'text-slate-500 hover:text-slate-300 hover:bg-slate-800',
            ].join(' ')
          }
        >
          {label}
          {badge != null && badge > 0 && (
            <span className="absolute -top-1 -right-1 flex h-4 w-4 items-center justify-center rounded-full bg-red-500 text-white text-[10px] font-bold leading-none">
              {badge > 9 ? '9+' : badge}
            </span>
          )}
        </NavLink>
      ))}

      <button
        onClick={signOut}
        className="ml-auto text-xs text-slate-600 hover:text-slate-400 transition-colors shrink-0"
      >
        Sign out
      </button>
    </nav>
  )
}
