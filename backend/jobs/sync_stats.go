package jobs

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"hockey_season/backend/email"
	"hockey_season/backend/store"
)

// nhlTeams lists every current franchise abbreviation the NHL roster API accepts.
var nhlTeams = []string{
	"ANA", "BOS", "BUF", "CGY", "CAR", "CHI", "COL", "CBJ", "DAL", "DET",
	"EDM", "FLA", "LAK", "MIN", "MTL", "NSH", "NJD", "NYI", "NYR", "OTT",
	"PHI", "PIT", "SJS", "SEA", "STL", "TBL", "TOR", "UTA", "VAN", "VGK",
	"WSH", "WPG",
}

// SyncStats is the daily 2AM job: it first resolves any missing nhl_ids by
// name-matching against live NHL rosters, then fetches game logs for every
// player that has an nhl_id and upserts them.  Newly injured players trigger
// an email to their manager and the league commissioner.
func SyncStats(st *store.Store, mailer email.Sender) {
	log.Println("[sync_stats] starting stats sync")
	start := time.Now()

	if err := matchNHLIDs(st); err != nil {
		log.Printf("[sync_stats] nhl id matching: %v", err)
	}

	players, err := st.ListPlayers()
	if err != nil {
		log.Printf("[sync_stats] list players: %v", err)
		return
	}

	synced, skipped := 0, 0
	for _, p := range players {
		if err := syncPlayerStats(st, mailer, p.ID); err != nil {
			log.Printf("[sync_stats] player %d (%s): %v", p.ID, p.Name, err)
		} else if nhlIDForPlayer(st, p.ID) != 0 {
			synced++
		} else {
			skipped++
		}
	}

	log.Printf("[sync_stats] done in %v — synced %d, skipped %d (no nhl_id)",
		time.Since(start), synced, skipped)
}

// ── NHL ID matching ───────────────────────────────────────────────────────────

type nhlRosterPlayer struct {
	ID        int `json:"id"`
	FirstName struct {
		Default string `json:"default"`
	} `json:"firstName"`
	LastName struct {
		Default string `json:"default"`
	} `json:"lastName"`
}

type nhlRosterResp struct {
	Forwards   []nhlRosterPlayer `json:"forwards"`
	Defensemen []nhlRosterPlayer `json:"defensemen"`
	Goalies    []nhlRosterPlayer `json:"goalies"`
}

// matchNHLIDs fetches the current NHL roster for every team and updates
// nhl_id for any player in our DB whose name matches (case-insensitive).
// Already-matched players (nhl_id IS NOT NULL) are skipped.
func matchNHLIDs(st *store.Store) error {
	// Only bother if any players are still unmatched.
	var unmatched int
	st.DB().QueryRow(`SELECT COUNT(*) FROM nhl_players WHERE nhl_id IS NULL`).Scan(&unmatched)
	if unmatched == 0 {
		return nil
	}
	log.Printf("[sync_stats] resolving nhl_id for %d unmatched players", unmatched)

	// Build name → nhl_id map from every team's roster.
	nameToID := make(map[string]int, 800)
	season := currentSeasonStr()
	client := &http.Client{Timeout: 10 * time.Second}

	for _, abbr := range nhlTeams {
		url := fmt.Sprintf("https://api-web.nhle.com/v1/roster/%s/%s", abbr, season)
		resp, err := client.Get(url)
		if err != nil {
			log.Printf("[sync_stats] roster %s: %v", abbr, err)
			continue
		}
		var r nhlRosterResp
		if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
			resp.Body.Close()
			log.Printf("[sync_stats] decode roster %s: %v", abbr, err)
			continue
		}
		resp.Body.Close()

		all := append(append(r.Forwards, r.Defensemen...), r.Goalies...)
		for _, p := range all {
			fullName := strings.ToLower(p.FirstName.Default + " " + p.LastName.Default)
			nameToID[fullName] = p.ID
		}
	}

	// Update nhl_id for matched players.
	rows, err := st.DB().Query(`SELECT id, name FROM nhl_players WHERE nhl_id IS NULL`)
	if err != nil {
		return fmt.Errorf("query unmatched: %w", err)
	}
	defer rows.Close()

	updated := 0
	for rows.Next() {
		var id int
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			continue
		}
		nhlID, ok := nameToID[strings.ToLower(name)]
		if !ok {
			continue
		}
		if _, err := st.DB().Exec(`UPDATE nhl_players SET nhl_id = ? WHERE id = ?`, nhlID, id); err != nil {
			log.Printf("[sync_stats] set nhl_id for %s: %v", name, err)
			continue
		}
		updated++
	}
	log.Printf("[sync_stats] matched %d/%d players to nhl_id", updated, unmatched)
	return nil
}

