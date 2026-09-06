import { NavLink } from 'react-router-dom'
import { useAuth } from '../context/AuthContext'

const links = [
  { to: '/draft', label: 'Draft' },
  { to: '/standings', label: 'Standings' },
  { to: '/schedule', label: 'Schedule' },
  { to: '/team', label: 'My Team' },
  { to: '/transactions', label: 'Transactions' },
  { to: '/trades', label: 'Trades' },
  { to: '/injuries', label: 'Injuries' },
  { to: '/admin', label: 'Admin' },
]

export function NavBar() {
  const { signOut } = useAuth()

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

      {links.map(({ to, label }) => (
        <NavLink
          key={to}
          to={to}
          className={({ isActive }) =>
            [
              'px-3 py-1.5 rounded-md text-xs font-semibold transition-colors shrink-0',
              isActive
                ? 'bg-slate-700 text-slate-100'
                : 'text-slate-500 hover:text-slate-300 hover:bg-slate-800',
            ].join(' ')
          }
        >
          {label}
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
