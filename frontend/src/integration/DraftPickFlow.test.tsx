/**
 * Integration tests for the live draft pick flow.
 *
 * These tests render the real DraftProvider (no MockDraftProvider shortcut) with
 * mocked network layers (fetch + WebSocket), exercising the full data path from
 * HTTP hydration through WebSocket events to component rendering.
 *
 * Key regression covered:
 *   After a pick_made WebSocket event the server broadcasts teams with
 *   isMe: false for every team (it has no idea which client is "me").
 *   DraftContext must re-apply the correct isMe value from its prior state,
 *   otherwise MyTeamView and MyRoster return null and go blank.
 */
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, waitFor, act } from '@testing-library/react'
import { DraftProvider } from '../context/DraftContext'
import { MyTeamView } from '../components/draft/MyTeamView'
import { MyRoster } from '../components/draft/MyRoster'
import {
  TEAMS,
  AVAILABLE_PLAYERS,
  DRAFT_STATE,
  CAP_LIMIT,
  SLOT_TARGETS,
} from '../data/mockDraft'

// ── WebSocket mock ─────────────────────────────────────────────────────────────

class MockWebSocket {
  static instances: MockWebSocket[] = []

  onopen:    ((e: Event) => void)        | null = null
  onmessage: ((e: MessageEvent) => void) | null = null
  onerror:   ((e: Event) => void)        | null = null
  onclose:   ((e: CloseEvent) => void)   | null = null
  readyState = 0 // CONNECTING

  url: string
  constructor(url: string) {
    this.url = url
    MockWebSocket.instances.push(this)
  }

  close() {
    this.readyState = 3 // CLOSED
    this.onclose?.(new CloseEvent('close'))
  }

  send(_: string) {}

  triggerOpen() {
    this.readyState = 1 // OPEN
    this.onopen?.(new Event('open'))
  }

  triggerMessage(data: unknown) {
    this.onmessage?.(new MessageEvent('message', { data: JSON.stringify(data) }))
  }

  /** The last instance created — after React StrictMode's double-invoke this is
   *  the real (non-abandoned) WebSocket that DraftContext actually uses. */
  static get active(): MockWebSocket | undefined {
    return MockWebSocket.instances.at(-1)
  }
}

// ── API response builders ──────────────────────────────────────────────────────

// Hat Trick Heroes is index 2 in TEAMS → backend assigns id 3
const MY_TEAM_ID = 3

function apiPlayer(p: typeof AVAILABLE_PLAYERS[0]) {
  return {
    id: parseInt(p.id),
    name: p.name,
    nhlTeam: p.team,
    nhlTeamCode: p.team,
    position: p.position,
    salary: p.salary,
    age: p.age,
    stats: p.stats,
  }
}

function apiTeam(t: typeof TEAMS[0], id: number, forcedIsMe?: boolean) {
  return {
    id,
    name: t.name,
    manager: t.manager,
    isMe: forcedIsMe ?? t.isMe,
    capUsed: t.capUsed,
    picks: t.picks.map(p => (p ? apiPlayer(p) : null)),
  }
}

function apiDraftFull() {
  return {
    draftState: {
      id: 1,
      status: 'in_progress',
      totalRounds: DRAFT_STATE.totalRounds,
      totalTeams: DRAFT_STATE.totalTeams,
      currentRound: DRAFT_STATE.currentRound,
      currentPick: DRAFT_STATE.currentPick,
      secondsPerPick: DRAFT_STATE.secondsRemaining,
    },
    teams: TEAMS.map((t, i) => apiTeam(t, i + 1)),
    players: AVAILABLE_PLAYERS.map(apiPlayer),
    config: { capLimit: CAP_LIMIT, slotTargets: SLOT_TARGETS },
    myTeamId: MY_TEAM_ID,
  }
}

/** Simulates a pick_made broadcast — backend sets isMe: false on all teams
 *  because it doesn't know which connected client is "me". */
function pickMadeEvent(nextRound: number, nextPick: number) {
  return {
    type: 'pick_made',
    payload: {
      draftState: {
        id: 1,
        status: 'in_progress',
        totalRounds: DRAFT_STATE.totalRounds,
        totalTeams: DRAFT_STATE.totalTeams,
        currentRound: nextRound,
        currentPick: nextPick,
        secondsPerPick: 90,
      },
      teams: TEAMS.map((t, i) => apiTeam(t, i + 1, false /* isMe: false for all */)),
    },
  }
}