// currentSeasonStr returns the NHL season key for the current season,
// e.g. "20252026" for the 2025-26 season.
func currentSeasonStr() string {
	now := time.Now()
	yr := now.Year()
	if now.Month() < time.October {
		yr--
	}
	return fmt.Sprintf("%d%d", yr, yr+1)
}

// ── Per-player stats sync ─────────────────────────────────────────────────────

type nhlGameLogEntry struct {
	GameDate string `json:"gameDate"`
	Goals    int    `json:"goals"`
	Assists  int    `json:"assists"`
	Wins     int    `json:"wins"`
	OTLosses int    `json:"otLosses"`
	Shutouts int    `json:"shutouts"`
}

type nhlGameLogResp struct {
	GameLog []nhlGameLogEntry `json:"gameLog"`
}

func nhlIDForPlayer(st *store.Store, playerID int) int {
	var nhlID int
	st.DB().QueryRow(`SELECT COALESCE(nhl_id, 0) FROM nhl_players WHERE id = ?`, playerID).Scan(&nhlID)
	return nhlID
}

func syncPlayerStats(st *store.Store, mailer email.Sender, playerID int) error {
	nhlID := nhlIDForPlayer(st, playerID)
	if nhlID == 0 {
		return nil
	}

	url := fmt.Sprintf("https://api-web.nhle.com/v1/player/%d/game-log/now", nhlID)
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("fetch: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("http %d", resp.StatusCode)
	}

	var data nhlGameLogResp
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return fmt.Errorf("decode: %w", err)
	}

	// Track the most recent game date for injury detection.
	var latestGame time.Time
	for _, entry := range data.GameLog {
		if err := st.UpsertGameLog(playerID, entry.GameDate,
			entry.Goals, entry.Assists, entry.Wins, entry.OTLosses, entry.Shutouts); err != nil {
			log.Printf("[sync_stats] upsert log player %d date %s: %v", playerID, entry.GameDate, err)
			continue
		}
		if t, err := time.Parse("2006-01-02", entry.GameDate); err == nil && t.After(latestGame) {
			latestGame = t
		}
	}

	// Injury detection: if a player hasn't played in 5+ days, increment misses.
	// If misses ≥ 3 and not already flagged, mark injured and notify.
	updateInjuryStatus(st, mailer, playerID, latestGame)
	return nil
}

