import { createContext, useContext, useState, type ReactNode } from 'react'

interface AuthState {
  token: string | null
  teamId: number | null
  leagueId: string
  draftId: string
}

interface AuthContextValue extends AuthState {
  signIn: (token: string, teamId: number, draftId: string, leagueId?: string) => void
  signOut: () => void
}

const AuthContext = createContext<AuthContextValue | null>(null)

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error('useAuth must be used within AuthProvider')
  return ctx
}

function loadFromStorage(): AuthState {
  return {
    token: localStorage.getItem('draft_token'),
    teamId: localStorage.getItem('team_id') ? Number(localStorage.getItem('team_id')) : null,
    leagueId: localStorage.getItem('league_id') ?? '1',
    draftId: localStorage.getItem('draft_id') ?? '1',
  }
}

export function AuthProvider({ children }: { children: ReactNode }) {
  const [state, setState] = useState<AuthState>(loadFromStorage)

  function signIn(token: string, teamId: number, draftId: string, leagueId = '1') {
    localStorage.setItem('draft_token', token)
    localStorage.setItem('team_id', String(teamId))
    localStorage.setItem('draft_id', draftId)
    localStorage.setItem('league_id', leagueId)
    setState({ token, teamId, draftId, leagueId })
  }

  function signOut() {
    localStorage.removeItem('draft_token')
    localStorage.removeItem('team_id')
    localStorage.removeItem('draft_id')
    setState({ token: null, teamId: null, draftId: '1', leagueId: '1' })
  }

  return (
    <AuthContext.Provider value={{ ...state, signIn, signOut }}>
      {children}
    </AuthContext.Provider>
  )
}

export const API_BASE =
  (import.meta.env.VITE_API_BASE as string | undefined) ?? 'http://localhost:8080'

/** Builds standard auth headers from a token. */
export function authHeaders(token: string | null): Record<string, string> {
  return token ? { Authorization: `Bearer ${token}` } : {}
}
