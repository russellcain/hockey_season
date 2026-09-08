package store

import (
	"fmt"
	"time"
)

// inferSeasonYear returns the year a season starts based on the current date.
// Oct–Dec → this year (e.g. Oct 2025 → 2025 for the 2025-26 season).
// Jan–Sep → last year (e.g. Aug 2026 → 2025 for the same season).
func inferSeasonYear() int {
	now := time.Now()
	if now.Month() >= time.October {
		return now.Year()
	}
	return now.Year() - 1
}

// CreateLeague inserts a new league and returns it.
func (s *Store) CreateLeague(name string, salaryCap int64) (*League, error) {
	return s.CreateLeagueForYear(name, salaryCap, inferSeasonYear())
}

// CreateLeagueForYear inserts a league with an explicit season start year.
func (s *Store) CreateLeagueForYear(name string, salaryCap int64, seasonYear int) (*League, error) {
	res, err := s.db.Exec(`
		INSERT INTO leagues (name, salary_cap, status, season_year) VALUES (?, ?, 'setup', ?)
	`, name, salaryCap, seasonYear)
	if err != nil {
		return nil, fmt.Errorf("create league: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	return s.GetLeague(int(id))
}

// GetLeague fetches a league by ID.
func (s *Store) GetLeague(id int) (*League, error) {
	var l League
	err := s.db.QueryRow(`
		SELECT id, name, salary_cap, status, season_year, created_at FROM leagues WHERE id = ?
	`, id).Scan(&l.ID, &l.Name, &l.SalaryCap, &l.Status, &l.SeasonYear, &l.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("get league %d: %w", id, err)
	}
	return &l, nil
}

// UpdateLeagueStatus advances the league to a new status.
func (s *Store) UpdateLeagueStatus(id int, status string) error {
	_, err := s.db.Exec(`UPDATE leagues SET status = ? WHERE id = ?`, status, id)
	return err
}

// ListLeagues returns all leagues.
func (s *Store) ListLeagues() ([]League, error) {
	rows, err := s.db.Query(`SELECT id, name, salary_cap, status, season_year, created_at FROM leagues ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var leagues []League
	for rows.Next() {
		var l League
		if err := rows.Scan(&l.ID, &l.Name, &l.SalaryCap, &l.Status, &l.SeasonYear, &l.CreatedAt); err != nil {
			return nil, err
		}
		leagues = append(leagues, l)
	}
	return leagues, rows.Err()
}
