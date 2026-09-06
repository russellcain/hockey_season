import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useAuth, API_BASE } from '../context/AuthContext'

export function JoinPage() {
  const { signIn } = useAuth()
  const navigate = useNavigate()
  const [code, setCode] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    if (!code.trim()) return
    setLoading(true)
    setError(null)
    try {
      const res = await fetch(`${API_BASE}/api/auth/join`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ code: code.trim() }),
      })
      if (!res.ok) {
        setError('Invalid team code. Check with your commissioner.')
        return
      }
      const data = (await res.json()) as { token: string; team: { id: number }; draftId: number }
      signIn(data.token, data.team?.id ?? 0, String(data.draftId))
      navigate('/draft')
    } catch {
      setError('Could not reach the draft server. Is it running?')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="min-h-screen bg-slate-950 flex items-center justify-center p-4">
      <div className="w-full max-w-sm">
        <div className="flex items-center gap-2 justify-center mb-8">
          <div className="w-8 h-8 rounded-lg bg-blue-600 flex items-center justify-center">
            <svg className="w-5 h-5 text-white" fill="currentColor" viewBox="0 0 20 20">
              <path d="M10 2a8 8 0 100 16A8 8 0 0010 2zm0 14a6 6 0 110-12 6 6 0 010 12z" />
              <path d="M10 5a1 1 0 011 1v4a1 1 0 01-.293.707l-2 2a1 1 0 01-1.414-1.414L9 9.586V6a1 1 0 011-1z" />
            </svg>
          </div>
          <span className="text-2xl font-bold text-white tracking-tight">Draftr</span>
        </div>

        <div className="bg-slate-900 border border-slate-800 rounded-2xl p-6">
          <h1 className="text-lg font-semibold text-slate-100 mb-1">Join the draft</h1>
          <p className="text-sm text-slate-500 mb-6">Enter the team code your commissioner sent you.</p>

          <form onSubmit={handleSubmit} className="space-y-4">
            <div>
              <label className="block text-xs font-semibold text-slate-400 uppercase tracking-wider mb-1.5">
                Team Code
              </label>
              <input
                type="text"
                value={code}
                onChange={e => setCode(e.target.value)}
                placeholder="e.g. hat-trick-heroes-code"
                autoFocus
                className="filter-control w-full px-3 py-2.5 text-sm placeholder-slate-600"
              />
            </div>

            {error && (
              <p className="text-xs text-red-400 bg-red-500/10 border border-red-500/20 rounded-lg px-3 py-2">
                {error}
              </p>
            )}

            <button
              type="submit"
              disabled={loading || !code.trim()}
              className="w-full bg-blue-600 hover:bg-blue-500 disabled:opacity-40 disabled:cursor-not-allowed text-white font-semibold text-sm py-2.5 rounded-lg transition-colors"
            >
              {loading ? 'Joining…' : 'Join Draft'}
            </button>
          </form>
        </div>

        <p className="text-center text-xs text-slate-700 mt-4">2026–27 Season Draft</p>
      </div>
    </div>
  )
}
