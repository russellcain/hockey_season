// Package mockdata seeds and tears down the development draft state.
// The seeded state mirrors frontend/src/data/mockDraft.ts:
//   - 8 fantasy teams, Hat Trick Heroes is "my team" (code: hat-trick-heroes-code)
//   - Draft session at round 3, pick 3 (Hat Trick Heroes is up next)
//   - Hat Trick Heroes: $77M cap used, 2 goalies drafted → tests over-cap + position-full
//
// Run the server with -mock to activate:
//   DRAFT_SECRET=dev-secret go run main.go -mock
//
// All rows inserted by Seed are removed by Cleanup on server shutdown.
package mockdata

import (
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"log"
)

// SeedResult holds the IDs of rows created by Seed so Cleanup can remove them.
type SeedResult struct {
	SessionID int
	TeamIDs   []int
}

type team struct {
	name    string
	manager string
	code    string // raw code entered by the drafter
	capUsed int64
}

// teams must be ordered so that index 0..7 maps to snake-order positions 0..7.
var teams = []team{
	{"Frozen Flames", "Alex K.", "frozen-flames-code", 36_000_000},
	{"Puck Norris", "Jamie T.", "puck-norris-code", 38_500_000},
	{"Hat Trick Heroes", "Sam R.", "hat-trick-heroes-code", 77_000_000}, // isMe
	{"Slapshot Squad", "Taylor M.", "slapshot-squad-code", 35_250_000},
	{"Ice Cold Cash", "Morgan B.", "ice-cold-cash-code", 40_000_000},
	{"Rink Rulers", "Jordan P.", "rink-rulers-code", 33_000_000},
	{"Zamboni Drivers", "Casey O.", "zamboni-drivers-code", 31_500_000},
	{"Five Hole Fellas", "Riley W.", "five-hole-fellas-code", 37_750_000},
}

// picks mirrors the non-null entries from AVAILABLE_PLAYERS in mockDraft.ts.
// teamIdx is 0-based index into the teams slice above.
type pick struct {
	teamIdx    int
	round      int
	pickNumber int
	playerName string
}

var picks = []pick{
	// Round 1 (odd → ascending)
	{0, 1, 1, "Auston Matthews"},
	{1, 1, 2, "David Pastrnak"},
	{2, 1, 3, "Leon Draisaitl"},
	{3, 1, 4, "Mikko Rantanen"},
	{4, 1, 5, "Brayden Point"},
	{5, 1, 6, "William Nylander"},
	{6, 1, 7, "Sebastian Aho"},
	{7, 1, 8, "Nathan MacKinnon"},
	// Round 2 (even → descending)
	{7, 2, 1, "Nico Hischier"},
	{6, 2, 2, "Matthew Tkachuk"},
	{5, 2, 3, "Miro Heiskanen"},
	{4, 2, 4, "Elias Pettersson"},
	{3, 2, 5, "Rasmus Dahlin"},
	{2, 2, 6, "Quinn Hughes"},
	{1, 2, 7, "Roman Josi"},
	{0, 2, 8, "Cale Makar"},
	// Round 3 (odd → ascending) — all 8 pre-seeded; session state = round 3 pick 3 "next"
	{0, 3, 1, "Igor Shesterkin"},
	{1, 3, 2, "Andrei Vasilevskiy"},
	{2, 3, 3, "Frederik Andersen"},
	{3, 3, 4, "Juuse Saros"},
	{4, 3, 5, "Thatcher Demko"},
	{5, 3, 6, "Brady Tkachuk"},
	{6, 3, 7, "Thomas Chabot"},
	{7, 3, 8, "Tim Stützle"},
	// Round 4 (even → descending; Rink Rulers and Zamboni Drivers skipped per mock)
	{7, 4, 1, "J.T. Miller"},
	{4, 4, 4, "Roope Hintz"},
	{3, 4, 5, "John Carlson"},
	{2, 4, 6, "Alex Pietrangelo"},
	{1, 4, 7, "Viktor Arvidsson"},
	{0, 4, 8, "Drew Doughty"},
	// Round 5 (odd → ascending)
	{0, 5, 1, "Mark Scheifele"},
	{1, 5, 2, "Gabriel Landeskog"},
	{2, 5, 3, "Sam Reinhart"},
	// Round 6 — Hat Trick Heroes fills 2nd goalie slot (pick 6 in descending round)
	{2, 6, 6, "Jake Oettinger"},
}

