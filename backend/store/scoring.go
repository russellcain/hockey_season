package store

import (
	"fmt"
	"time"
)

// GetRosterPlayerStats returns season-aggregate Stats keyed by player ID for
// every player currently on a team's roster. Skaters get goals/assists/fp;
// goalies get wins/otl/shutouts/fp.
func (s *Store) GetRosterPlayerStats(leagueID, teamID int) (map[int]Stats, error) {
	rows, err := s.db.Query(`
		SELECT rs.player_id, np.position,
		       COALESCE(SUM(gl.goals), 0),
		       COALESCE(SUM(gl.assists), 0),
		       COALESCE(SUM(gl.wins), 0),
		       COALESCE(SUM(gl.otl), 0),
		       COALESCE(SUM(gl.shutouts), 0),
		       COUNT(gl.game_date)
		FROM roster_slots rs
		JOIN nhl_players np ON np.id = rs.player_id
		LEFT JOIN player_game_logs gl ON gl.player_id = rs.player_id
		WHERE rs.league_id = ? AND rs.team_id = ?
		GROUP BY rs.player_id, np.position
	`, leagueID, teamID)
	if err != nil {
		return nil, fmt.Errorf("GetRosterPlayerStats: %w", err)
	}
	defer rows.Close()

	out := make(map[int]Stats)
	for rows.Next() {
		var playerID int
		var pos string
		var st Stats
		if err := rows.Scan(&playerID, &pos,
			&st.Goals, &st.Assists, &st.Wins, &st.OTL, &st.Shutouts, &st.GamesPlayed,
		); err != nil {
			return nil, err
		}
		if pos == "G" {
			st.FP = float64(st.Wins)*2 + float64(st.OTL) + float64(st.Shutouts)
		} else {
			st.FP = float64(st.Goals + st.Assists)
		}
		out[playerID] = st
	}
	return out, rows.Err()
}

// UpsertGameLog inserts or replaces a player's game log entry for a date.
func (s *Store) UpsertGameLog(playerID int, date string, goals, assists, wins, otl, shutouts int) error {
	_, err := s.db.Exec(`
		INSERT INTO player_game_logs (player_id, game_date, goals, assists, wins, otl, shutouts)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(player_id, game_date) DO UPDATE SET
		  goals = excluded.goals,
		  assists = excluded.assists,
		  wins = excluded.wins,
		  otl = excluded.otl,
		  shutouts = excluded.shutouts
	`, playerID, date, goals, assists, wins, otl, shutouts)
	return err
}

// TeamScore computes a team's scoring breakdown over [from, to].
// It counts stats for all players currently on the roster (active+substitute)
// whose game_date falls in the range.
// Note: a complete implementation would filter by roster membership at time of game.
func (s *Store) TeamScore(leagueID, teamID int, from, to time.Time) (ScoreBreakdown, error) {
	fromStr := from.Format("2006-01-02")
	toStr := to.Format("2006-01-02")

	rows, err := s.db.Query(`
		SELECT np.position,
		       COALESCE(SUM(gl.goals), 0),
		       COALESCE(SUM(gl.assists), 0),
		       COALESCE(SUM(gl.wins), 0),
		       COALESCE(SUM(gl.otl), 0),
		       COALESCE(SUM(gl.shutouts), 0)
		FROM roster_slots rs
		JOIN nhl_players np ON np.id = rs.player_id
		LEFT JOIN player_game_logs gl
		       ON gl.player_id = rs.player_id
		      AND gl.game_date BETWEEN ? AND ?
		WHERE rs.league_id = ? AND rs.team_id = ?
		  AND rs.slot_type IN ('active', 'substitute')
		GROUP BY np.position
	`, fromStr, toStr, leagueID, teamID)
	if err != nil {
		return ScoreBreakdown{}, fmt.Errorf("team score: %w", err)
	}
	defer rows.Close()

	var sb ScoreBreakdown
	for rows.Next() {
		var pos string
		var goals, assists, wins, otl, shutouts int
		if err := rows.Scan(&pos, &goals, &assists, &wins, &otl, &shutouts); err != nil {
			return ScoreBreakdown{}, err
		}
		switch pos {
		case "G":
			sb.GoalieWins += wins
			sb.GoalieOTL += otl
			sb.GoalieSO += shutouts
		default: // F or D
			sb.Goals += goals
			sb.Assists += assists
		}
	}
	if err := rows.Err(); err != nil {
		return ScoreBreakdown{}, err
	}

	sb.Total = float64(sb.Goals+sb.Assists) +
		float64(sb.GoalieWins)*2 +
		float64(sb.GoalieOTL)*1 +
		float64(sb.GoalieSO)*1

	return sb, nil
}

