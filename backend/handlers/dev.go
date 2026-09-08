package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"hockey_season/backend/store"
)

// nhlSimStart is the base date used by the simulator for dayIdx=0.
// Must match the hardcoded value in cmd/simulate/main.go so RNG seeds align.
var nhlSimStart = time.Date(2025, time.October, 7, 0, 0, 0, 0, time.UTC)

// DevHandler provides test-only endpoints that advance simulated game time.
// Register its routes only when the server is started with -dev.
type DevHandler struct {
	store *store.Store
}

func NewDev(s *store.Store) *DevHandler {
	return &DevHandler{store: s}
}

// ── Response types ────────────────────────────────────────────────────────────

type devMatchupResult struct {
	ID         int              `json:"id"`
	HomeTeam   string           `json:"homeTeam"`
	AwayTeam   string           `json:"awayTeam"`
	HomeScore  float64          `json:"homeScore"`
	AwayScore  float64          `json:"awayScore"`
	HomePoints int              `json:"homePoints"`
	AwayPoints int              `json:"awayPoints"`
	TopScorers []devTopScorer   `json:"topScorers"`
}

type devTopScorer struct {
	Name     string `json:"name"`
	Team     string `json:"team"`
	Position string `json:"position"`
	Points   int    `json:"points"` // fantasy pts this week
}

type devAdvanceResp struct {
	Week          int                `json:"week"`
	WeekStart     string             `json:"weekStart"`
	WeekEnd       string             `json:"weekEnd"`
	LogsSeeded    int                `json:"logsSeeded"`
	Matchups      []devMatchupResult `json:"matchups"`
	NewInjuries   []string           `json:"newInjuries"`
	ClearedInjuries []string         `json:"clearedInjuries"`
}

// ── Handler ───────────────────────────────────────────────────────────────────

