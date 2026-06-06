import { useState, useEffect } from 'react'
import { DRAFT_STATE, TEAMS, currentPickingTeamIndex } from '../../data/mockDraft'

export function PickTimer() {
  const [seconds, setSeconds] = useState(DRAFT_STATE.secondsRemaining)
  const pickingTeam = TEAMS[currentPickingTeamIndex(DRAFT_STATE)]
  const isMe = pickingTeam.isMe

  useEffect(() => {
    if (seconds <= 0) return
    const id = setInterval(() => setSeconds(s => Math.max(0, s - 1)), 1000)
    return () => clearInterval(id)
  }, [seconds])

  const pct = seconds / 90
  const urgent = seconds <= 20
  const circumference = 2 * Math.PI * 20

  return (
    <div className="flex items-center gap-4">
      <div className="relative w-12 h-12 shrink-0">
        <svg className="w-12 h-12 -rotate-90" viewBox="0 0 48 48">
          <circle cx="24" cy="24" r="20" fill="none" stroke="#1e293b" strokeWidth="4" />
          <circle
            cx="24" cy="24" r="20" fill="none"
            stroke={urgent ? '#ef4444' : isMe ? '#f59e0b' : '#3b82f6'}
            strokeWidth="4"
            strokeDasharray={circumference}
            strokeDashoffset={circumference * (1 - pct)}
            strokeLinecap="round"
            className="transition-all duration-1000"
          />
        </svg>
        <span className={[
          'absolute inset-0 flex items-center justify-center text-xs font-mono font-bold',
          urgent ? 'text-red-400' : isMe ? 'text-amber-400' : 'text-slate-200',
        ].join(' ')}>
          {seconds > 59 ? `${Math.floor(seconds / 60)}:${String(seconds % 60).padStart(2, '0')}` : seconds}
        </span>
      </div>

      <div>
        <div className="text-xs text-slate-500 uppercase tracking-wider">Now Picking</div>
        <div className={['text-sm font-semibold', isMe ? 'text-amber-300' : 'text-slate-200'].join(' ')}>
          {pickingTeam.name}
          {isMe && <span className="ml-1 text-amber-400"> — that's you!</span>}
        </div>
        <div className="text-xs text-slate-500">
          Round {DRAFT_STATE.currentRound}, Pick {DRAFT_STATE.currentPick} of {DRAFT_STATE.totalTeams}
        </div>
      </div>
    </div>
  )
}