// GetStandings returns aggregate and H2H standings for a league.
func (s *Store) GetStandings(leagueID int) (aggregate []TeamStanding, h2h []TeamStanding, err error) {
	teams, err := s.ListTeamsByLeague(leagueID)
	if err != nil {
		return nil, nil, fmt.Errorf("standings teams: %w", err)
	}

	// Build H2H standings from matchups table
	type h2hRow struct {
		teamID  int
		points  int
		wins    int
		ties    int
		losses  int
	}
	h2hMap := make(map[int]*h2hRow)
	for _, t := range teams {
		h2hMap[t.ID] = &h2hRow{teamID: t.ID}
	}

	rows, err := s.db.Query(`
		SELECT home_team_id, away_team_id, home_points, away_points
		FROM matchups WHERE league_id = ?
	`, leagueID)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var homeID, awayID, homePoints, awayPoints int
		if err := rows.Scan(&homeID, &awayID, &homePoints, &awayPoints); err != nil {
			return nil, nil, err
		}
		if r, ok := h2hMap[homeID]; ok {
			r.points += homePoints
			if homePoints > awayPoints {
				r.wins++
			} else if homePoints == awayPoints && homePoints > 0 {
				r.ties++
			} else if homePoints > 0 || awayPoints > 0 {
				r.losses++
			}
		}
		if r, ok := h2hMap[awayID]; ok {
			r.points += awayPoints
			if awayPoints > homePoints {
				r.wins++
			} else if awayPoints == homePoints && awayPoints > 0 {
				r.ties++
			} else if homePoints > 0 || awayPoints > 0 {
				r.losses++
			}
		}
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}

	// Aggregate scores: sum all scored matchup points + compute season totals
	// For aggregate, use season-long stats from player_game_logs via roster
	now := time.Now()
	seasonStart := time.Date(now.Year(), time.October, 1, 0, 0, 0, 0, time.UTC)
	if now.Month() < time.October {
		seasonStart = time.Date(now.Year()-1, time.October, 1, 0, 0, 0, 0, time.UTC)
	}

	aggMap := make(map[int]*TeamStanding)
	for _, t := range teams {
		aggMap[t.ID] = &TeamStanding{Team: t}
	}

	for _, t := range teams {
		sb, err := s.TeamScore(leagueID, t.ID, seasonStart, now)
		if err != nil {
			return nil, nil, fmt.Errorf("score for team %d: %w", t.ID, err)
		}
		if ts, ok := aggMap[t.ID]; ok {
			ts.TotalPoints = sb.Total
			ts.Goals = sb.Goals
		}
	}

	for _, t := range teams {
		ts := aggMap[t.ID]
		hr := h2hMap[t.ID]
		ts.H2HPoints = hr.points
		ts.H2HWins = hr.wins
		ts.H2HTies = hr.ties
		ts.H2HLosses = hr.losses
		aggregate = append(aggregate, *ts)
		h2h = append(h2h, *ts)
	}

	// Sort aggregate by TotalPoints desc
	for i := 0; i < len(aggregate)-1; i++ {
		for j := i + 1; j < len(aggregate); j++ {
			if aggregate[j].TotalPoints > aggregate[i].TotalPoints ||
				(aggregate[j].TotalPoints == aggregate[i].TotalPoints && aggregate[j].Goals > aggregate[i].Goals) {
				aggregate[i], aggregate[j] = aggregate[j], aggregate[i]
			}
		}
	}

	// Sort H2H by H2HPoints desc
	for i := 0; i < len(h2h)-1; i++ {
		for j := i + 1; j < len(h2h); j++ {
			if h2h[j].H2HPoints > h2h[i].H2HPoints ||
				(h2h[j].H2HPoints == h2h[i].H2HPoints && h2h[j].H2HWins > h2h[i].H2HWins) {
				h2h[i], h2h[j] = h2h[j], h2h[i]
			}
		}
	}

	return aggregate, h2h, nil
}