// Seed inserts all mock data and returns the created IDs for later cleanup.
// It is safe to call multiple times — existing rows are skipped via INSERT OR IGNORE
// on unique constraints, but a new draft_sessions row is always inserted.
func Seed(db *sql.DB, secret string) (*SeedResult, error) {
	teamIDs, err := seedTeams(db, secret)
	if err != nil {
		return nil, fmt.Errorf("seed teams: %w", err)
	}

	sessionID, err := seedSession(db)
	if err != nil {
		return nil, fmt.Errorf("seed session: %w", err)
	}

	if err := seedPicks(db, sessionID, teamIDs); err != nil {
		return nil, fmt.Errorf("seed picks: %w", err)
	}

	log.Printf("[mock] session %d ready — Hat Trick Heroes code: hat-trick-heroes-code", sessionID)
	log.Println("[mock] all team codes: <team-name>-code  (e.g. frozen-flames-code)")
	return &SeedResult{SessionID: sessionID, TeamIDs: teamIDs}, nil
}

// Cleanup removes all rows inserted by Seed. Called on server shutdown.
func Cleanup(db *sql.DB, result *SeedResult) {
	if result == nil {
		return
	}
	if _, err := db.Exec(`DELETE FROM draft_picks WHERE session_id = ?`, result.SessionID); err != nil {
		log.Printf("[mock] cleanup picks: %v", err)
	}
	if _, err := db.Exec(`DELETE FROM draft_sessions WHERE id = ?`, result.SessionID); err != nil {
		log.Printf("[mock] cleanup session: %v", err)
	}
	for _, id := range result.TeamIDs {
		if _, err := db.Exec(`DELETE FROM fantasy_teams WHERE id = ?`, id); err != nil {
			log.Printf("[mock] cleanup team %d: %v", id, err)
		}
	}
	log.Printf("[mock] cleaned up session %d", result.SessionID)
}

func seedTeams(db *sql.DB, secret string) ([]int, error) {
	ids := make([]int, len(teams))
	for i, t := range teams {
		h := codeHash(t.code, secret)
		_, err := db.Exec(
			`INSERT OR IGNORE INTO fantasy_teams (name, manager, code_hash, cap_used) VALUES (?,?,?,?)`,
			t.name, t.manager, h, t.capUsed,
		)
		if err != nil {
			return nil, fmt.Errorf("insert team %s: %w", t.name, err)
		}
		if err := db.QueryRow(`SELECT id FROM fantasy_teams WHERE code_hash = ?`, h).Scan(&ids[i]); err != nil {
			return nil, fmt.Errorf("read team id %s: %w", t.name, err)
		}
	}
	return ids, nil
}

func seedSession(db *sql.DB) (int, error) {
	res, err := db.Exec(
		`INSERT INTO draft_sessions (status, total_rounds, total_teams, current_round, current_pick, seconds_per_pick, cap_limit)
		 VALUES ('in_progress', 15, 8, 3, 3, 90, 82500000)`,
	)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	return int(id), err
}

func seedPicks(db *sql.DB, sessionID int, teamIDs []int) error {
	for _, p := range picks {
		teamID := teamIDs[p.teamIdx]

		var playerID int
		err := db.QueryRow(`SELECT id FROM nhl_players WHERE name = ?`, p.playerName).Scan(&playerID)
		if err != nil {
			log.Printf("[mock] player not found in nhl_players, skipping pick: %q", p.playerName)
			continue
		}

		_, err = db.Exec(
			`INSERT OR IGNORE INTO draft_picks (session_id, team_id, player_id, round, pick_number) VALUES (?,?,?,?,?)`,
			sessionID, teamID, playerID, p.round, p.pickNumber,
		)
		if err != nil {
			return fmt.Errorf("insert pick %s r%d: %w", p.playerName, p.round, err)
		}
	}
	return nil
}

func codeHash(code, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(code))
	return hex.EncodeToString(mac.Sum(nil))
}
