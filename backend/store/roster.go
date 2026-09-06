package store

import (
	"database/sql"
	"errors"
	"fmt"
)

var (
	ErrNotOnRoster      = errors.New("player not on roster")
	ErrAlreadyRostered  = errors.New("player already rostered")
	ErrOverTransactions = errors.New("transaction limit reached")
)

// GetRoster returns all roster slots for a team in a league, with full player data.
func (s *Store) GetRoster(leagueID, teamID int) ([]RosterSlot, error) {
	rows, err := s.db.Query(`
		SELECT rs.id, rs.team_id, rs.player_id, rs.slot_type, rs.original_player_id,
		       np.name, np.nhl_team_name, np.nhl_team_code, np.position, np.salary_cap_hit, np.age
		FROM roster_slots rs
		JOIN nhl_players np ON np.id = rs.player_id
		WHERE rs.league_id = ? AND rs.team_id = ?
		ORDER BY np.position, np.name
	`, leagueID, teamID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var slots []RosterSlot
	for rows.Next() {
		var rs RosterSlot
		var origID sql.NullInt64
		var salaryText, ageText string
		if err := rows.Scan(
			&rs.ID, &rs.TeamID, &rs.PlayerID, &rs.SlotType, &origID,
			&rs.Player.Name, &rs.Player.NhlTeam, &rs.Player.NhlTeamCode,
			&rs.Player.Position, &salaryText, &ageText,
		); err != nil {
			return nil, err
		}
		rs.Player.ID = rs.PlayerID
		rs.Player.Salary = parseSalary(salaryText)
		rs.Player.Age = parseAge(ageText)
		if origID.Valid {
			v := int(origID.Int64)
			rs.OriginalPlayerID = &v
		}
		slots = append(slots, rs)
	}
	return slots, rows.Err()
}

// AddToRoster inserts a new roster slot. originalPlayerID is set for injury subs.
func (s *Store) AddToRoster(leagueID, teamID, playerID int, slotType string, originalPlayerID *int) error {
	_, err := s.db.Exec(`
		INSERT INTO roster_slots (team_id, player_id, league_id, slot_type, original_player_id)
		VALUES (?, ?, ?, ?, ?)
	`, teamID, playerID, leagueID, slotType, originalPlayerID)
	return err
}

// RemoveFromRoster deletes a roster slot.
func (s *Store) RemoveFromRoster(leagueID, teamID, playerID int) error {
	res, err := s.db.Exec(`
		DELETE FROM roster_slots
		WHERE league_id = ? AND team_id = ? AND player_id = ?
	`, leagueID, teamID, playerID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotOnRoster
	}
	return nil
}

// IsPlayerRostered returns true if any team in the league has the player.
func (s *Store) IsPlayerRostered(leagueID, playerID int) (bool, error) {
	var n int
	err := s.db.QueryRow(`
		SELECT COUNT(*) FROM roster_slots
		WHERE league_id = ? AND player_id = ?
	`, leagueID, playerID).Scan(&n)
	return n > 0, err
}

// GetTeamCapUsed sums salary_cap_hit for active and injured roster slots.
func (s *Store) GetTeamCapUsed(leagueID, teamID int) (int64, error) {
	rows, err := s.db.Query(`
		SELECT np.salary_cap_hit
		FROM roster_slots rs
		JOIN nhl_players np ON np.id = rs.player_id
		WHERE rs.league_id = ? AND rs.team_id = ?
		  AND rs.slot_type IN ('active', 'injured')
	`, leagueID, teamID)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	var total int64
	for rows.Next() {
		var salaryText string
		if err := rows.Scan(&salaryText); err != nil {
			return 0, err
		}
		total += parseSalary(salaryText)
	}
	return total, rows.Err()
}

// GetAvailablePlayers returns players not on any roster in the league, with optional filters.
func (s *Store) GetAvailablePlayers(leagueID int, position, nameQuery string, maxSalary int64) ([]Player, error) {
	query := `
		SELECT np.id, np.name, np.nhl_team_name, np.nhl_team_code, np.position, np.salary_cap_hit, np.age
		FROM nhl_players np
		WHERE np.id NOT IN (
			SELECT rs.player_id FROM roster_slots rs WHERE rs.league_id = ?
		)
	`
	args := []any{leagueID}

	if position != "" {
		query += " AND np.position = ?"
		args = append(args, position)
	}
	if nameQuery != "" {
		query += " AND LOWER(np.name) LIKE '%' || LOWER(?) || '%'"
		args = append(args, nameQuery)
	}
	if maxSalary > 0 {
		query += " AND CAST(REPLACE(np.salary_cap_hit, ',', '') AS INTEGER) <= ?"
		args = append(args, maxSalary)
	}
	query += " ORDER BY np.name LIMIT 100"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("available players: %w", err)
	}
	defer rows.Close()

	var players []Player
	for rows.Next() {
		var p Player
		var salaryText, ageText string
		if err := rows.Scan(&p.ID, &p.Name, &p.NhlTeam, &p.NhlTeamCode, &p.Position, &salaryText, &ageText); err != nil {
			return nil, err
		}
		p.Salary = parseSalary(salaryText)
		p.Age = parseAge(ageText)
		players = append(players, p)
	}
	return players, rows.Err()
}

