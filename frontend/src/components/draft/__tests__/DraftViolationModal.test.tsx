import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { DraftViolationModal } from '../DraftViolationModal'
import type { Violation } from '../DraftViolationModal'
import type { Player } from '../../../data/mockDraft'

const forwardPlayer: Player = {
  id: '99', name: 'Test Player', position: 'F', team: 'EDM',
  salary: 12_500_000, age: 27, stats: { goals: 30, assists: 40 },
}

const goaliePlayer: Player = {
  id: '98', name: 'Test Goalie', position: 'G', team: 'TOR',
  salary: 5_000_000, age: 29, stats: { goals: 0, assists: 0, wins: 30, gaa: 2.40 },
}

const overCapViolation: Violation = {
  kind: 'over-cap',
  player: forwardPlayer,
  capRemaining: 5_500_000,
  shortfall: 7_000_000,
}

const positionFullViolation: Violation = {
  kind: 'position-full',
  player: goaliePlayer,
  slotsFilled: 2,
  slotsTarget: 2,
}

describe('DraftViolationModal — over cap', () => {
  it('shows the modal title', () => {
    render(<DraftViolationModal violation={overCapViolation} onDismiss={vi.fn()} />)
    expect(screen.getByText('Over Cap Limit')).toBeInTheDocument()
  })

  it('shows the player name', () => {
    render(<DraftViolationModal violation={overCapViolation} onDismiss={vi.fn()} />)
    expect(screen.getByText('Test Player')).toBeInTheDocument()
  })

  it('shows cap remaining, player cost, and shortfall', () => {
    render(<DraftViolationModal violation={overCapViolation} onDismiss={vi.fn()} />)
    expect(screen.getByText('Cap remaining')).toBeInTheDocument()
    expect(screen.getByText('Player cost')).toBeInTheDocument()
    expect(screen.getByText('Shortfall')).toBeInTheDocument()
    expect(screen.getByText('$5.5M')).toBeInTheDocument()  // cap remaining
    expect(screen.getByText('$12.5M')).toBeInTheDocument() // player cost
    expect(screen.getByText('$7M')).toBeInTheDocument()    // shortfall
  })

  it('calls onDismiss when Got it is clicked', async () => {
    const onDismiss = vi.fn()
    render(<DraftViolationModal violation={overCapViolation} onDismiss={onDismiss} />)
    await userEvent.click(screen.getByRole('button', { name: /got it/i }))
    expect(onDismiss).toHaveBeenCalledTimes(1)
  })

  it('calls onDismiss when backdrop is clicked', async () => {
    const onDismiss = vi.fn()
    const { container } = render(<DraftViolationModal violation={overCapViolation} onDismiss={onDismiss} />)
    // The backdrop is the outermost fixed div
    await userEvent.click(container.firstChild as HTMLElement)
    expect(onDismiss).toHaveBeenCalledTimes(1)
  })
})

describe('DraftViolationModal — position full', () => {
  it('shows the position label in the title', () => {
    render(<DraftViolationModal violation={positionFullViolation} onDismiss={vi.fn()} />)
    expect(screen.getByText('Goalie Slots Full')).toBeInTheDocument()
  })

  it('shows the player name', () => {
    render(<DraftViolationModal violation={positionFullViolation} onDismiss={vi.fn()} />)
    expect(screen.getByText('Test Goalie')).toBeInTheDocument()
  })

  it('shows the slot count', () => {
    render(<DraftViolationModal violation={positionFullViolation} onDismiss={vi.fn()} />)
    // body text unique to the position-full explanation (title already tested above)
    expect(screen.getByText(/you've used all/i)).toBeInTheDocument()
    expect(screen.getByText('2')).toBeInTheDocument()
  })

  it('calls onDismiss when Got it is clicked', async () => {
    const onDismiss = vi.fn()
    render(<DraftViolationModal violation={positionFullViolation} onDismiss={onDismiss} />)
    await userEvent.click(screen.getByRole('button', { name: /got it/i }))
    expect(onDismiss).toHaveBeenCalledTimes(1)
  })

  it('uses correct label for Forward position', () => {
    const fwdFull: Violation = { kind: 'position-full', player: forwardPlayer, slotsFilled: 9, slotsTarget: 9 }
    render(<DraftViolationModal violation={fwdFull} onDismiss={vi.fn()} />)
    expect(screen.getByText('Forward Slots Full')).toBeInTheDocument()
  })

  it('does not show cap breakdown table', () => {
    render(<DraftViolationModal violation={positionFullViolation} onDismiss={vi.fn()} />)
    expect(screen.queryByText('Cap remaining')).not.toBeInTheDocument()
    expect(screen.queryByText('Shortfall')).not.toBeInTheDocument()
  })
})
