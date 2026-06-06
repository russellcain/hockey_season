import { describe, it, expect } from 'vitest'
import { filterPlayers, type FilterParams } from '../filterLogic'
import type { Player, Position } from '../../../data/mockDraft'

// Minimal player fixtures — no dependency on full mock data
const players: Player[] = [
  { id: '1', name: 'Alice Forward',  position: 'F', team: 'EDM', salary: 10_000_000, age: 25, stats: { goals: 30, assists: 40 } },
  { id: '2', name: 'Bob Defence',    position: 'D', team: 'TOR', salary:  8_000_000, age: 28, stats: { goals: 10, assists: 30 } },
  { id: '3', name: 'Carol Goalie',   position: 'G', team: 'EDM', salary:  6_000_000, age: 30, stats: { goals: 0,  assists: 0, wins: 35, gaa: 2.1 } },
  { id: '4', name: 'Dave Forward',   position: 'F', team: 'BOS', salary:  3_000_000, age: 22, stats: { goals: 20, assists: 25 } },
  { id: '5', name: 'Eve Defence',    position: 'D', team: 'EDM', salary:  5_000_000, age: 26, stats: { goals: 8,  assists: 22 } },
]

const base: FilterParams = {
  query: '',
  posFilter: 'ALL',
  nhlTeamFilter: 'ALL',
  hideTaken: false,
  showDraftable: false,
  draftedBy: new Map(),
  capRemaining: 20_000_000,
  fullPositions: new Set(),
}

describe('filterPlayers', () => {
  describe('position filter', () => {
    it('ALL returns every player', () => {
      expect(filterPlayers(players, base)).toHaveLength(5)
    })

    it('F returns only forwards', () => {
      const result = filterPlayers(players, { ...base, posFilter: 'F' })
      expect(result).toHaveLength(2)
      expect(result.every(p => p.position === 'F')).toBe(true)
    })

    it('D returns only defenders', () => {
      const result = filterPlayers(players, { ...base, posFilter: 'D' })
      expect(result).toHaveLength(2)
      expect(result.every(p => p.position === 'D')).toBe(true)
    })

    it('G returns only goalies', () => {
      const result = filterPlayers(players, { ...base, posFilter: 'G' })
      expect(result).toHaveLength(1)
      expect(result[0].name).toBe('Carol Goalie')
    })
  })

  describe('NHL team filter', () => {
    it('ALL includes players from all teams', () => {
      expect(filterPlayers(players, base)).toHaveLength(5)
    })

    it('EDM returns only Edmonton players', () => {
      const result = filterPlayers(players, { ...base, nhlTeamFilter: 'EDM' })
      expect(result).toHaveLength(3)
      expect(result.every(p => p.team === 'EDM')).toBe(true)
    })

    it('TOR returns only Toronto players', () => {
      const result = filterPlayers(players, { ...base, nhlTeamFilter: 'TOR' })
      expect(result).toHaveLength(1)
      expect(result[0].name).toBe('Bob Defence')
    })

    it('returns empty array when no players match team', () => {
      expect(filterPlayers(players, { ...base, nhlTeamFilter: 'VAN' })).toHaveLength(0)
    })
  })

  describe('hide taken', () => {
    it('when off, drafted player appears in results', () => {
      const draftedBy = new Map([['1', 'Some Team']])
      const result = filterPlayers(players, { ...base, hideTaken: false, draftedBy })
      expect(result.some(p => p.id === '1')).toBe(true)
    })

    it('when on, drafted player is excluded', () => {
      const draftedBy = new Map([['1', 'Some Team']])
      const result = filterPlayers(players, { ...base, hideTaken: true, draftedBy })
      expect(result.some(p => p.id === '1')).toBe(false)
      expect(result).toHaveLength(4)
    })

    it('when on, undrafted players are not affected', () => {
      const draftedBy = new Map([['1', 'Some Team']])
      const result = filterPlayers(players, { ...base, hideTaken: true, draftedBy })
      expect(result.some(p => p.id === '4')).toBe(true)
    })
  })

  describe('draftable only', () => {
    it('excludes drafted players', () => {
      const draftedBy = new Map([['4', 'Some Team']])
      const result = filterPlayers(players, { ...base, showDraftable: true, draftedBy })
      expect(result.some(p => p.id === '4')).toBe(false)
    })

    it('excludes players over the remaining cap', () => {
      const result = filterPlayers(players, { ...base, showDraftable: true, capRemaining: 5_000_000 })
      // Alice ($10M) and Bob ($8M) and Carol ($6M) are over $5M cap — only Dave ($3M) and Eve ($5M) pass
      expect(result.some(p => p.id === '1')).toBe(false) // Alice $10M
      expect(result.some(p => p.id === '2')).toBe(false) // Bob $8M
      expect(result.some(p => p.id === '3')).toBe(false) // Carol $6M — over $5M
    })

    it('includes a player whose salary equals cap remaining', () => {
      const result = filterPlayers(players, { ...base, showDraftable: true, capRemaining: 5_000_000 })
      expect(result.some(p => p.id === '5')).toBe(true) // Eve exactly $5M
    })

    it('excludes players in full positions', () => {
      const fullPositions = new Set<Position>(['F'])
      const result = filterPlayers(players, { ...base, showDraftable: true, fullPositions })
      expect(result.some(p => p.position === 'F')).toBe(false)
    })

    it('includes players that meet all three criteria', () => {
      const result = filterPlayers(players, { ...base, showDraftable: true, capRemaining: 20_000_000 })
      expect(result).toHaveLength(5)
    })

    it('supersedes hideTaken — draftable filter handles taken exclusion itself', () => {
      const draftedBy = new Map([['1', 'Some Team']])
      // hideTaken false, but showDraftable true → drafted player still excluded
      const result = filterPlayers(players, { ...base, showDraftable: true, hideTaken: false, draftedBy })
      expect(result.some(p => p.id === '1')).toBe(false)
    })
  })

  describe('search query', () => {
    it('filters by partial name match', () => {
      const result = filterPlayers(players, { ...base, query: 'alice' })
      expect(result).toHaveLength(1)
      expect(result[0].name).toBe('Alice Forward')
    })

    it('is case-insensitive', () => {
      expect(filterPlayers(players, { ...base, query: 'ALICE' })).toHaveLength(1)
      expect(filterPlayers(players, { ...base, query: 'Alice' })).toHaveLength(1)
    })

    it('returns empty array when no name matches', () => {
      expect(filterPlayers(players, { ...base, query: 'zzznomatch' })).toHaveLength(0)
    })

    it('matches on substring', () => {
      const result = filterPlayers(players, { ...base, query: 'ward' }) // "Forward"
      expect(result).toHaveLength(2)
    })
  })

  describe('combined filters', () => {
    it('position + team: forwards on EDM', () => {
      const result = filterPlayers(players, { ...base, posFilter: 'F', nhlTeamFilter: 'EDM' })
      expect(result).toHaveLength(1)
      expect(result[0].name).toBe('Alice Forward')
    })

    it('draftable + position: only draftable forwards', () => {
      const fullPositions = new Set<Position>(['D'])
      const result = filterPlayers(players, { ...base, showDraftable: true, posFilter: 'F', fullPositions })
      expect(result.every(p => p.position === 'F')).toBe(true)
    })

    it('search + position: search within a position', () => {
      const result = filterPlayers(players, { ...base, posFilter: 'D', query: 'bob' })
      expect(result).toHaveLength(1)
      expect(result[0].name).toBe('Bob Defence')
    })
  })
})