// AdvanceWeek handles POST /api/dev/leagues/{id}/advance-week.
// It finds the next unscored week, seeds game logs for its date range using the
// same deterministic RNG as the simulator, scores the matchups, runs injury
// detection, and returns a full summary.
func (h *DevHandler) AdvanceWeek(w http.ResponseWriter, r *http.Request) {
	leagueID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	// 1. Determine next week to advance.
	weekNum, err := h.nextWeek(leagueID)
	if err != nil {
		http.Error(w, fmt.Sprintf("determine week: %v", err), http.StatusInternalServerError)
		return
	}
	if weekNum == 0 {
		http.Error(w, "no matchups scheduled for this league", http.StatusBadRequest)
		return
	}

	// 2. Compute week date range.
	var seasonYear int
	if err := h.store.DB().QueryRow(`SELECT season_year FROM leagues WHERE id = ?`, leagueID).Scan(&seasonYear); err != nil {
		http.Error(w, "get season_year", http.StatusInternalServerError)
		return
	}
	seasonStart := store.FirstMondayOfOctober(seasonYear)
	weekStart := seasonStart.AddDate(0, 0, (weekNum-1)*7)
	weekEnd := weekStart.AddDate(0, 0, 6)

	// 3. Seed game logs for every day in the week.
	logsSeeded, err := h.seedWeekLogs(leagueID, weekStart, weekEnd)
	if err != nil {
		http.Error(w, fmt.Sprintf("seed logs: %v", err), http.StatusInternalServerError)
		return
	}
	log.Printf("[dev] week %d: seeded %d game log entries (%s – %s)", weekNum, logsSeeded, weekStart.Format("Jan 2"), weekEnd.Format("Jan 2"))

	// 4. Score the matchups.
	if err := h.store.UpdateMatchupScores(leagueID, weekNum); err != nil {
		http.Error(w, fmt.Sprintf("score matchups: %v", err), http.StatusInternalServerError)
		return
	}

	// 5. Build matchup results with top scorers.
	matchups, err := h.store.GetWeekMatchups(leagueID, weekNum)
	if err != nil {
		http.Error(w, "get matchups", http.StatusInternalServerError)
		return
	}
	var results []devMatchupResult
	for _, m := range matchups {
		top := h.topScorers(leagueID, m.HomeTeamID, m.AwayTeamID, weekStart, weekEnd)
		results = append(results, devMatchupResult{
			ID:         m.ID,
			HomeTeam:   m.HomeTeam.Name,
			AwayTeam:   m.AwayTeam.Name,
			HomeScore:  m.HomeScore,
			AwayScore:  m.AwayScore,
			HomePoints: m.HomePoints,
			AwayPoints: m.AwayPoints,
			TopScorers: top,
		})
	}

	// 6. Injury detection.
	newInj, cleared := h.runInjuryDetection(leagueID, weekStart, weekEnd)

	resp := devAdvanceResp{
		Week:            weekNum,
		WeekStart:       weekStart.Format("2006-01-02"),
		WeekEnd:         weekEnd.Format("2006-01-02"),
		LogsSeeded:      logsSeeded,
		Matchups:        results,
		NewInjuries:     newInj,
		ClearedInjuries: cleared,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// ── Helpers ───────────────────────────────────────────────────────────────────

// nextWeek returns the lowest week_number in matchups that has home_score=0
// and away_score=0, i.e., not yet scored.
func (h *DevHandler) nextWeek(leagueID int) (int, error) {
	var week int
	err := h.store.DB().QueryRow(`
		SELECT COALESCE(MIN(week_number), 0)
		FROM matchups
		WHERE league_id = ? AND home_score = 0 AND away_score = 0
	`, leagueID).Scan(&week)
	return week, err
}

// seedWeekLogs seeds deterministic game logs for every rostered player for
// every day of [weekStart, weekEnd] using the same RNG as the simulator.
func (h *DevHandler) seedWeekLogs(leagueID int, weekStart, weekEnd time.Time) (int, error) {
	type entry struct {
		playerID int
		position string
		salary   int64
	}

	rows, err := h.store.DB().Query(`
		SELECT rs.player_id, np.position,
		       CAST(REPLACE(np.salary_cap_hit, ',', '') AS INTEGER)
		FROM roster_slots rs
		JOIN nhl_players np ON np.id = rs.player_id
		WHERE rs.league_id = ?
	`, leagueID)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	var roster []entry
	for rows.Next() {
		var e entry
		if err := rows.Scan(&e.playerID, &e.position, &e.salary); err != nil {
			return 0, err
		}
		roster = append(roster, e)
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}

	inserted := 0
	for d := weekStart; !d.After(weekEnd); d = d.AddDate(0, 0, 1) {
		dayIdx := int(d.Sub(nhlSimStart).Hours() / 24)
		date := d.Format("2006-01-02")

		for _, e := range roster {
			rng := rand.New(rand.NewSource(int64(e.playerID)*100_000 + int64(dayIdx)))

			var goals, assists, wins, otl, shutouts int
			if e.position == "G" {
				wins, otl, shutouts = devGoalieStats(rng, e.salary)
			} else {
				goals, assists = devSkaterStats(rng, e.salary)
			}
			if goals+assists+wins+otl+shutouts == 0 {
				continue
			}
			if err := h.store.UpsertGameLog(e.playerID, date, goals, assists, wins, otl, shutouts); err != nil {
				return inserted, err
			}
			inserted++
		}
	}
	return inserted, nil
}

// topScorers returns the top 3 fantasy scorers across both teams for the week.
func (h *DevHandler) topScorers(leagueID, homeTeamID, awayTeamID int, weekStart, weekEnd time.Time) []devTopScorer {
	from := weekStart.Format("2006-01-02")
	to := weekEnd.Format("2006-01-02")
	rows, err := h.store.DB().Query(`
		SELECT np.name, np.position, ft.name AS team_name,
		       COALESCE(SUM(CASE WHEN np.position != 'G' THEN gl.goals + gl.assists ELSE gl.wins*2 + gl.otl + gl.shutouts END), 0) AS pts
		FROM roster_slots rs
		JOIN nhl_players np ON np.id = rs.player_id
		JOIN fantasy_teams ft ON ft.id = rs.team_id
		LEFT JOIN player_game_logs gl
		       ON gl.player_id = rs.player_id AND gl.game_date BETWEEN ? AND ?
		WHERE rs.league_id = ? AND rs.team_id IN (?, ?)
		  AND rs.slot_type IN ('active', 'substitute')
		GROUP BY rs.player_id
		HAVING pts > 0
		ORDER BY pts DESC
		LIMIT 5
	`, from, to, leagueID, homeTeamID, awayTeamID)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var out []devTopScorer
	for rows.Next() {
		var s devTopScorer
		rows.Scan(&s.Name, &s.Position, &s.Team, &s.Points)
		out = append(out, s)
	}
	return out
}

// runInjuryDetection checks each rostered player's game log presence for the
// week. Players with no logs increment consecutive_misses; those with logs reset
// it. Players reaching 2 consecutive misses are flagged injured.
func (h *DevHandler) runInjuryDetection(leagueID int, weekStart, weekEnd time.Time) (newInj, cleared []string) {
	from := weekStart.Format("2006-01-02")
	to := weekEnd.Format("2006-01-02")

	rows, err := h.store.DB().Query(`
		SELECT rs.player_id, np.name, np.nhl_team_code,
		       EXISTS(
		           SELECT 1 FROM player_game_logs gl
		           WHERE gl.player_id = rs.player_id AND gl.game_date BETWEEN ? AND ?
		       ) AS played,
		       COALESCE(inj.consecutive_misses, 0),
		       rs.slot_type
		FROM roster_slots rs
		JOIN nhl_players np ON np.id = rs.player_id
		LEFT JOIN injury_flags inj ON inj.player_id = rs.player_id
		WHERE rs.league_id = ?
	`, from, to, leagueID)
	if err != nil {
		log.Printf("[dev] injury detection query: %v", err)
		return
	}
	defer rows.Close()

	type playerRow struct {
		id      int
		name    string
		team    string
		played  bool
		misses  int
		slotType string
	}
	var players []playerRow
	for rows.Next() {
		var p playerRow
		rows.Scan(&p.id, &p.name, &p.team, &p.played, &p.misses, &p.slotType)
		players = append(players, p)
	}

	for _, p := range players {
		if p.played {
			// Played this week — resolve any injury, reset misses.
			if p.slotType == "injured" {
				if err := h.store.ResolvePlayerInjury(leagueID, p.id); err == nil {
					cleared = append(cleared, fmt.Sprintf("%s (%s)", p.name, p.team))
				}
			}
			h.store.DB().Exec(`
				UPDATE injury_flags SET consecutive_misses = 0 WHERE player_id = ?`, p.id)
		} else {
			// Missed this week — increment counter.
			h.store.DB().Exec(`
				INSERT INTO injury_flags (player_id, consecutive_misses, is_ltir)
				VALUES (?, 1, 0)
				ON CONFLICT(player_id) DO UPDATE SET consecutive_misses = consecutive_misses + 1
			`, p.id)

			newMisses := p.misses + 1
			if newMisses >= 2 && p.slotType == "active" {
				if err := h.store.FlagPlayerInjured(leagueID, p.id, false); err == nil {
					newInj = append(newInj, fmt.Sprintf("%s (%s)", p.name, p.team))
					log.Printf("[dev] flagged %s as injured (missed %d weeks)", p.name, newMisses)
				}
			}
		}
	}
	return
}

// ── Stat generators (identical to cmd/simulate/main.go) ──────────────────────

func devSkaterStats(rng *rand.Rand, salary int64) (goals, assists int) {
	if rng.Float64() > 0.42 {
		return 0, 0
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
		goals++
	}
	if rng.Float64() < aRate {
		assists = 1
	}
	if rng.Float64() < aRate*0.4 {
		assists++
	}
	return goals, assists
}

func devGoalieStats(rng *rand.Rand, salary int64) (wins, otl, shutouts int) {
	if rng.Float64() > 0.22 {
		return 0, 0, 0
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
	return wins, otl, shutouts
}
