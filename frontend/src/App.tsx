import { BrowserRouter, Routes, Route, Navigate, Outlet, useNavigate } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { AuthProvider, useAuth } from './context/AuthContext'
import { DraftProvider } from './context/DraftContext'
import { NavBar } from './components/NavBar'
import { DevBar } from './components/DevBar'
import { DraftRoom } from './pages/DraftRoom'
import { JoinPage } from './pages/JoinPage'
import { StandingsPage } from './pages/StandingsPage'
import { SchedulePage } from './pages/SchedulePage'
import { TeamPage } from './pages/TeamPage'
import { TransactionsPage } from './pages/TransactionsPage'
import { InjuriesPage } from './pages/InjuriesPage'
import { TradesPage } from './pages/TradesPage'
import { AdminPage } from './pages/AdminPage'
import { TeamDetailPage } from './pages/TeamDetailPage'

const queryClient = new QueryClient({
  defaultOptions: { queries: { retry: 1, staleTime: 30_000 } },
})

const DEV_MODE = import.meta.env.VITE_DEV_MODE === 'true'

/** Wraps authenticated pages: redirects to /join if not signed in, otherwise shows NavBar. */
function RequireAuth() {
  const { token } = useAuth()
  if (!token) return <Navigate to="/join" replace />
  return (
    <div className="flex flex-col h-screen bg-slate-950 overflow-hidden">
      <NavBar />
      <div className={`flex-1 overflow-auto ${DEV_MODE ? 'pb-14' : ''}`}>
        <Outlet />
      </div>
      {DEV_MODE && <DevBar />}
    </div>
  )
}

/** Draft page needs the DraftProvider wrapper. */
function DraftPage() {
  const { token, draftId, signOut } = useAuth()
  const navigate = useNavigate()
  // token is guaranteed non-null here (RequireAuth guards this route)
  return (
    <DraftProvider
      draftId={draftId}
      token={token!}
      onDraftComplete={() => navigate('/standings', { replace: true })}
    >
      <DraftRoom onSignOut={signOut} />
    </DraftProvider>
  )
}

function AppRoutes() {
  const { token } = useAuth()

  return (
    <Routes>
      {/* Public: join page, redirects to /draft when already authenticated */}
      <Route
        path="/join"
        element={token ? <Navigate to="/draft" replace /> : <JoinPage />}
      />

      {/* Authenticated layout — NavBar visible on all children */}
      <Route element={<RequireAuth />}>
        <Route path="/draft" element={<DraftPage />} />
        <Route path="/standings" element={<StandingsPage />} />
        <Route path="/schedule" element={<SchedulePage />} />
        <Route path="/team" element={<TeamPage />} />
        <Route path="/transactions" element={<TransactionsPage />} />
        <Route path="/injuries" element={<InjuriesPage />} />
        <Route path="/trades" element={<TradesPage />} />
        <Route path="/admin" element={<AdminPage />} />
        <Route path="/teams/:teamId" element={<TeamDetailPage />} />
      </Route>

      {/* Fallback */}
      <Route path="*" element={<Navigate to="/draft" replace />} />
    </Routes>
  )
}

export default function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <AuthProvider>
        <BrowserRouter>
          <AppRoutes />
        </BrowserRouter>
      </AuthProvider>
    </QueryClientProvider>
  )
}
