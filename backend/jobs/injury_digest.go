package jobs

import (
	"fmt"
	"log"
	"strings"
	"time"

	"hockey_season/backend/email"
	"hockey_season/backend/store"
)

// DigestInjuries is the nightly 11PM job.  It finds injury-sub transactions
// from the past 24 hours, builds a digest, and emails it to the commissioner.
func DigestInjuries(st *store.Store, mailer email.Sender) {
	log.Println("[injury_digest] starting daily injury digest")
	start := time.Now()
	since := start.Add(-24 * time.Hour)

	rows, err := st.DB().Query(`
		SELECT t.team_id, t.dropped_player_id, t.added_player_id, t.created_at,
		       dp.name, ap.name
		FROM transactions t
		JOIN nhl_players dp ON dp.id = t.dropped_player_id
		JOIN nhl_players ap ON ap.id = t.added_player_id
		WHERE t.txn_type = 'injury_sub'
		  AND t.created_at >= ?
		ORDER BY t.created_at
	`, since.Format("2006-01-02T15:04:05"))
	if err != nil {
		log.Printf("[injury_digest] query: %v", err)
		return
	}
	defer rows.Close()

	var entries []digestEntry
	for rows.Next() {
		var e digestEntry
		var dropID, addID int
		if err := rows.Scan(&e.teamID, &dropID, &addID, &e.createdAt, &e.droppedName, &e.addedName); err != nil {
			log.Printf("[injury_digest] scan: %v", err)
			continue
		}
		entries = append(entries, e)
	}

	if len(entries) == 0 {
		log.Println("[injury_digest] no injury activity in last 24h")
		return
	}

	log.Printf("[injury_digest] %d injury-sub moves in last 24h — sending digest", len(entries))

	// Find all leagues that had activity and email their commissioners.
	leaguesSeen := map[int]bool{}
	for _, e := range entries {
		// Determine which league this team belongs to.
		var lid int
		st.DB().QueryRow(
			`SELECT COALESCE(league_id, 0) FROM fantasy_teams WHERE id = ?`, e.teamID,
		).Scan(&lid)
		if lid == 0 || leaguesSeen[lid] {
			continue
		}
		leaguesSeen[lid] = true
		sendDigestForLeague(st, mailer, lid, entries, since)
	}
}

type digestEntry struct {
	teamID      int
	droppedName string
	addedName   string
	createdAt   string
}

func sendDigestForLeague(st *store.Store, mailer email.Sender, leagueID int, entries []digestEntry, since time.Time) {
	teams, err := st.GetLeagueTeamEmails(leagueID)
	if err != nil || len(teams) == 0 {
		return
	}
	comm := teams[0]
	if comm.Email == "" {
		log.Printf("[injury_digest] league %d commissioner has no email set — skipping", leagueID)
		return
	}

	var rows []string
	for _, e := range entries {
		// Only include entries that belong to this league.
		var lid int
		st.DB().QueryRow(`SELECT COALESCE(league_id,0) FROM fantasy_teams WHERE id = ?`, e.teamID).Scan(&lid)
		if lid != leagueID {
			continue
		}
		var teamName string
		st.DB().QueryRow(`SELECT name FROM fantasy_teams WHERE id = ?`, e.teamID).Scan(&teamName)
		rows = append(rows, fmt.Sprintf(
			"<tr><td>%s</td><td>%s</td><td>%s</td><td>%s</td></tr>",
			teamName, e.droppedName, e.addedName, e.createdAt[:16],
		))
	}
	if len(rows) == 0 {
		return
	}

	html := fmt.Sprintf(`
<h2>Draftr — Daily Injury Digest</h2>
<p>%d injury substitution(s) since %s:</p>
<table border="1" cellpadding="6" style="border-collapse:collapse">
  <thead><tr><th>Team</th><th>Injured (out)</th><th>Sub (in)</th><th>Time</th></tr></thead>
  <tbody>%s</tbody>
</table>
<p>— Draftr</p>
`, len(rows), since.Format("Jan 2, 3:04 PM"), strings.Join(rows, ""))

	subject := fmt.Sprintf("[Draftr] Injury digest — %d move(s) today", len(rows))
	if err := mailer.Send(comm.Email, subject, html); err != nil {
		log.Printf("[injury_digest] send to %s: %v", comm.Email, err)
	} else {
		log.Printf("[injury_digest] digest sent to %s", comm.Email)
	}
}
