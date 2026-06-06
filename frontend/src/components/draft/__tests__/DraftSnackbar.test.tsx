import { describe, it, expect, vi, afterEach } from 'vitest'
import { render, screen, act } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { DraftSnackbar } from '../DraftSnackbar'
import type { Player } from '../../../data/mockDraft'

const player: Player = {
  id: '34', name: 'Boone Jenner', position: 'F', team: 'CBJ',
  salary: 4_300_000, age: 31, stats: { goals: 24, assists: 22 },
}

afterEach(() => {
  vi.useRealTimers()
})

describe('DraftSnackbar', () => {
  it('shows the drafted player name', () => {
    render(<DraftSnackbar player={player} onDismiss={vi.fn()} />)
    expect(screen.getByText('Boone Jenner')).toBeInTheDocument()
  })

  it('shows the player position', () => {
    render(<DraftSnackbar player={player} onDismiss={vi.fn()} />)
    // position appears in the pos-pill span
    expect(screen.getByText('F')).toBeInTheDocument()
  })

  it('shows the player salary', () => {
    render(<DraftSnackbar player={player} onDismiss={vi.fn()} />)
    expect(screen.getByText('$4.3M')).toBeInTheDocument()
  })

  it('has an accessible status role for screen readers', () => {
    render(<DraftSnackbar player={player} onDismiss={vi.fn()} />)
    expect(screen.getByRole('status')).toBeInTheDocument()
  })

  it('calls onDismiss when the dismiss button is clicked', async () => {
    const onDismiss = vi.fn()
    render(<DraftSnackbar player={player} onDismiss={onDismiss} />)
    await userEvent.click(screen.getByRole('button', { name: /dismiss/i }))
    expect(onDismiss).toHaveBeenCalledTimes(1)
  })

  it('auto-dismisses after 3 seconds', async () => {
    vi.useFakeTimers()
    const onDismiss = vi.fn()
    render(<DraftSnackbar player={player} onDismiss={onDismiss} />)
    expect(onDismiss).not.toHaveBeenCalled()
    await act(async () => { vi.advanceTimersByTime(3000) })
    expect(onDismiss).toHaveBeenCalledTimes(1)
  })

  it('does not auto-dismiss before 3 seconds', async () => {
    vi.useFakeTimers()
    const onDismiss = vi.fn()
    render(<DraftSnackbar player={player} onDismiss={onDismiss} />)
    await act(async () => { vi.advanceTimersByTime(2999) })
    expect(onDismiss).not.toHaveBeenCalled()
  })

  it('clears the timer on unmount', () => {
    vi.useFakeTimers()
    const onDismiss = vi.fn()
    const { unmount } = render(<DraftSnackbar player={player} onDismiss={onDismiss} />)
    unmount()
    act(() => { vi.advanceTimersByTime(3000) })
    expect(onDismiss).not.toHaveBeenCalled()
  })
})
