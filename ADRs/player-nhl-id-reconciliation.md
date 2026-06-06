# ADR: Player ↔ NHL ID Reconciliation

**Status**: Accepted  
**Related issue**: #3 — Define a player persistence layer

---

## Context

The salary scraper (`draft/main.go`) pulls player data from capwages.com and produces `data/player-salaries.json`. Each record has a player's name, team, position, age, and cap hit — but no NHL-assigned player ID.

The NHL API (used in daily game scraping, issue #5) identifies players by an integer ID (e.g. `8478402` for Connor McDavid). Every subsequent feature that touches player data — points tracking, injury detection, roster management — will need to join on this ID.

The reconciliation needs to run **once per season**, before the draft, and produce the `nhl_players` reference table.

---

## Decision

Match each salary entry to an NHL player ID by scoping the search to that player's known team, then matching on normalized full name.

**Concrete steps:**
1. For each unique team in `player-salaries.json`, map the capwages team name to an NHL API tricode (static lookup table, see `draft/seed/main.go`).
2. Fetch `GET https://api-web.nhle.com/v1/roster/{tricode}/{season}` — one call per team (32 total).
3. Build a `lowercase(firstName + " " + lastName) → nhl_id` lookup from the API response.
4. Match each salary entry against its team's lookup. Exact normalized match wins.
5. Players with no match are inserted with `nhl_id = NULL` and logged; they require manual resolution.

---

## Alternatives Considered

**Global name search without team scoping**  
The NHL does not offer a free-text player search endpoint. Fuzzy-matching a name across all ~800 active players would require an edit-distance pass with ambiguity for common names (e.g. two "Eric Staal" equivalents in history). Team-scoped matching narrows the field to ~25 players per team and makes exact matching sufficient for the vast majority of cases.

**Scrape NHL IDs directly from capwages**  
capwages URLs do not expose NHL player IDs; they use their own slugs. Adding a second scraping pass per player would be ~1600 HTTP requests vs. 32 today, and is fragile against DOM changes.

**Manual curation**  
Maintaining a handcrafted name → ID CSV is brittle across trades and signings and doesn't scale past one season.

---

## Rationale

Team-scoped exact name matching is the lowest-effort approach that covers ~95%+ of players correctly. The main sources of mismatch are:

- **Mid-season trades**: capwages lists the player under their current contract team; the NHL API roster reflects their current NHL team. A player traded after the salary scrape runs will not match.
- **Name formatting differences**: accented characters, hyphens, or nicknames may differ between capwages and the NHL API (e.g. "Marc-Edouard Vlasic" vs "Marc-Édouard Vlasic").
- **AHL/two-way contracts**: some capwages entries may be on an NHL roster page but absent from the active NHL API roster at scrape time.

These edge cases are logged and left with `nhl_id = NULL`. Because the table is only seeded once per season, a short manual cleanup pass over the null rows is acceptable and preferable to a complex fuzzy-matching pipeline.

---

## Implications

- The seed script (`draft/seed/main.go`) must be re-run at the start of each season after the salary scrape. Update `nhlSeason` const to match (e.g. `20252026`).
- `nhl_id` is nullable in `nhl_players`. Any downstream query joining on NHL ID must account for null rows.
- The tricode lookup table in `draft/seed/main.go` must be updated if teams relocate or rebrand (e.g. Utah Hockey Club → Utah Mammoth in 2025).
- Future improvement: after the daily game scraper (issue #5) is wired up, unmatched players could be auto-resolved by cross-referencing scoring events with the known null-ID set.

---

## Related

- Issue #3: Define a player persistence layer
- Issue #5: Scraping Daily NHL Games
- `data/migrations/001_create_nhl_players.sql`
- `draft/seed/main.go`
