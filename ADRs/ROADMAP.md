# Fantasy Hockey League — Engineering Roadmap

## Stack

| Layer | Technology |
|---|---|
| Backend API | Go (net/http + chi router) |
| Database | SQLite (existing) → PostgreSQL if scale demands |
| Frontend | React 18 + TypeScript (Vite) |
| Styling | Tailwind CSS |
| Real-time | WebSockets (gorilla/websocket) for live draft |
| Email | Resend (or SMTP fallback) |
| Auth | JWT, invite-link based (private league) |
| Scheduler | Go cron (robfig/cron) for stat sync + injury detection |

---

## Epics

1. [Infrastructure & Data Layer](#epic-1-infrastructure--data-layer)
2. [Player Data Pipeline](#epic-2-player-data-pipeline)
3. [League Setup & Draft](#epic-3-league-setup--draft)
4. [Scoring Engine](#epic-4-scoring-engine)
5. [Roster Transactions](#epic-5-roster-transactions)
6. [Injury Handling](#epic-6-injury-handling)
7. [Trades](#epic-7-trades)
8. [Admin & Commissioner Tools](#epic-8-admin--commissioner-tools)

---

## Epic 1: Infrastructure & Data Layer

Foundation that every other epic depends on. Should be completed first.

### INFRA-001 — Go HTTP server scaffold
Set up the Go API entrypoint: chi router, middleware chain (CORS, request logging, error recovery), graceful shutdown, config via environment variables. No business logic — just the skeleton.

### INFRA-002 — Database migrations framework
Integrate a migration runner (golang-migrate or goose). Port `001_create_nhl_players.sql` into the framework. All subsequent schema changes must be managed as numbered migrations.

### INFRA-003 — Core schema migrations

Create the following tables:

- `leagues` (id, name, salary_cap, created_at)
- `users` (id, email, display_name, is_committee_member, created_at)
- `teams` (id, league_id, user_id, name, trades_used, transactions_used)
- `roster_slots` (id, team_id, player_id, slot_type: `active`|`injured`|`substitute`, original_player_id nullable, created_at)
- `transactions` (id, team_id, dropped_player_id, added_player_id, type: `elective`|`injury_sub`|`trade`, created_at)
- `trades` (id, league_id, status: `pending`|`approved`|`rejected`, submitted_by_team_id, reviewed_by_user_id nullable, created_at)
- `trade_legs` (id, trade_id, from_team_id, to_team_id, player_id)
- `matchups` (id, league_id, week_number, home_team_id, away_team_id, home_score, away_score, home_points, away_points)
- `player_game_logs` (id, player_id, game_date, goals, assists, wins, otl, shutouts, games_played)
- `injury_flags` (id, player_id, flagged_at, resolved_at nullable, consecutive_misses int)

### INFRA-004 — Auth: invite-link registration + JWT
Generate single-use invite links (commissioner sends these). On first visit the user sets their display name and password. Issue a JWT (short-lived access + refresh token). Middleware to protect all non-public routes.

### INFRA-005 — Email service integration
Wrap an email provider (Resend recommended) behind an internal `Mailer` interface in Go. Implement: `SendInvite`, `SendInjuryAlert`, `SendInjuryDigest`, `SendTradeNotification`, `SendDraftReminder`. All calls should be async (goroutine + retry queue).

### INFRA-006 — React app scaffold (Vite + TypeScript)
Bootstrap the `frontend/` subdirectory: Vite, React 18, TypeScript strict mode, Tailwind CSS, React Router v6, TanStack Query for server state, Axios for API calls. Add a `Makefile` target to build and serve.

### INFRA-007 — Shared API contract (OpenAPI spec)
Define an `openapi.yaml` covering all endpoints. Use `oapi-codegen` to generate Go server stubs and TypeScript client types. This enforces consistency and avoids hand-rolling fetch calls.

---

## Epic 2: Player Data Pipeline

Mostly building on existing work in `draft/`.

### DATA-001 — Refactor salary scraper to use Spotrac
The rules specify Spotrac as the salary source (not capwages.com). Update `draft/main.go` to target Spotrac. Keep the same `Player` struct output shape. **Acceptance:** `player-salaries.json` populated from Spotrac without manual edits.

### DATA-002 — Stats ingestion job: skaters
Go cron job that hits the NHL Stats API (`/v1/player/{id}/game-log`) daily during the season. Upsert goals and assists per player per game into `player_game_logs`. Run after 2AM ET (games are typically finished).

### DATA-003 — Stats ingestion job: goalies
Same as DATA-002 but extract wins, OTL (overtime losses), and shutouts for players with `position = 'G'`.

### DATA-004 — Consecutive games missed tracker
Part of the stats sync job. After each daily sync, for each player without a game log entry for the last N days, increment their `injury_flags.consecutive_misses`. If they have a log entry, reset to 0 and resolve any open injury flag.

### DATA-005 — LTIR feed
Poll the NHL API (or a secondary source) for LTIR placements. On detection, immediately set `injury_flags` for that player regardless of the consecutive-misses count.

### DATA-006 — Player search API endpoint
`GET /players?q=<name>&position=<F|D|G>&max_salary=<int>&available=true` — returns players not currently on any roster in the league, filtered to the given criteria. Used by both draft UI and transaction UI.

---

## Epic 3: League Setup & Draft

### DRAFT-001 — League creation endpoint
`POST /leagues` — commissioner creates a league, sets the salary cap, and the number of teams. Returns a league ID and the commissioner's team is auto-created.

### DRAFT-002 — Invite and team creation flow
Commissioner sends invites (INFRA-005). Each invite is scoped to a league. On acceptance, a `teams` row is created for the user. Frontend: simple onboarding screen (set team name).

### DRAFT-003 — Randomized draft order generation
`POST /leagues/:id/draft/order` — randomly assign draft positions. Store order in a `draft_order` table (team_id, pick_position). Commissioner triggers this; result is visible to all users.

### DRAFT-004 — Snake draft engine (backend)
WebSocket hub in Go. State machine: `waiting` → `in_progress` → `complete`. Tracks whose turn it is using snake order logic (round 1: picks 1→N, round 2: N→1, etc.). On each pick:
1. Validate the selecting team is next in order.
2. Validate cap compliance post-pick.
3. Validate position slots (max 12F / 6D / 2G).
4. Insert into `roster_slots`.
5. Broadcast updated state to all connected clients.

### DRAFT-005 — Draft room UI (React)
Real-time draft interface. Components:
- `DraftBoard` — grid of all picks by round and team
- `AvailablePlayerList` — filterable/searchable, shows salary and position
- `MyRoster` — current roster with cap usage meter
- `PickTimer` — countdown for the active picker (optional, nice-to-have)
- `DraftOrder` — sidebar showing who picks next

Connects via WebSocket. Non-active pickers can watch but not pick.

### DRAFT-006 — H2H schedule generation
`POST /leagues/:id/schedule/generate` — triggered immediately after the draft completes. Each team plays every other team exactly 3 times across the season. Assign matchups to weeks (Monday–Sunday). Store in `matchups`. Return the full schedule.

### DRAFT-007 — Post-draft salary confirmation flow
Before the draft, pull fresh salaries from Spotrac (DATA-001) and present a confirmation screen to the commissioner showing the new cap figure. Commissioner acknowledges before the draft is unlocked.

---

## Epic 4: Scoring Engine

### SCORE-001 — Aggregate score calculator
Service function that, given a team and a date range (default: season start to today), sums:
- Skater points: goals + assists (1pt each)
- Goalie wins: 2pts each
- Goalie OTL: 1pt each
- Goalie shutouts: 1pt each

Only counts stats from players while they were on the team's roster.

### SCORE-002 — H2H weekly score job
Runs Sunday night after the last game. For each `matchups` row in the current week:
1. Compute each team's score via SCORE-001 for Mon 9AM – Sun EOG.
2. Award points: win=2, loss=0, tie=1 each.
3. Update `matchups.home_score`, `matchups.away_score`, `matchups.home_points`, `matchups.away_points`.

### SCORE-003 — Standings API
`GET /leagues/:id/standings` — returns both:
- Aggregate standings: teams sorted by total season points, tiebreaker by total goals, then by top individual player points.
- H2H standings: teams sorted by H2H points, tiebreaker by straight wins (excluding 1-point ties), then by head-to-head record between tied teams.

### SCORE-004 — Standings page (React)
Two tabs: Aggregate and Head-to-Head. Each shows team name, record/points, and tiebreaker stats. Link through to individual team roster view.

### SCORE-005 — H2H schedule page (React)
Calendar-style view of the season's matchup schedule. Current week highlighted. Clicking a matchup shows the two rosters and their scores for that week.

### SCORE-006 — Team detail page (React)
Shows a team's full roster, current cap usage, transaction count remaining (15 - used), trades remaining (3 - used), and per-player season stats.

---

## Epic 5: Roster Transactions

### TXN-001 — Transaction validation service
Pure Go service (no side effects) that, given a team state and a proposed drop+add, returns `valid` or an error string. Checks:
- Team has fewer than 15 elective transactions used.
- Dropped player is on the team.
- Added player is in the unclaimed pool.
- Added player is the same position as dropped player (same position slot).
- Resulting roster is cap-compliant.
- Resulting roster does not violate position maximums.

### TXN-002 — Execute transaction endpoint
`POST /teams/:id/transactions` with body `{ drop_player_id, add_player_id }`. Calls TXN-001. On success: update `roster_slots`, insert `transactions` row, increment `teams.transactions_used`. Returns updated roster and cap.

### TXN-003 — Elective transaction UI (React)
On the team management page: a "Make a Move" button that opens a two-step flow:
1. Select a player to drop (shows their position and salary).
2. Search available players filtered to same position, within cap budget remaining after drop.

Shows live cap impact before confirming. Disabled when 15 transactions exhausted.

---

## Epic 6: Injury Handling

### INJ-001 — Injury detection trigger
Extend DATA-004/DATA-005 so that when a player is newly flagged as injured (LTIR or 3 consecutive misses), the system:
1. Marks their `roster_slots` row as `slot_type = 'injured'`.
2. Triggers an email to the owning team's manager (INFRA-005 `SendInjuryAlert`).

### INJ-002 — Substitute player selection endpoint
`POST /teams/:id/injury-subs` with body `{ injured_player_id, substitute_player_id }`. Validation:
- Injured player is on this team with `slot_type = 'injured'`.
- Substitute is in the unclaimed pool.
- Substitute plays the same position as the injured player.
- Substitute's cap hit ≤ the *original* injured player's cap hit (handles chained injuries: look up the chain to find the root player's cap).
- Insert a new `roster_slots` row for the sub with `slot_type = 'substitute'` and `original_player_id` pointing to the root injured player.

### INJ-003 — Chained injury logic
When a substitute player is subsequently flagged as injured (3 missed games while on a team):
1. Find the root injured player via `original_player_id` chain.
2. Return the current substitute to the unclaimed pool (`roster_slots` deleted).
3. Allow the team to pick a new substitute whose cap ≤ root player's cap.

### INJ-004 — Injury recovery detection
When DATA-004 detects a player has appeared in a game log after being flagged:
1. Mark `injury_flags.resolved_at`.
2. Return the substitute to the unclaimed pool.
3. Restore the original player to `slot_type = 'active'`.
4. Email the team manager.

### INJ-005 — Cut injured player flow
`POST /teams/:id/cut` with body `{ player_id }` where player is currently injured. Logic:
1. Remove substitute from roster and return to unclaimed pool.
2. Remove original injured player from roster and return to unclaimed pool.
3. The cap slot is now free using the original player's salary, available for a new elective transaction.

**Note:** this consumes one of the team's 15 elective transactions.

### INJ-006 — Daily injury digest email
Cron job at end of each day: compile all injury sub moves that happened since the previous digest. Send a single email to all league members (INFRA-005 `SendInjuryDigest`) listing changes. Skip days with no activity.

### INJ-007 — Injury status UI (React)
On the team page, injured players show a distinct badge. A separate "Injuries" panel lets the manager select a substitute from the eligible pool (same position, cap-filtered). Shows the eligible sub list with cap hits.

---

## Epic 7: Trades

### TRADE-001 — Trade proposal endpoint
`POST /leagues/:id/trades` — body contains two sets of player IDs being swapped. Validation (pre-approval):
- Both teams have trade slots remaining (< 3 used).
- Each team would be cap and position compliant after the swap.
- No player appears in both sides.
Inserts a `trades` row with `status = 'pending'` and the corresponding `trade_legs`. Notifies ethics committee members by email.

### TRADE-002 — Ethics committee review endpoint
`POST /trades/:id/review` with body `{ decision: 'approved'|'rejected', notes: string }`. Only callable by users with `is_committee_member = true`. On approval:
1. Move players between rosters.
2. Increment `teams.trades_used` for both teams.
3. Email all league members.

On rejection, email the proposing team with the notes.

### TRADE-003 — Trade proposal UI (React)
Multi-step form: select the other team → select players to send → select players to receive → review cap impact for both sides → submit. Show remaining trade slots for both teams. Live cap compliance indicator.

### TRADE-004 — Ethics committee dashboard (React)
Accessible only to committee members. Lists pending trades with full roster impact visualization. Approve/reject with optional notes. Links to both teams' full rosters for due diligence.

### TRADE-005 — Trade history page (React)
League-level trade log. Shows all approved and rejected trades with dates and (optionally) rejection notes. Visible to all league members.

---

## Epic 8: Admin & Commissioner Tools

### ADMIN-001 — Commissioner dashboard (React)
Protected route. Controls: set/update salary cap, lock/unlock draft, trigger draft order randomization, view all pending trades, view injury log, send custom email to all members.

### ADMIN-002 — Manual player stat override
`PATCH /player-game-logs/:id` — commissioner can correct a stat entry if the NHL API returned bad data. Audit-logged.

### ADMIN-003 — Manual injury flag override
`POST /players/:id/injury-flags` and `DELETE /players/:id/injury-flags/:flag_id` — force-flag or force-resolve an injury. Useful if the automated detection lags. Sends appropriate emails.

### ADMIN-004 — Transaction/trade slot reset
`PATCH /teams/:id` — allow commissioner to reset `transactions_used` or `trades_used` in edge cases (commissioner discretion). Audit-logged.

### ADMIN-005 — Season state machine
League-level status: `setup` → `draft_ready` → `drafting` → `in_season` → `complete`. Gating: transactions and scoring only work in `in_season`. Draft only works in `drafting`. Prevents accidental actions.

---

## Dependency Order

```
INFRA-001 → INFRA-002 → INFRA-003
INFRA-003 → all other epics
INFRA-004 → DRAFT-002, all authenticated endpoints
INFRA-005 → INJ-001, INJ-006, TRADE-001, TRADE-002
INFRA-006 → all React tickets
INFRA-007 → INFRA-006 (generates TS types)

DATA-001 → DRAFT-007
DATA-002 + DATA-003 + DATA-004 + DATA-005 → INJ-001, SCORE-001

DRAFT-001 → DRAFT-002 → DRAFT-003 → DRAFT-004 → DRAFT-005
DRAFT-004 → DRAFT-006 (schedule generated post-draft)

SCORE-001 → SCORE-002 → SCORE-003 → SCORE-004, SCORE-005

TXN-001 → TXN-002 → TXN-003

INJ-001 → INJ-002 → INJ-003
INJ-002 → INJ-007
INJ-004 → INJ-007
INJ-005 → INJ-007

TRADE-001 → TRADE-002 → TRADE-003, TRADE-004
```

---

## Milestone Summary

| Milestone | Epics | Goal |
|---|---|---|
| M1: Foundations | INFRA 1–7 | Running API + React app + DB schema |
| M2: Data Ready | DATA 1–6 | Salary + stats pipeline live |
| M3: Draft | DRAFT 1–7 | Full live snake draft with cap validation |
| M4: Season Start | SCORE 1–6, TXN 1–3 | Scoring + elective transactions |
| M5: Injuries | INJ 1–7 | Automated injury detection + substitution |
| M6: Trades | TRADE 1–5 | Trade proposal + committee workflow |
| M7: Polish | ADMIN 1–5 | Commissioner tools + season state machine |
