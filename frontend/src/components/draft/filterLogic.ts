import type { Player, Position } from '../../data/mockDraft'

export interface FilterParams {
  query: string
  posFilter: Position | 'ALL'
  nhlTeamFilter: string
  hideTaken: boolean
  showDraftable: boolean
  draftedBy: Map<string, string>
  capRemaining: number
  fullPositions: Set<Position>
}

export function filterPlayers(players: Player[], params: FilterParams): Player[] {
  return players.filter(p => {
    if (params.showDraftable) {
      if (params.draftedBy.has(p.id)) return false
      if (p.salary > params.capRemaining) return false
      if (params.fullPositions.has(p.position)) return false
    } else if (params.hideTaken && params.draftedBy.has(p.id)) return false

    if (params.posFilter !== 'ALL' && p.position !== params.posFilter) return false
    if (params.nhlTeamFilter !== 'ALL' && p.team !== params.nhlTeamFilter) return false
    if (params.query && !p.name.toLowerCase().includes(params.query.toLowerCase())) return false

    return true
  })
}
