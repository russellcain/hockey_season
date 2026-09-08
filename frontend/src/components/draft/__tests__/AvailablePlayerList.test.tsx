import { describe, it, expect } from 'vitest'
import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { AvailablePlayerList } from '../AvailablePlayerList'
import { MockDraftProvider } from './testUtils'

function renderWithProvider() {
  return render(
    <MockDraftProvider>
      <AvailablePlayerList />
    </MockDraftProvider>
  )
}

// Known facts from mock data that drive these tests:
//   capUsed = $77M  →  capRemaining = $5.5M
//   Goalie slots full: Andersen + Oettinger (2/2)
//   isMyTurn = true (round 3, pick 3 resolves to my team)
//
//   Over-cap player  : Connor McDavid ($12.5M, F, EDM)
//   Position-full    : Linus Ullmark  ($5M,   G, BOS)  — affordable but G slot full
//   Draftable player : Boone Jenner   ($4.3M, F, CBJ)  — under cap, F slot open

describe('AvailablePlayerList', () => {
  describe('initial render', () => {
    it('renders available player names', () => {
      renderWithProvider()
      expect(screen.getByText('Connor McDavid')).toBeInTheDocument()
      expect(screen.getByText('Boone Jenner')).toBeInTheDocument()
    })

    it('shows the "it\'s your pick" banner when it is my turn', () => {
      renderWithProvider()
      expect(screen.getByText(/it's your pick/i)).toBeInTheDocument()
    })

    it('shows a players available count', () => {
      renderWithProvider()
      expect(screen.getByText(/players available/i)).toBeInTheDocument()
    })
  })

  describe('position filter', () => {
    it('shows only forwards when F is selected', async () => {
      renderWithProvider()
      await userEvent.click(screen.getByRole('button', { name: 'F' }))
      // McDavid (F) should be visible
      expect(screen.getByText('Connor McDavid')).toBeInTheDocument()
      // Ullmark (G) should not be visible
      expect(screen.queryByText('Linus Ullmark')).not.toBeInTheDocument()
    })

    it('shows only goalies when G is selected', async () => {
      renderWithProvider()
      await userEvent.click(screen.getByRole('button', { name: 'G' }))
      expect(screen.getByText('Linus Ullmark')).toBeInTheDocument()
      expect(screen.queryByText('Connor McDavid')).not.toBeInTheDocument()
    })

    it('restores all players when ALL is re-selected', async () => {
      renderWithProvider()
      await userEvent.click(screen.getByRole('button', { name: 'F' }))
      await userEvent.click(screen.getByRole('button', { name: 'ALL' }))
      expect(screen.getByText('Linus Ullmark')).toBeInTheDocument()
    })
  })

  describe('hide taken toggle', () => {
    it('hides drafted players when activated', async () => {
      renderWithProvider()
      // Connor McDavid is available (not drafted), Draisaitl is drafted (my pick, shown greyed out)
      // With hide taken on, drafted players are removed
      await userEvent.click(screen.getByRole('button', { name: /hide taken/i }))
      // All visible players should not have "Drafted by" text
      const draftedLabels = screen.queryAllByText(/drafted by/i)
      expect(draftedLabels).toHaveLength(0)
    })

    it('shows drafted players again when deactivated', async () => {
      renderWithProvider()
      await userEvent.click(screen.getByRole('button', { name: /hide taken/i }))
      await userEvent.click(screen.getByRole('button', { name: /hide taken/i }))
      expect(screen.getAllByText(/drafted by/i).length).toBeGreaterThan(0)
    })
  })

  describe('draftable only filter', () => {
    it('hides over-cap players when activated', async () => {
      renderWithProvider()
      await userEvent.click(screen.getByRole('button', { name: /draftable only/i }))
      expect(screen.queryByText('Connor McDavid')).not.toBeInTheDocument()
    })

    it('hides position-full players when activated', async () => {
      renderWithProvider()
      await userEvent.click(screen.getByRole('button', { name: /draftable only/i }))
      expect(screen.queryByText('Linus Ullmark')).not.toBeInTheDocument()
    })

    it('shows Boone Jenner (genuinely draftable) when activated', async () => {
      renderWithProvider()
      await userEvent.click(screen.getByRole('button', { name: /draftable only/i }))
      expect(screen.getByText('Boone Jenner')).toBeInTheDocument()
    })

    it('composing draftable with position filter further narrows results', async () => {
      renderWithProvider()
      await userEvent.click(screen.getByRole('button', { name: /draftable only/i }))
      await userEvent.click(screen.getByRole('button', { name: 'G' }))
      // No draftable goalies (slots full), so list should be empty
      expect(screen.getByText(/no players match/i)).toBeInTheDocument()
    })
  })

  describe('over-cap violation', () => {
    it('shows the over-cap modal when clicking an over-cap player', async () => {
      renderWithProvider()
      const row = screen.getByText('Connor McDavid').closest('[class*="player-row"]') as HTMLElement
        ?? screen.getByText('Connor McDavid').closest('div[class]') as HTMLElement
      await userEvent.click(row)
      expect(screen.getByText('Over Cap Limit')).toBeInTheDocument()
    })

    it('shows correct shortfall in the modal', async () => {
      renderWithProvider()
      await userEvent.click(screen.getByText('Connor McDavid').closest('div[class]') as HTMLElement)
      // Scope to the shortfall row — $7M also appears on Landeskog's row in the player list
      const shortfallRow = screen.getByText('Shortfall').parentElement!
      expect(shortfallRow).toHaveTextContent('$7M')
    })

    it('dismisses the over-cap modal when Got it is clicked', async () => {
      renderWithProvider()
      await userEvent.click(screen.getByText('Connor McDavid').closest('div[class]') as HTMLElement)
      await userEvent.click(screen.getByRole('button', { name: /got it/i }))
      expect(screen.queryByText('Over Cap Limit')).not.toBeInTheDocument()
    })
  })

  describe('position-full violation', () => {
    it('shows the position-full modal when clicking Ullmark (G slots full)', async () => {
      renderWithProvider()
      const row = screen.getByText('Linus Ullmark').closest('div[class]') as HTMLElement
      await userEvent.click(row)
      expect(screen.getByText('Goalie Slots Full')).toBeInTheDocument()
    })

    it('shows the correct slot count in the modal body', async () => {
      renderWithProvider()
      await userEvent.click(screen.getByText('Linus Ullmark').closest('div[class]') as HTMLElement)
      expect(screen.getByText(/you've used all/i)).toBeInTheDocument()
    })

    it('dismisses position-full modal when Got it is clicked', async () => {
      renderWithProvider()
      await userEvent.click(screen.getByText('Linus Ullmark').closest('div[class]') as HTMLElement)
      await userEvent.click(screen.getByRole('button', { name: /got it/i }))
      expect(screen.queryByText('Goalie Slots Full')).not.toBeInTheDocument()
    })
  })

  describe('successful draft and snackbar', () => {
    it('shows the snackbar after drafting a draftable player', async () => {
      renderWithProvider()
      // Click Boone Jenner — the one genuinely draftable player
      const row = screen.getByText('Boone Jenner').closest('div[class]') as HTMLElement
      await userEvent.click(row)
      expect(screen.getByRole('status')).toBeInTheDocument()
      expect(within(screen.getByRole('status')).getByText('Boone Jenner')).toBeInTheDocument()
    })

    it('snackbar shows the drafted player salary', async () => {
      renderWithProvider()
      await userEvent.click(screen.getByText('Boone Jenner').closest('div[class]') as HTMLElement)
      expect(within(screen.getByRole('status')).getByText('$4.3M')).toBeInTheDocument()
    })

    it('drafted player is marked as taken after draft', async () => {
      renderWithProvider()
      // Count existing "Drafted by Hat Trick Heroes" labels (my 6 pre-existing picks)
      const beforeCount = screen.getAllByText(/drafted by hat trick heroes/i).length
      await userEvent.click(screen.getByText('Boone Jenner').closest('div[class]') as HTMLElement)
      // Dismiss snackbar so Jenner's name doesn't appear there too
      await userEvent.click(screen.getByRole('button', { name: /dismiss/i }))
      expect(screen.getAllByText(/drafted by hat trick heroes/i)).toHaveLength(beforeCount + 1)
    })

    it('drafted player is excluded when draftable filter is on', async () => {
      renderWithProvider()
      await userEvent.click(screen.getByRole('button', { name: /draftable only/i }))
      expect(screen.getByText('Boone Jenner')).toBeInTheDocument()

      await userEvent.click(screen.getByText('Boone Jenner').closest('div[class]') as HTMLElement)
      // Dismiss snackbar — it keeps Jenner's name in the DOM after draft
      await userEvent.click(screen.getByRole('button', { name: /dismiss/i }))

      // Jenner is now drafted, draftable filter removes them
      expect(screen.queryByText('Boone Jenner')).not.toBeInTheDocument()
    })

    it('snackbar is dismissed by its dismiss button', async () => {
      renderWithProvider()
      await userEvent.click(screen.getByText('Boone Jenner').closest('div[class]') as HTMLElement)
      await userEvent.click(screen.getByRole('button', { name: /dismiss/i }))
      expect(screen.queryByRole('status')).not.toBeInTheDocument()
    })
  })
})