// updateInjuryStatus checks whether a player is overdue and flags/resolves
// injuries across all leagues they are rostered in.
func updateInjuryStatus(st *store.Store, mailer email.Sender, playerID int, lastPlayed time.Time) {
	daysSinceGame := int(time.Since(lastPlayed).Hours() / 24)

	// Resolve: they played within the last 2 days — clear any active injury.
	if daysSinceGame <= 2 && !lastPlayed.IsZero() {
		rows, err := st.DB().Query(`
			SELECT DISTINCT rs.league_id FROM roster_slots rs
			WHERE rs.player_id = ? AND rs.slot_type = 'injured'
		`, playerID)
		if err != nil {
			return
		}
		defer rows.Close()
		for rows.Next() {
			var lid int
			rows.Scan(&lid)
			if err := st.ResolvePlayerInjury(lid, playerID); err == nil {
				sendInjuryResolvedEmail(st, mailer, lid, playerID)
			}
		}
		// Also reset consecutive_misses counter.
		st.DB().Exec(`UPDATE injury_flags SET consecutive_misses = 0 WHERE player_id = ?`, playerID)
		return
	}

	// Possible injury: ≥ 5 days without a game.
	if daysSinceGame < 5 {
		return
	}

	// Increment consecutive_misses.
	st.DB().Exec(`
		INSERT INTO injury_flags (player_id, consecutive_misses, is_ltir)
		VALUES (?, 1, 0)
		ON CONFLICT(player_id) DO UPDATE SET consecutive_misses = consecutive_misses + 1
	`, playerID)

	var misses int
	st.DB().QueryRow(`SELECT consecutive_misses FROM injury_flags WHERE player_id = ?`, playerID).Scan(&misses)
	if misses < 3 {
		return
	}

	// Flag injured in every league this player is actively rostered in.
	rows, err := st.DB().Query(`
		SELECT DISTINCT rs.league_id FROM roster_slots rs
		WHERE rs.player_id = ? AND rs.slot_type = 'active'
	`, playerID)
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var lid int
		rows.Scan(&lid)
		if err := st.FlagPlayerInjured(lid, playerID, false); err == nil {
			sendInjuryAlertEmail(st, mailer, lid, playerID)
		}
	}
}

// ── Email helpers ─────────────────────────────────────────────────────────────

func sendInjuryAlertEmail(st *store.Store, mailer email.Sender, leagueID, playerID int) {
	p, err := st.GetPlayer(playerID)
	if err != nil {
		return
	}
	owner, err := st.GetPlayerOwnerTeam(leagueID, playerID)
	if err != nil {
		return
	}

	subject := fmt.Sprintf("[Draftr] ⚠️ Injury alert: %s (%s)", p.Name, p.NhlTeamCode)
	html := fmt.Sprintf(`
<p>Hi %s,</p>
<p><strong>%s</strong> (%s · %s) on your team <strong>%s</strong> has not appeared in an NHL game for 5+ days and has been automatically marked as <strong>injured</strong>.</p>
<p>Visit the <a href="#">Injuries page</a> to add an eligible substitute.</p>
<p>— Draftr</p>
`, owner.Manager, p.Name, p.Position, p.NhlTeam, owner.Name)

	targets := buildTargets(st, leagueID, owner)
	for _, to := range targets {
		if err := mailer.Send(to, subject, html); err != nil {
			log.Printf("[email] injury alert to %s: %v", to, err)
		}
	}
}

func sendInjuryResolvedEmail(st *store.Store, mailer email.Sender, leagueID, playerID int) {
	p, err := st.GetPlayer(playerID)
	if err != nil {
		return
	}
	owner, err := st.GetPlayerOwnerTeam(leagueID, playerID)
	if err != nil {
		return
	}

	subject := fmt.Sprintf("[Draftr] ✅ %s is back", p.Name)
	html := fmt.Sprintf(`
<p>Hi %s,</p>
<p><strong>%s</strong> (%s) has returned to game action and has been automatically moved back to <strong>active</strong> on your roster.</p>
<p>Any substitute slot has been cleared.</p>
<p>— Draftr</p>
`, owner.Manager, p.Name, p.NhlTeam)

	targets := buildTargets(st, leagueID, owner)
	for _, to := range targets {
		if err := mailer.Send(to, subject, html); err != nil {
			log.Printf("[email] injury resolved to %s: %v", to, err)
		}
	}
}

// buildTargets returns deduplicated email addresses: the affected manager +
// the commissioner (if different and if they have an email on file).
func buildTargets(st *store.Store, leagueID int, owner *store.TeamEmailInfo) []string {
	seen := map[string]bool{}
	var out []string
	if owner.Email != "" {
		seen[owner.Email] = true
		out = append(out, owner.Email)
	}
	// Commissioner = lowest-ID team in this league.
	teams, _ := st.GetLeagueTeamEmails(leagueID)
	if len(teams) > 0 {
		comm := teams[0]
		if comm.Email != "" && !seen[comm.Email] {
			out = append(out, comm.Email)
		}
	}
	return out
}
