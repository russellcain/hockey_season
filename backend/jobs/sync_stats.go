package jobs

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"hockey_season/backend/store"
)

// SyncStats fetches recent game logs from the NHL Stats API for all players
// with a known nhl_id, upserts them into player_game_logs, then updates
// consecutive_misses for injury detection.
func SyncStats(st *store.Store) {
	log.Println("[sync_stats] starting stats sync")
	start := time.Now()

	players, err := st.ListPlayers()
	if err != nil {
		log.Printf("[sync_stats] list players: %v", err)
		return
	}

	synced := 0
	for _, p := range players {
		if err := syncPlayerStats(st, p.ID); err != nil {
			log.Printf("[sync_stats] player %d (%s): %v", p.ID, p.Name, err)
		} else {
			synced++
		}
	}

	log.Printf("[sync_stats] done in %v — synced %d/%d players", time.Since(start), synced, len(players))
}

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

// nhlIDForPlayer retrieves the nhl_id for a player by their internal ID.
// Returns 0 if not set (we skip those players).
func nhlIDForPlayer(st *store.Store, playerID int) int {
	// The Store currently doesn't expose nhl_id; we call raw SQL via the exported DB.
	var nhlID int
	st.DB().QueryRow(`SELECT COALESCE(nhl_id, 0) FROM nhl_players WHERE id = ?`, playerID).Scan(&nhlID)
	return nhlID
}

func syncPlayerStats(st *store.Store, playerID int) error {
	nhlID := nhlIDForPlayer(st, playerID)
	if nhlID == 0 {
		return nil // skip players with no nhl_id
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

	for _, entry := range data.GameLog {
		if err := st.UpsertGameLog(playerID, entry.GameDate,
			entry.Goals, entry.Assists, entry.Wins, entry.OTLosses, entry.Shutouts); err != nil {
			log.Printf("[sync_stats] upsert log player %d date %s: %v", playerID, entry.GameDate, err)
		}
	}
	return nil
}
