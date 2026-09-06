package jobs

import (
	"log"
	"time"

	"hockey_season/backend/store"
)

// DigestInjuries queries injury-sub transactions from the last 24 hours and
// logs a summary (email delivery is deferred to a future implementation).
func DigestInjuries(st *store.Store) {
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

	type entry struct {
		teamID       int
		droppedName  string
		addedName    string
		createdAt    string
	}

	var entries []entry
	for rows.Next() {
		var e entry
		var dropID, addID int
		if err := rows.Scan(&e.teamID, &dropID, &addID, &e.createdAt, &e.droppedName, &e.addedName); err != nil {
			log.Printf("[injury_digest] scan: %v", err)
			continue
		}
		entries = append(entries, e)
	}

	if len(entries) == 0 {
		log.Printf("[injury_digest] no injury sub activity in last 24h — skipping digest")
		return
	}

	log.Printf("[injury_digest] %d injury sub moves since %v:", len(entries), since.Format("2006-01-02 15:04"))
	for _, e := range entries {
		log.Printf("  [team %d] subbed out %s → in %s at %s", e.teamID, e.droppedName, e.addedName, e.createdAt)
	}
	log.Printf("[email deferred] would send injury digest to all league members")
}
