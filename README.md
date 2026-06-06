# hockey_season
This is the centralized repository which cobbles together the various projects over the years to get a fantasy hockey league up for a few friends. The stack is going to be varied; I've written a salary scraper in Go, a drafting FE in typescript, who knows what will be added tomorrow.

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
