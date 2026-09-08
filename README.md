# hockey_season
This is the centralized repository which cobbles together the various projects over the years to get a fantasy hockey league up for a few friends. The stack is going to be varied; I've written a salary scraper in Go, a drafting FE in typescript, who knows what will be added tomorrow.

## Running the mock draft locally

This spins up a fully wired backend + frontend with fake teams and picks so you can develop and test the draft room without coordinating eight real managers.

### Prerequisites

- [Go 1.22+](https://go.dev/dl/)
- [Node.js 20+](https://nodejs.org/)
- [Firefox Developer Edition](https://www.mozilla.org/en-US/firefox/developer/) (or any modern browser)
- `data/hockey_season.db` seeded with NHL players (one-time step below)

### One-time setup: seed the player database

This only needs to be done once per season (or after deleting the database). It hits the NHL API so requires an internet connection.

```bash
cd draft
go run ./seed
```

You should see output like `done: 900 inserted, 0 skipped, 12 unmatched`. The `nhl_players` table is now populated.

### Tab 1 — backend

```bash
cd backend
DRAFT_SECRET=dev-secret go run main.go -mock
```

The `-mock` flag seeds eight fantasy teams and a draft session at round 3, pick 3 (Hat Trick Heroes are up). On shutdown (`Ctrl-C`) all seeded rows are removed automatically. You should see:

```
[mock] session N ready — Hat Trick Heroes code: hat-trick-heroes-code
draft server listening on :8080
```

### Tab 2 — frontend

```bash
cd frontend
npm install        # first time only
npm run dev
```

Open the URL printed by Vite (typically `http://localhost:5173`) in Firefox Developer Edition.

### Joining as a team

When prompted for a team code, use any of the eight mock codes:

| Team | Code |
|------|------|
| Frozen Flames | `frozen-flames-code` |
| Puck Norris | `puck-norris-code` |
| Hat Trick Heroes | `hat-trick-heroes-code` |
| Slapshot Squad | `slapshot-squad-code` |
| Ice Cold Cash | `ice-cold-cash-code` |
| Rink Rulers | `rink-rulers-code` |
| Zamboni Drivers | `zamboni-drivers-code` |
| Five Hole Fellas | `five-hole-fellas-code` |

Hat Trick Heroes is the "my team" perspective (round 3, pick 3 is theirs, cap nearly full, goalie slots taken — useful for testing cap and position-full violations). Use the other codes in additional browser tabs to simulate other managers.

To switch teams, click **Sign out** in the top-right corner and enter a different code.

### Notes

- The backend session ID increments on every restart. The frontend picks up the correct session ID from the login response, so you don't need to change anything between restarts — just sign out and sign back in with your team code.
- Live pick updates are broadcast over WebSocket. Open two tabs with different team codes and draft a player in one — the other updates immediately.
- To reset the draft state entirely, stop the backend (`Ctrl-C`) and restart it with `-mock` again.

## frontend

The [`frontend/`](frontend/README.md) directory contains the browser-based draft room — player browsing, cap tracking, violation modals, and a draft snackbar. See [frontend/README.md](frontend/README.md) for how to run the app and tests.

## draft

The `draft/` directory contains a Go tool that scrapes current NHL player salaries from [capwages.com](https://capwages.com) and writes them to a local JSON file.

### Prerequisites

- [Go 1.22+](https://go.dev/dl/)

### Run the salary scraper

```bash
cd draft
go run .
```

This visits each NHL team's cap page and writes `player-salaries.json` in the current directory. Each entry looks like:

```json
{
  "name": "Connor McDavid",
  "age": "28",
  "pos": "F",
  "salary_cap_hit": "12,500,000",
  "team": "Edmonton Oilers"
}
```

Retired players and unsigned free agents (RFA/UFA) are excluded. Positions are simplified to `F`, `D`, or `G`.

### Seed the player database

After running the scraper, populate `data/hockey_season.db` with all NHL players and their NHL-assigned IDs:

```bash
cd draft
go run ./seed
```

This runs `data/migrations/001_create_nhl_players.sql` (creating the `nhl_players` table if needed), then fetches each team's active roster from the NHL API to resolve player IDs by name. The script is safe to re-run — existing rows are skipped. Players whose names couldn't be matched are inserted with a `null` NHL ID and printed to stderr for manual review.

> **Each season**: update the `nhlSeason` constant in `draft/seed/main.go` before re-running.
