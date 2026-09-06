import { useState, type ReactNode } from 'react'
import { DraftContext, type DraftContextValue } from '../../../context/DraftContext'
import {
  AVAILABLE_PLAYERS,
  TEAMS,
  DRAFT_STATE,
  CAP_LIMIT,
  SLOT_TARGETS,
  type Player,
  type DraftTeam,
} from '../../../data/mockDraft'

export function MockDraftProvider({ children }: { children: ReactNode }) {
  const [teams, setTeams] = useState<DraftTeam[]>(TEAMS)

  const makePick = async (playerId: string): Promise<string | null> => {
    const player = AVAILABLE_PLAYERS.find(p => p.id === playerId)
    if (!player) return 'Player not found'

    setTeams(prev => prev.map(team => {
      if (!team.isMe) return team
      const firstNull = team.picks.indexOf(null)
      const newPicks: (Player | null)[] = [...team.picks]
      if (firstNull >= 0) newPicks[firstNull] = player
      return { ...team, picks: newPicks, capUsed: team.capUsed + player.salary }
    }))
    return null
  }

  const value: DraftContextValue = {
    draftState: DRAFT_STATE,
    teams,
    players: AVAILABLE_PLAYERS,
    capLimit: CAP_LIMIT,
    slotTargets: SLOT_TARGETS,
    myTeamID: 3,
    isLoading: false,
    error: null,
    makePick,
  }

  return <DraftContext.Provider value={value}>{children}</DraftContext.Provider>
}
