// Package main implements a full-season simulation for the fantasy hockey league.
// It creates a league, drafts 8 teams via a deterministic snake draft, seeds
// game logs for the entire 2025-26 NHL season, scores all matchup weeks, runs
// correctness assertions, and prints standings.
//
// Usage:
//
//	go run ./cmd/simulate [flags]
//	  -db          path to simulation SQLite file  (default: ../data/sim.db)
//	  -migrations  path to migrations directory    (default: ../data/migrations)
//	  -reset       delete the DB file and start fresh
package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strings"
	"time"

	"hockey_season/backend/snake"
	"hockey_season/backend/store"
)

// ── Constants ────────────────────────────────────────────────────────────────

const (
	numTeams   = 8
	numRounds  = 20
	capLimit   = 104_000_000
	seasonYear = 2025
)

var slotTargets = map[string]int{"F": 12, "D": 6, "G": 2}

var teamNames = [numTeams]string{
	"Ice Breakers", "Puck Savages", "Frozen Assets", "Hat Trick Ponies",
	"Bender Patrol", "Slap Shot Kings", "Penalty Box Heroes", "The Waiver Wire",
}

// 2025-26 NHL regular season window.
var (
	nhlSeasonStart = time.Date(2025, time.October, 7, 0, 0, 0, 0, time.UTC)
	nhlSeasonEnd   = time.Date(2026, time.April, 18, 0, 0, 0, 0, time.UTC)
)

// ── Entry point ──────────────────────────────────────────────────────────────

func main() {
	dbPath := flag.String("db", "../data/sim.db", "path to simulation SQLite file")
	migrationsDir := flag.String("migrations", "../data/migrations", "path to migrations directory")
	playersJSON := flag.String("players", "../data/player-salaries.json", "path to player-salaries.json")
	reset := flag.Bool("reset", false, "delete the simulation DB and start fresh")
	draftOnly := flag.Bool("draft-only", false, "stop after draft+rosters; skip schedule generation and game log seeding")
	flag.Parse()

	if *reset {
		if err := os.Remove(*dbPath); err != nil && !os.IsNotExist(err) {
			log.Fatalf("reset: remove %s: %v", *dbPath, err)
		}
		fmt.Printf("✓ Removed %s\n", *dbPath)
	}

	st, err := store.Open(*dbPath, *migrationsDir)
	if err != nil {
		log.Fatalf("open store: %v", err)
	}
	defer st.Close()

	// 1. Seed NHL players from JSON (idempotent — INSERT OR IGNORE)
	playerCount, err := seedPlayers(st.DB(), *playersJSON)
	if err != nil {
		log.Fatalf("seed players: %v", err)
	}
	fmt.Printf("✓ %d NHL players available\n", playerCount)

	// 2. League
	league, err := st.CreateLeagueForYear("2025-26 Simulation", capLimit, seasonYear)
	if err != nil {
		log.Fatalf("create league: %v", err)
	}
	fmt.Printf("✓ League #%d created (season_year=%d)\n", league.ID, league.SeasonYear)

	// 3. Teams
	teamIDs, err := createTeams(st.DB(), league.ID)
	if err != nil {
		log.Fatalf("create teams: %v", err)
	}
	fmt.Printf("✓ %d teams created\n", len(teamIDs))

	// 4. Draft session
	sessionID, err := createDraftSession(st.DB(), league.ID)
	if err != nil {
		log.Fatalf("create draft session: %v", err)
	}
	fmt.Printf("✓ Draft session #%d created\n", sessionID)

	// 5. Snake draft
	if err := autoDraft(st.DB(), sessionID, teamIDs); err != nil {
		log.Fatalf("auto-draft: %v", err)
	}
	fmt.Printf("✓ Snake draft complete (%d rounds × %d teams = %d picks)\n",
		numRounds, numTeams, numRounds*numTeams)

	if *draftOnly {
		// Populate roster_slots only — leave schedule generation and status
		// advancement for the admin UI so you can test that flow manually.
		if _, err := st.DB().Exec(`
			INSERT OR IGNORE INTO roster_slots (team_id, player_id, league_id, slot_type)
			SELECT dp.team_id, dp.player_id, ?, 'active'
			FROM draft_picks dp
			WHERE dp.session_id = ?
		`, league.ID, sessionID); err != nil {
			log.Fatalf("populate roster_slots: %v", err)
		}
		// Advance to draft_ready so the admin UI "Generate Schedule" button is reachable.
		if err := st.UpdateLeagueStatus(league.ID, "draft_ready"); err != nil {
			log.Fatalf("update league status: %v", err)
		}
		fmt.Println("✓ Roster slots populated (draft_only mode)")
		fmt.Printf("\nLeague ID: %d   Draft session ID: %d\n", league.ID, sessionID)
		fmt.Println("Start the backend, then use devtoken to log in and hit Admin → Generate Schedule.")
		return
	}

	// 6. Finalise: populate roster_slots + H2H schedule + advance league status
	if err := st.FinaliseDraft(sessionID); err != nil {
		log.Fatalf("finalise draft: %v", err)
	}
	fmt.Println("✓ Roster slots populated and H2H schedule generated")

	// 7. Seed game logs for the full season
	logCount, err := seedGameLogs(st, league.ID)
	if err != nil {
		log.Fatalf("seed game logs: %v", err)
	}
	fmt.Printf("✓ %d game log entries seeded (Oct 7 2025 – Apr 18 2026)\n", logCount)

	// 8. Score all matchup weeks
	maxWeek, err := scoreAllWeeks(st, league.ID)
	if err != nil {
		log.Fatalf("score weeks: %v", err)
	}
	fmt.Printf("✓ %d weeks scored\n", maxWeek)

	// 9. Verify correctness on sampled weeks
	failures := verifyWeeks(st, league.ID, maxWeek)
	if failures > 0 {
		fmt.Printf("\n✗ %d verification failure(s) — exit 1\n", failures)
		os.Exit(1)
	}
	fmt.Println("✓ All assertions passed")

	// 10. Print standings
	printStandings(st, league.ID)
}

