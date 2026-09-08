import { useState } from 'react'
import { DraftBoard } from '../components/draft/DraftBoard'
import { AvailablePlayerList } from '../components/draft/AvailablePlayerList'
import { MyTeamView } from '../components/draft/MyTeamView'
import { MyRoster } from '../components/draft/MyRoster'
import { DraftOrder } from '../components/draft/DraftOrder'
import { PickTimer } from '../components/draft/PickTimer'
import { useDraft } from '../context/DraftContext'

type MainTab = 'players' | 'my-team'

export function DraftRoom({ onSignOut }: { onSignOut: () => void }) {
  const [mainTab, setMainTab] = useState<MainTab>('players')
  const { draftState, isLoading, error } = useDraft()

  if (isLoading) {
    return (
      <div className="flex h-screen items-center justify-center bg-slate-950">
        <div className="text-slate-500 text-sm">Loading draft…</div>
      </div>
    )
  }

  if (error) {
    return (
      <div className="flex h-screen flex-col items-center justify-center gap-3 bg-slate-950">
        <p className="text-red-400 text-sm">{error}</p>
        <button onClick={onSignOut} className="text-xs text-slate-500 underline">Sign out</button>
      </div>
    )
  }

  const isDraftComplete = draftState?.status === 'complete'

  return (
    <div className="flex flex-col h-screen bg-slate-950 overflow-hidden">
      {/* Header */}
      <header className="flex items-center justify-between px-4 py-3 bg-slate-900 border-b border-slate-800 shrink-0">
        <div className="flex items-center gap-3">
          <div className="flex items-center gap-2">
            <div className="w-6 h-6 rounded bg-blue-600 flex items-center justify-center">
              <svg className="w-4 h-4 text-white" fill="currentColor" viewBox="0 0 20 20">
                <path d="M10 2a8 8 0 100 16A8 8 0 0010 2zm0 14a6 6 0 110-12 6 6 0 010 12z" />
                <path d="M10 5a1 1 0 011 1v4a1 1 0 01-.293.707l-2 2a1 1 0 01-1.414-1.414L9 9.586V6a1 1 0 011-1z" />
              </svg>
            </div>
            <span className="font-bold text-white tracking-tight">Draftr</span>
          </div>
          <span className="text-slate-600">·</span>
          <span className="text-sm text-slate-400 font-medium">2026–27 Season Draft</span>
          {isDraftComplete ? (
            <span className="inline-flex items-center gap-1.5 px-2 py-0.5 rounded-full bg-slate-700 border border-slate-600 text-slate-400 text-xs font-medium">
              Complete — Read Only
            </span>
          ) : (
            <span className="inline-flex items-center gap-1.5 px-2 py-0.5 rounded-full bg-green-500/15 border border-green-500/30 text-green-400 text-xs font-medium">
              <span className="w-1.5 h-1.5 rounded-full bg-green-400 animate-pulse" />
              Live
            </span>
          )}
        </div>
        <div className="flex items-center gap-3">
          {!isDraftComplete && <PickTimer />}
          <button onClick={onSignOut} className="text-xs text-slate-600 hover:text-slate-400 transition-colors">
            Sign out
          </button>
        </div>
      </header>

      {/* Draft Board */}
      <div className="px-4 py-3 bg-slate-900/50 border-b border-slate-800 shrink-0">
        <div className="flex items-center gap-2 mb-2">
          <h2 className="text-xs font-semibold text-slate-400 uppercase tracking-wider">Draft Board</h2>
          {draftState && !isDraftComplete && (
            <span className="text-xs text-slate-600">
              Round {draftState.currentRound} of {draftState.totalRounds} · Pick {draftState.currentPick} of {draftState.totalTeams}
            </span>
          )}
          {isDraftComplete && (
            <span className="text-xs text-slate-600">
              {draftState!.totalRounds} rounds · {draftState!.totalTeams} teams · all picks complete
            </span>
          )}
        </div>
        <DraftBoard />
      </div>

      {/* Main body: 3-column layout */}
      <div className="flex flex-1 min-h-0 overflow-hidden">
        <aside className="w-52 shrink-0 bg-slate-900/30 border-r border-slate-800 flex flex-col p-3 overflow-hidden">
          <DraftOrder />
        </aside>

        <main className="flex-1 flex flex-col p-4 overflow-hidden bg-slate-950">
          {isDraftComplete ? (
            <div className="flex flex-col items-center justify-center h-full gap-4 text-center">
              <div className="w-12 h-12 rounded-full bg-green-500/15 border border-green-500/30 flex items-center justify-center">
                <svg className="w-6 h-6 text-green-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                </svg>
              </div>
              <div>
                <p className="text-slate-100 font-semibold text-sm">Draft Complete</p>
                <p className="text-slate-500 text-xs mt-1">All {draftState!.totalRounds * draftState!.totalTeams} picks have been made. The board above is read-only.</p>
              </div>
              <p className="text-xs text-slate-600">Head to <span className="text-slate-400">My Team</span> or <span className="text-slate-400">Standings</span> to continue.</p>
            </div>
          ) : (
            <>
              <div className="flex items-center gap-1 mb-3 bg-slate-900 rounded-lg p-1 self-start">
                {(['players', 'my-team'] as const).map(tab => (
                  <button
                    key={tab}
                    onClick={() => setMainTab(tab)}
                    className={[
                      'px-4 py-1.5 rounded-md text-xs font-semibold transition-all',
                      mainTab === tab
                        ? 'bg-slate-700 text-slate-100 shadow'
                        : 'text-slate-500 hover:text-slate-300',
                    ].join(' ')}
                  >
                    {tab === 'players' ? 'Players' : 'My Team'}
                  </button>
                ))}
              </div>
              {mainTab === 'players' ? <AvailablePlayerList /> : <MyTeamView />}
            </>
          )}
        </main>

        <aside className="w-64 shrink-0 bg-slate-900/30 border-l border-slate-800 flex flex-col p-3 overflow-hidden">
          <div className="flex items-center gap-2 mb-3">
            <h2 className="text-sm font-semibold text-slate-200">My Roster</h2>
          </div>
          <MyRoster />
        </aside>
      </div>
    </div>
  )
}