// GetRosterSlot returns a single slot by league, team and player.
func (s *Store) GetRosterSlot(leagueID, teamID, playerID int) (*RosterSlot, error) {
	var rs RosterSlot
	var origID sql.NullInt64
	var salaryText, ageText string
	err := s.db.QueryRow(`
		SELECT rs.id, rs.team_id, rs.player_id, rs.slot_type, rs.original_player_id,
		       np.name, np.nhl_team_name, np.nhl_team_code, np.position, np.salary_cap_hit, np.age
		FROM roster_slots rs
		JOIN nhl_players np ON np.id = rs.player_id
		WHERE rs.league_id = ? AND rs.team_id = ? AND rs.player_id = ?
	`, leagueID, teamID, playerID).Scan(
		&rs.ID, &rs.TeamID, &rs.PlayerID, &rs.SlotType, &origID,
		&rs.Player.Name, &rs.Player.NhlTeam, &rs.Player.NhlTeamCode,
		&rs.Player.Position, &salaryText, &ageText,
	)
	if err != nil {
		return nil, err
	}
	rs.Player.ID = rs.PlayerID
	rs.Player.Salary = parseSalary(salaryText)
	rs.Player.Age = parseAge(ageText)
	if origID.Valid {
		v := int(origID.Int64)
		rs.OriginalPlayerID = &v
	}
	return &rs, nil
}

// GetSubForInjured returns the current substitute slot for an injured player (if any).
func (s *Store) GetSubForInjured(leagueID, teamID, injuredPlayerID int) (*RosterSlot, error) {
	var rs RosterSlot
	var origID sql.NullInt64
	var salaryText, ageText string
	err := s.db.QueryRow(`
		SELECT rs.id, rs.team_id, rs.player_id, rs.slot_type, rs.original_player_id,
		       np.name, np.nhl_team_name, np.nhl_team_code, np.position, np.salary_cap_hit, np.age
		FROM roster_slots rs
		JOIN nhl_players np ON np.id = rs.player_id
		WHERE rs.league_id = ? AND rs.team_id = ?
		  AND rs.slot_type = 'substitute'
		  AND rs.original_player_id = ?
	`, leagueID, teamID, injuredPlayerID).Scan(
		&rs.ID, &rs.TeamID, &rs.PlayerID, &rs.SlotType, &origID,
		&rs.Player.Name, &rs.Player.NhlTeam, &rs.Player.NhlTeamCode,
		&rs.Player.Position, &salaryText, &ageText,
	)
	if err != nil {
		return nil, err
	}
	rs.Player.ID = rs.PlayerID
	rs.Player.Salary = parseSalary(salaryText)
	rs.Player.Age = parseAge(ageText)
	if origID.Valid {
		v := int(origID.Int64)
		rs.OriginalPlayerID = &v
	}
	return &rs, nil
}