// ── Setup helpers ─────────────────────────────────────────────────────────────

// nhlTeamCodes maps full franchise names to the standard 3-letter code.
var nhlTeamCodes = map[string]string{
	"Anaheim Ducks": "ANA", "Boston Bruins": "BOS", "Buffalo Sabres": "BUF",
	"Calgary Flames": "CGY", "Carolina Hurricanes": "CAR", "Chicago Blackhawks": "CHI",
	"Colorado Avalanche": "COL", "Columbus Blue Jackets": "CBJ", "Dallas Stars": "DAL",
	"Detroit Red Wings": "DET", "Edmonton Oilers": "EDM", "Florida Panthers": "FLA",
	"Los Angeles Kings": "LAK", "Minnesota Wild": "MIN", "Montréal Canadiens": "MTL",
	"Montreal Canadiens": "MTL", "Nashville Predators": "NSH", "New Jersey Devils": "NJD",
	"New York Islanders": "NYI", "New York Rangers": "NYR", "Ottawa Senators": "OTT",
	"Philadelphia Flyers": "PHI", "Pittsburgh Penguins": "PIT", "San Jose Sharks": "SJS",
	"Seattle Kraken": "SEA", "St. Louis Blues": "STL", "Tampa Bay Lightning": "TBL",
	"Toronto Maple Leafs": "TOR", "Utah Hockey Club": "UTA", "Vancouver Canucks": "VAN",
	"Vegas Golden Knights": "VGK", "Washington Capitals": "WSH", "Winnipeg Jets": "WPG",
	"Arizona Coyotes": "ARI",
}

func teamCode(name string) string {
	if code, ok := nhlTeamCodes[name]; ok {
		return code
	}
	// Fallback: uppercase initials of each word.
	var b strings.Builder
	for _, w := range strings.Fields(name) {
		if len(w) > 0 {
			b.WriteByte(w[0])
		}
	}
	return strings.ToUpper(b.String())
}

