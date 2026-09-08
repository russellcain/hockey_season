package store

import (
	"errors"
	"fmt"
	"time"
)

var ErrScheduleExists = errors.New("schedule already generated")

// GenerateH2HSchedule creates a round-robin schedule where each team plays every
// other team exactly 3 times, distributed across ~23 weeks of NHL season (Oct–Apr).
func (s *Store) GenerateH2HSchedule(leagueID int) error {
	// Check if schedule already exists
	var existing int
	s.db.QueryRow(`SELECT COUNT(*) FROM matchups WHERE league_id = ?`, leagueID).Scan(&existing)
	if existing > 0 {
		return ErrScheduleExists
	}

	teams, err := s.ListTeamsByLeague(leagueID)
	if err != nil || len(teams) < 2 {
		return fmt.Errorf("need at least 2 teams: %w", err)
	}

	// Build all matchup pairs (each pair 3 times)
	type pair struct{ home, away int }
	var matchups []pair
	for rep := 0; rep < 3; rep++ {
		for i := 0; i < len(teams); i++ {
			for j := i + 1; j < len(teams); j++ {
				if rep%2 == 0 {
					matchups = append(matchups, pair{teams[i].ID, teams[j].ID})
				} else {
					matchups = append(matchups, pair{teams[j].ID, teams[i].ID})
				}
			}
		}
	}

	// Distribute across 23 weeks, filling slots round-robin
	gamesPerWeek := len(teams) / 2
	if gamesPerWeek == 0 {
		gamesPerWeek = 1
	}

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	week := 1
	weekCount := 0
	for _, m := range matchups {
		_, err := tx.Exec(`
			INSERT INTO matchups (league_id, week_number, home_team_id, away_team_id)
			VALUES (?, ?, ?, ?)
		`, leagueID, week, m.home, m.away)
		if err != nil {
			return fmt.Errorf("insert matchup: %w", err)
		}
		weekCount++
		if weekCount >= gamesPerWeek {
			weekCount = 0
			week++
		}
	}

	return tx.Commit()
}

// GetMatchups returns all matchups for a league, with team data populated.
func (s *Store) GetMatchups(leagueID int) ([]Matchup, error) {
	return s.queryMatchups(leagueID, 0, false)
}

// GetWeekMatchups returns matchups for a specific week.
func (s *Store) GetWeekMatchups(leagueID, weekNumber int) ([]Matchup, error) {
	return s.queryMatchups(leagueID, weekNumber, true)
}

func (s *Store) queryMatchups(leagueID, weekNumber int, filterWeek bool) ([]Matchup, error) {
	query := `
		SELECT m.id, m.week_number,
		       m.home_team_id, ht.name, ht.manager,
		       COALESCE(ht.cap_used, 0), COALESCE(ht.transactions_used, 0), COALESCE(ht.trades_used, 0),
		       m.away_team_id, at.name, at.manager,
		       COALESCE(at.cap_used, 0), COALESCE(at.transactions_used, 0), COALESCE(at.trades_used, 0),
		       m.home_score, m.away_score, m.home_points, m.away_points
		FROM matchups m
		JOIN fantasy_teams ht ON ht.id = m.home_team_id
		JOIN fantasy_teams at ON at.id = m.away_team_id
		WHERE m.league_id = ?
	`
	args := []any{leagueID}
	if filterWeek {
		query += " AND m.week_number = ?"
		args = append(args, weekNumber)
	}
	query += " ORDER BY m.week_number, m.id"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Matchup
	for rows.Next() {
		var m Matchup
		if err := rows.Scan(
			&m.ID, &m.WeekNumber,
			&m.HomeTeamID, &m.HomeTeam.Name, &m.HomeTeam.Manager,
			&m.HomeTeam.CapUsed, &m.HomeTeam.TransactionsUsed, &m.HomeTeam.TradesUsed,
			&m.AwayTeamID, &m.AwayTeam.Name, &m.AwayTeam.Manager,
			&m.AwayTeam.CapUsed, &m.AwayTeam.TransactionsUsed, &m.AwayTeam.TradesUsed,
			&m.HomeScore, &m.AwayScore, &m.HomePoints, &m.AwayPoints,
		); err != nil {
			return nil, err
		}
		m.HomeTeam.ID = m.HomeTeamID
		m.AwayTeam.ID = m.AwayTeamID
		out = append(out, m)
	}
	return out, rows.Err()
}

// UpdateMatchupScores computes and persists scores for all matchups in a week.
// Scores cover the calendar week (Mon–Sun) in which the week_number falls.
func (s *Store) UpdateMatchupScores(leagueID, weekNumber int) error {
	// Derive season start from the league's season_year so this works correctly
	// regardless of what year the server is running.
	var seasonYear int
	if err := s.db.QueryRow(`SELECT season_year FROM leagues WHERE id = ?`, leagueID).Scan(&seasonYear); err != nil {
		return fmt.Errorf("UpdateMatchupScores: get season_year: %w", err)
	}
	seasonStart := firstMondayOfOctober(seasonYear)
	weekStart := seasonStart.AddDate(0, 0, (weekNumber-1)*7)
	weekEnd := weekStart.AddDate(0, 0, 6)

	matchups, err := s.GetWeekMatchups(leagueID, weekNumber)
	if err != nil {
		return err
	}

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, m := range matchups {
		homeScore, err := s.TeamScore(leagueID, m.HomeTeamID, weekStart, weekEnd)
		if err != nil {
			return err
		}
		awayScore, err := s.TeamScore(leagueID, m.AwayTeamID, weekStart, weekEnd)
		if err != nil {
			return err
		}

		homePoints, awayPoints := scoreToPoints(homeScore.Total, awayScore.Total)

		_, err = tx.Exec(`
			UPDATE matchups
			SET home_score = ?, away_score = ?, home_points = ?, away_points = ?
			WHERE id = ?
		`, homeScore.Total, awayScore.Total, homePoints, awayPoints, m.ID)
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}

func scoreToPoints(home, away float64) (homePoints, awayPoints int) {
	if home > away {
		return 2, 0
	} else if away > home {
		return 0, 2
	}
	return 1, 1
}

// FirstMondayOfOctober returns the first Monday on or after October 1 of year.
// This is the canonical fantasy season start used for week-number calculations.
func FirstMondayOfOctober(year int) time.Time { return firstMondayOfOctober(year) }

func firstMondayOfOctober(year int) time.Time {
	oct1 := time.Date(year, time.October, 1, 0, 0, 0, 0, time.UTC)
	// weekday: 0=Sunday, 1=Monday, ...
	dow := int(oct1.Weekday())
	daysUntilMonday := (8 - dow) % 7
	if dow == 1 {
		daysUntilMonday = 0
	}
	return oct1.AddDate(0, 0, daysUntilMonday)
}