// ── Helpers ────────────────────────────────────────────────────────────────────

function setup() {
  vi.stubGlobal(
    'fetch',
    vi.fn().mockImplementation((url: string) => {
      if (String(url).includes('/pick')) {
        return Promise.resolve({ ok: true, json: () => Promise.resolve({ success: true }) })
      }
      if (String(url).includes('/api/draft/')) {
        return Promise.resolve({ ok: true, json: () => Promise.resolve(apiDraftFull()) })
      }
      return Promise.resolve({ ok: false, status: 404 })
    })
  )
  vi.stubGlobal('WebSocket', MockWebSocket)
}

function renderDraftViews() {
  return render(
    <DraftProvider draftId="1" token="test-token">
      <MyTeamView />
      <MyRoster />
    </DraftProvider>
  )
}

async function waitForLoad() {
  // MyRoster displays the team name once fetch resolves
  await waitFor(() => {
    expect(screen.getByText('Hat Trick Heroes')).toBeInTheDocument()
  })
}

// ── Tests ──────────────────────────────────────────────────────────────────────

describe('Draft pick flow (integration)', () => {
  beforeEach(() => {
    MockWebSocket.instances = []
    setup()
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('shows my team after the initial HTTP load', async () => {
    renderDraftViews()
    await waitForLoad()

    expect(screen.getByText('Hat Trick Heroes')).toBeInTheDocument()
    // Cap section — both MyTeamView and MyRoster show these values
    expect(screen.getAllByText('$77M').length).toBeGreaterThan(0)
    expect(screen.getAllByText('$5.5M').length).toBeGreaterThan(0)
  })

  it('my team remains visible after a pick_made WebSocket event', async () => {
    renderDraftViews()
    await waitForLoad()

    // Trigger a pick_made event — server sends isMe: false for every team.
    // Before the fix, DraftContext would overwrite teams with that broadcast
    // verbatim, causing MyTeamView and MyRoster to return null (blank page).
    act(() => {
      MockWebSocket.active?.triggerOpen()
      MockWebSocket.active?.triggerMessage(pickMadeEvent(3, 4))
    })

    expect(screen.getByText('Hat Trick Heroes')).toBeInTheDocument()
  })

  it('cap is still displayed correctly after a pick_made event', async () => {
    renderDraftViews()
    await waitForLoad()

    act(() => {
      MockWebSocket.active?.triggerOpen()
      MockWebSocket.active?.triggerMessage(pickMadeEvent(3, 4))
    })

    // Hat Trick Heroes' cap values should be unchanged (no one picked from our team)
    // Both MyTeamView and MyRoster render these values
    expect(screen.getAllByText('$77M').length).toBeGreaterThan(0)
    expect(screen.getAllByText('$5.5M').length).toBeGreaterThan(0)
  })

  it('roster slots are still rendered after a pick_made event', async () => {
    renderDraftViews()
    await waitForLoad()

    act(() => {
      MockWebSocket.active?.triggerOpen()
      MockWebSocket.active?.triggerMessage(pickMadeEvent(3, 4))
    })

    // Both MyTeamView and MyRoster render position group headers when a "me" team exists
    expect(screen.getAllByText('Forwards').length).toBeGreaterThan(0)
    expect(screen.getAllByText('Defence').length).toBeGreaterThan(0)
    expect(screen.getAllByText('Goalies').length).toBeGreaterThan(0)
  })

  it('consecutive pick_made events keep the team visible', async () => {
    renderDraftViews()
    await waitForLoad()

    act(() => {
      MockWebSocket.active?.triggerOpen()
      MockWebSocket.active?.triggerMessage(pickMadeEvent(3, 4))
    })
    act(() => {
      MockWebSocket.active?.triggerMessage(pickMadeEvent(3, 5))
    })
    act(() => {
      MockWebSocket.active?.triggerMessage(pickMadeEvent(3, 6))
    })

    expect(screen.getByText('Hat Trick Heroes')).toBeInTheDocument()
  })
})
