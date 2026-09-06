import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useRef,
  useState,
  type ReactNode,
} from 'react'
import type { DraftState, DraftTeam, Player } from '../data/mockDraft'

// ── API shapes (what the backend sends) ──────────────────────────────────────

interface ApiPlayer {
  id: number
  name: string
  nhlTeam: string
  nhlTeamCode: string
  position: 'F' | 'D' | 'G'
  salary: number
  age: number
  stats: { goals: number; assists: number; wins?: number; gaa?: number }
}

interface ApiTeam {
  id: number
  name: string
  manager: string
  isMe: boolean
  capUsed: number
  picks: (ApiPlayer | null)[]
}

interface ApiDraftState {
  id: number
  status: string
  totalRounds: number
  totalTeams: number
  currentRound: number
  currentPick: number
  secondsPerPick: number
}

interface ApiDraftFull {
  draftState: ApiDraftState
  teams: ApiTeam[]
  players: ApiPlayer[]
  config: { capLimit: number; slotTargets: Record<string, number> }
  myTeamId: number
}

// ── Adapters (API → app types) ───────────────────────────────────────────────

function adaptPlayer(p: ApiPlayer): Player {
  return {
    id: String(p.id),
    name: p.name,
    position: p.position,
    team: p.nhlTeamCode,
    salary: p.salary,
    age: p.age,
    stats: p.stats,
  }
}

function adaptTeam(t: ApiTeam): DraftTeam {
  return {
    id: String(t.id),
    name: t.name,
    manager: t.manager,
    isMe: t.isMe,
    capUsed: t.capUsed,
    picks: t.picks.map(p => (p ? adaptPlayer(p) : null)),
  }
}

function adaptDraftState(ds: ApiDraftState): DraftState {
  return {
    totalRounds: ds.totalRounds,
    totalTeams: ds.totalTeams,
    currentRound: ds.currentRound,
    currentPick: ds.currentPick,
    status: 'in_progress',
    secondsRemaining: ds.secondsPerPick,
  }
}

// ── Context ──────────────────────────────────────────────────────────────────

export interface DraftContextValue {
  draftState: DraftState | null
  teams: DraftTeam[]
  players: Player[]
  capLimit: number
  slotTargets: Record<string, number>
  myTeamID: number | null
  isLoading: boolean
  error: string | null
  /** Returns a server-side error string, or null on success. */
  makePick: (playerId: string) => Promise<string | null>
}

export const DraftContext = createContext<DraftContextValue | null>(null)

export function useDraft(): DraftContextValue {
  const ctx = useContext(DraftContext)
  if (!ctx) throw new Error('useDraft must be used within DraftProvider')
  return ctx
}

// ── Provider ─────────────────────────────────────────────────────────────────

const API_BASE = (import.meta.env.VITE_API_BASE as string | undefined) ?? 'http://localhost:8080'

export function DraftProvider({
  draftId,
  token,
  onDraftComplete,
  children,
}: {
  draftId: string
  token: string
  /** Called once when the backend broadcasts draft_complete. */
  onDraftComplete?: () => void
  children: ReactNode
}) {
  const [draftState, setDraftState] = useState<DraftState | null>(null)
  const [teams, setTeams] = useState<DraftTeam[]>([])
  const [players, setPlayers] = useState<Player[]>([])
  const [capLimit, setCapLimit] = useState(82_500_000)
  const [slotTargets, setSlotTargets] = useState<Record<string, number>>({ F: 9, D: 4, G: 2 })
  const [myTeamID, setMyTeamID] = useState<number | null>(null)
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const wsRef = useRef<WebSocket | null>(null)

  // Initial hydration
  useEffect(() => {
    let cancelled = false
    setIsLoading(true)
    setError(null)

    fetch(`${API_BASE}/api/draft/${draftId}`, {
      headers: { Authorization: `Bearer ${token}` },
    })
      .then(r => {
        if (!r.ok) throw new Error(`HTTP ${r.status}`)
        return r.json() as Promise<ApiDraftFull>
      })
      .then(data => {
        if (cancelled) return
        setDraftState(adaptDraftState(data.draftState))
        setTeams(data.teams.map(adaptTeam))
        setPlayers(data.players.map(adaptPlayer))
        setCapLimit(data.config.capLimit)
        setSlotTargets(data.config.slotTargets)
        setMyTeamID(data.myTeamId)
      })
      .catch(e => {
        if (!cancelled) setError(e instanceof Error ? e.message : 'Failed to load draft')
      })
      .finally(() => {
        if (!cancelled) setIsLoading(false)
      })

    return () => { cancelled = true }
  }, [draftId, token])

  // Real-time WebSocket updates
  useEffect(() => {
    let abandoned = false
    const wsBase = API_BASE.replace(/^http/, 'ws')
    const ws = new WebSocket(
      `${wsBase}/ws/draft/${draftId}?token=${encodeURIComponent(token)}`
    )

    ws.onopen = () => {
      if (abandoned) ws.close()
      else wsRef.current = ws
    }

    ws.onmessage = evt => {
      if (abandoned) return
      try {
        const msg = JSON.parse(evt.data as string) as {
          type: string
          payload: { teams: ApiTeam[]; draftState: ApiDraftState }
        }
        if (msg.type === 'pick_made') {
          // Broadcast teams have isMe: false for all — re-apply from prior state
          setTeams(prev => msg.payload.teams.map(t => ({
            ...adaptTeam(t),
            isMe: prev.find(p => p.id === String(t.id))?.isMe ?? false,
          })))
          setDraftState(adaptDraftState(msg.payload.draftState))
        } else if (msg.type === 'draft_complete') {
          // Give the backend a moment to finish FinaliseDraft before navigating
          setTimeout(() => onDraftComplete?.(), 1500)
        }
      } catch { /* ignore malformed frames */ }
    }

    ws.onerror = () => {
      if (!abandoned) wsRef.current?.close()
    }

    return () => {
      abandoned = true
      ws.close()
    }
  }, [draftId, token])

  const makePick = useCallback(
    async (playerId: string): Promise<string | null> => {
      try {
        const res = await fetch(`${API_BASE}/api/draft/${draftId}/pick`, {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
            Authorization: `Bearer ${token}`,
          },
          body: JSON.stringify({ playerId: Number(playerId) }),
        })
        const data = (await res.json()) as { success: boolean; error?: string }
        return data.success ? null : (data.error ?? 'Unknown error')
      } catch {
        return 'Network error'
      }
    },
    [draftId, token],
  )

  return (
    <DraftContext.Provider
      value={{ draftState, teams, players, capLimit, slotTargets, myTeamID, isLoading, error, makePick }}
    >
      {children}
    </DraftContext.Provider>
  )
}