// seedPlayers loads player-salaries.json into nhl_players. Safe to call on an
// already-populated DB — INSERT OR IGNORE skips duplicates on (name, nhl_team_code).
func seedPlayers(db *sql.DB, jsonPath string) (int, error) {
	f, err := os.Open(jsonPath)
	if err != nil {
		return 0, fmt.Errorf("open %s: %w", jsonPath, err)
	}
	defer f.Close()

	var raw []struct {
		Name         string `json:"name"`
		Age          string `json:"age"`
		Pos          string `json:"pos"`
		SalaryCapHit string `json:"salary_cap_hit"`
		Team         string `json:"team"`
	}
	if err := json.NewDecoder(f).Decode(&raw); err != nil {
		return 0, fmt.Errorf("decode JSON: %w", err)
	}

	tx, err := db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT OR IGNORE INTO nhl_players (name, nhl_team_name, nhl_team_code, position, salary_cap_hit, age)
		VALUES (?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return 0, err
	}
	defer stmt.Close()

	for _, p := range raw {
		if _, err := stmt.Exec(p.Name, p.Team, teamCode(p.Team), p.Pos, p.SalaryCapHit, p.Age); err != nil {
			return 0, fmt.Errorf("insert %s: %w", p.Name, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}

	var count int
	db.QueryRow(`SELECT COUNT(*) FROM nhl_players`).Scan(&count)
	return count, nil
}

func createTeams(db *sql.DB, leagueID int) ([]int, error) {
	ids := make([]int, 0, numTeams)
	for i, name := range teamNames {
		codeHash := fmt.Sprintf("sim_%d_team_%d", leagueID, i)
		res, err := db.Exec(`
			INSERT INTO fantasy_teams (name, manager, code_hash, cap_used, league_id)
			VALUES (?, ?, ?, 0, ?)
		`, name, fmt.Sprintf("Manager %d", i+1), codeHash, leagueID)
		if err != nil {
			return nil, fmt.Errorf("insert team %q: %w", name, err)
		}
		id, _ := res.LastInsertId()
		ids = append(ids, int(id))
	}
	return ids, nil
}

func createDraftSession(db *sql.DB, leagueID int) (int, error) {
	res, err := db.Exec(`
		INSERT INTO draft_sessions
		  (status, total_rounds, total_teams, current_round, current_pick, seconds_per_pick, cap_limit, league_id)
		VALUES ('in_progress', ?, ?, 1, 1, 90, ?, ?)
	`, numRounds, numTeams, capLimit, leagueID)
	if err != nil {
		return 0, err
	}
	id, _ := res.LastInsertId()
	return int(id), nil
}

// ── Auto-draft ────────────────────────────────────────────────────────────────

type simPlayer struct {
	id       int
	position string
	salary   int64
}

// minPickSalary is the per-pick cap reservation for future rounds.
// Set to the NHL minimum salary so teams always keep enough room to
// complete the draft. The JSON data includes $100K AHL/development
// contracts which would otherwise wreck the reservation math.
const minPickSalary int64 = 775_000

// autoDraft executes a greedy snake draft: each team picks the highest-salary
// available player that fits their open position slots and remaining cap space,
// while reserving enough budget for all remaining picks at minimum salary.
func autoDraft(db *sql.DB, sessionID int, teamIDs []int) error {
	// Load only players on realistic NHL contracts (salary >= NHL minimum).
	// Excludes ~100K AHL/development entries that would corrupt cap math.
	rows, err := db.Query(`
		SELECT id, position,
		       CAST(REPLACE(salary_cap_hit, ',', '') AS INTEGER) AS sal
		FROM nhl_players
		WHERE CAST(REPLACE(salary_cap_hit, ',', '') AS INTEGER) >= ?
		ORDER BY sal DESC, id ASC
	`, minPickSalary)
	if err != nil {
		return fmt.Errorf("load players: %w", err)
	}
	defer rows.Close()

	var players []simPlayer
	for rows.Next() {
		var p simPlayer
		if err := rows.Scan(&p.id, &p.position, &p.salary); err != nil {
			return err
		}
		players = append(players, p)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	log.Printf("[sim] %d eligible players loaded for draft", len(players))

	// Per-team state.
	posCount := make(map[int]map[string]int, numTeams)
	capUsed := make(map[int]int64, numTeams)
	for _, tid := range teamIDs {
		posCount[tid] = map[string]int{"F": 0, "D": 0, "G": 0}
	}
	drafted := make(map[int]bool, numTeams*numRounds)

	for round := 1; round <= numRounds; round++ {
		// Picks each team still needs to make after the current round.
		picksAfter := int64(numRounds - round)

		for pickIdx, teamIdx := range snake.Order(round, numTeams) {
			tid := teamIDs[teamIdx]
			pickInRound := pickIdx + 1

			found := false
			for i := range players {
				p := &players[i]
				if drafted[p.id] {
					continue
				}
				if posCount[tid][p.position] >= slotTargets[p.position] {
					continue
				}
				// Must still be able to afford all remaining picks at minimum salary.
				if capUsed[tid]+p.salary+minPickSalary*picksAfter > capLimit {
					continue
				}

				if _, err := db.Exec(`
					INSERT INTO draft_picks (session_id, team_id, player_id, round, pick_number)
					VALUES (?, ?, ?, ?, ?)
				`, sessionID, tid, p.id, round, pickInRound); err != nil {
					return fmt.Errorf("insert pick r%d p%d: %w", round, pickInRound, err)
				}

				drafted[p.id] = true
				posCount[tid][p.position]++
				capUsed[tid] += p.salary
				found = true
				break
			}
			if !found {
				// Find the first unfilled position slot for this team.
				needPos := ""
				for _, pos := range []string{"F", "D", "G"} {
					if posCount[tid][pos] < slotTargets[pos] {
						needPos = pos
						break
					}
				}
				if needPos == "" {
					// All slots are somehow full — nothing to do this pick.
					log.Printf("[sim] ⚠ team %d round %d: all slots full, skipping pick", tid, round)
					continue
				}

				// Insert a unique placeholder row in nhl_players so foreign-key
				// constraints are satisfied and every pick has a distinct player_id.
				const defaultSalary int64 = 975_000
				res, err := db.Exec(`
					INSERT INTO nhl_players (name, nhl_team_name, nhl_team_code, position, salary_cap_hit, age)
					VALUES (?, 'TBD', 'TBD', ?, '975,000', '0')
				`, fmt.Sprintf("DEFAULT %s (team %d rnd %d)", needPos, tid, round), needPos)
				if err != nil {
					return fmt.Errorf("insert default player r%d t%d: %w", round, tid, err)
				}
				defaultID64, _ := res.LastInsertId()
				defaultID := int(defaultID64)

				if _, err := db.Exec(`
					INSERT INTO draft_picks (session_id, team_id, player_id, round, pick_number)
					VALUES (?, ?, ?, ?, ?)
				`, sessionID, tid, defaultID, round, pickInRound); err != nil {
					return fmt.Errorf("insert default pick r%d p%d: %w", round, pickInRound, err)
				}

				drafted[defaultID] = true
				posCount[tid][needPos]++
				capUsed[tid] += defaultSalary
				log.Printf("[sim] ⚠ team %d round %d: no valid %s pick — inserted DEFAULT placeholder (id=%d, capUsed=$%dm)",
					tid, round, needPos, defaultID, capUsed[tid]/1_000_000)
			}
		}
	}

	// Mark draft complete.
	if _, err := db.Exec(`
		UPDATE draft_sessions SET status = 'complete', current_round = ?, current_pick = 1
		WHERE id = ?
	`, numRounds+1, sessionID); err != nil {
		return err
	}

	// Write final cap used to each team.
	for _, tid := range teamIDs {
		if _, err := db.Exec(
			`UPDATE fantasy_teams SET cap_used = ? WHERE id = ?`, capUsed[tid], tid,
		); err != nil {
			return err
		}
	}

	return nil
}

// ── Game log seeding ──────────────────────────────────────────────────────────

type rosterEntry struct {
	playerID int
	position string
	salary   int64
}

// seedGameLogs generates deterministic stats for every rostered player for
// every day of the 2025-26 NHL season. The RNG seed is derived from playerID
// and day index, so results are fully reproducible.
func seedGameLogs(st *store.Store, leagueID int) (int, error) {
	rows, err := st.DB().Query(`
		SELECT rs.player_id,
		       np.position,
		       CAST(REPLACE(np.salary_cap_hit, ',', '') AS INTEGER) AS sal
		FROM roster_slots rs
		JOIN nhl_players np ON np.id = rs.player_id
		WHERE rs.league_id = ?
	`, leagueID)
	if err != nil {
		return 0, fmt.Errorf("load roster: %w", err)
	}
	defer rows.Close()

	var roster []rosterEntry
	for rows.Next() {
		var e rosterEntry
		if err := rows.Scan(&e.playerID, &e.position, &e.salary); err != nil {
			return 0, err
		}
		roster = append(roster, e)
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}

	totalDays := int(nhlSeasonEnd.Sub(nhlSeasonStart).Hours()/24) + 1
	inserted := 0

	for dayIdx := 0; dayIdx < totalDays; dayIdx++ {
		date := nhlSeasonStart.AddDate(0, 0, dayIdx).Format("2006-01-02")

		for _, e := range roster {
			// Deterministic per-player-per-day seed.
			rng := rand.New(rand.NewSource(int64(e.playerID)*100_000 + int64(dayIdx)))

			var goals, assists, wins, otl, shutouts int
			if e.position == "G" {
				wins, otl, shutouts = goalieStats(rng, e.salary)
			} else {
				goals, assists = skaterStats(rng, e.salary)
			}

			if goals+assists+wins+otl+shutouts == 0 {
				continue // no game or nothing scored — skip
			}
			if err := st.UpsertGameLog(e.playerID, date, goals, assists, wins, otl, shutouts); err != nil {
				return inserted, fmt.Errorf("upsert log player=%d date=%s: %w", e.playerID, date, err)
			}
			inserted++
		}
	}
	return inserted, nil
}

// skaterStats returns (goals, assists) for a skater on a given day.
// P(game) ≈ 0.42 (82 games / 194 days). Rates scale with salary.
func skaterStats(rng *rand.Rand, salary int64) (goals, assists int) {
	if rng.Float64() > 0.42 {
		return 0, 0 // no game today
	}
	var gRate, aRate float64
	switch {
	case salary >= 8_000_000:
		gRate, aRate = 0.22, 0.38
	case salary >= 6_000_000:
		gRate, aRate = 0.16, 0.30
	case salary >= 4_000_000:
		gRate, aRate = 0.11, 0.22
	case salary >= 2_000_000:
		gRate, aRate = 0.07, 0.15
	default:
		gRate, aRate = 0.04, 0.10
	}
	if rng.Float64() < gRate {
		goals = 1
	}
	if rng.Float64() < gRate*0.4 {
		goals++ // multi-goal game
	}
	if rng.Float64() < aRate {
		assists = 1
	}
	if rng.Float64() < aRate*0.4 {
		assists++
	}
	return goals, assists
}

// goalieStats returns (wins, otl, shutouts) for a goalie on a given day.
// P(start) ≈ 0.22 (split between two goalies over 82 games).
func goalieStats(rng *rand.Rand, salary int64) (wins, otl, shutouts int) {
	if rng.Float64() > 0.22 {
		return 0, 0, 0 // didn't start
	}
	var wRate, oRate, soRate float64
	switch {
	case salary >= 5_000_000:
		wRate, oRate, soRate = 0.58, 0.14, 0.08
	case salary >= 3_000_000:
		wRate, oRate, soRate = 0.52, 0.14, 0.05
	default:
		wRate, oRate, soRate = 0.44, 0.14, 0.03
	}
	r := rng.Float64()
	if r < wRate {
		wins = 1
		if rng.Float64() < soRate {
			shutouts = 1
		}
	} else if r < wRate+oRate {
		otl = 1
	}
	// plain loss: no scoreable stats
	return wins, otl, shutouts
}

// ── Scoring ───────────────────────────────────────────────────────────────────

func scoreAllWeeks(st *store.Store, leagueID int) (int, error) {
	var maxWeek int
	if err := st.DB().QueryRow(
		`SELECT COALESCE(MAX(week_number), 0) FROM matchups WHERE league_id = ?`, leagueID,
	).Scan(&maxWeek); err != nil {
		return 0, err
	}
	for w := 1; w <= maxWeek; w++ {
		if err := st.UpdateMatchupScores(leagueID, w); err != nil {
			return w - 1, fmt.Errorf("week %d: %w", w, err)
		}
	}
	return maxWeek, nil
}

// ── Verification ──────────────────────────────────────────────────────────────

// verifyWeeks checks 3 sampled weeks: beginning, middle, and last.
// For each matchup it asserts that TeamScore() matches the stored scores and
// that home_points + away_points == 2.
func verifyWeeks(st *store.Store, leagueID, maxWeek int) int {
	checkWeeks := []int{1, maxWeek / 2, maxWeek}
	// deduplicate (e.g. when maxWeek is very small)
	seen := map[int]bool{}
	weeks := checkWeeks[:0]
	for _, w := range checkWeeks {
		if w > 0 && !seen[w] {
			weeks = append(weeks, w)
			seen[w] = true
		}
	}

	seasonStart := firstMondayOfOctober(seasonYear)
	failures := 0

	for _, w := range weeks {
		matchups, err := st.GetWeekMatchups(leagueID, w)
		if err != nil {
			fmt.Printf("  ✗ week %d: GetWeekMatchups: %v\n", w, err)
			failures++
			continue
		}
		if len(matchups) == 0 {
			fmt.Printf("  ✗ week %d: no matchups\n", w)
			failures++
			continue
		}

		weekStart := seasonStart.AddDate(0, 0, (w-1)*7)
		weekEnd := weekStart.AddDate(0, 0, 6)

		weekFail := 0
		for _, m := range matchups {
			homeScore, err := st.TeamScore(leagueID, m.HomeTeamID, weekStart, weekEnd)
			if err != nil {
				fmt.Printf("  ✗ week %d matchup %d: home TeamScore: %v\n", w, m.ID, err)
				weekFail++
				continue
			}
			awayScore, err := st.TeamScore(leagueID, m.AwayTeamID, weekStart, weekEnd)
			if err != nil {
				fmt.Printf("  ✗ week %d matchup %d: away TeamScore: %v\n", w, m.ID, err)
				weekFail++
				continue
			}

			if absf(homeScore.Total-m.HomeScore) > 0.01 {
				fmt.Printf("  ✗ week %d matchup %d: home score %.1f ≠ stored %.1f\n",
					w, m.ID, homeScore.Total, m.HomeScore)
				weekFail++
			}
			if absf(awayScore.Total-m.AwayScore) > 0.01 {
				fmt.Printf("  ✗ week %d matchup %d: away score %.1f ≠ stored %.1f\n",
					w, m.ID, awayScore.Total, m.AwayScore)
				weekFail++
			}
			if pts := m.HomePoints + m.AwayPoints; pts != 2 {
				fmt.Printf("  ✗ week %d matchup %d: points sum = %d (want 2)\n", w, m.ID, pts)
				weekFail++
			}
		}
		if weekFail == 0 {
			fmt.Printf("  ✓ week %2d: %d matchups OK\n", w, len(matchups))
		}
		failures += weekFail
	}
	return failures
}

func absf(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// firstMondayOfOctober mirrors the unexported function in store/matchup.go.
func firstMondayOfOctober(year int) time.Time {
	oct1 := time.Date(year, time.October, 1, 0, 0, 0, 0, time.UTC)
	dow := int(oct1.Weekday())
	daysUntilMonday := (8 - dow) % 7
	if dow == 1 {
		daysUntilMonday = 0
	}
	return oct1.AddDate(0, 0, daysUntilMonday)
}

// ── Standings ─────────────────────────────────────────────────────────────────

func printStandings(st *store.Store, leagueID int) {
	agg, h2h, err := st.GetStandings(leagueID)
	if err != nil {
		fmt.Printf("\nstandings error: %v\n", err)
		return
	}

	fmt.Println("\n╔══ AGGREGATE STANDINGS (total fantasy points) ═══════════════╗")
	fmt.Printf("  %-4s  %-22s  %7s  %5s\n", "Rank", "Team", "Pts", "Goals")
	for i, ts := range agg {
		fmt.Printf("  %-4d  %-22s  %7.1f  %5d\n", i+1, ts.Team.Name, ts.TotalPoints, ts.Goals)
	}

	fmt.Println("\n╔══ H2H STANDINGS ════════════════════════════════════════════╗")
	fmt.Printf("  %-4s  %-22s  %5s  %4s  %4s  %4s\n", "Rank", "Team", "H2HPt", "W", "T", "L")
	for i, ts := range h2h {
		fmt.Printf("  %-4d  %-22s  %5d  %4d  %4d  %4d\n",
			i+1, ts.Team.Name, ts.H2HPoints, ts.H2HWins, ts.H2HTies, ts.H2HLosses)
	}
	fmt.Println()
}
