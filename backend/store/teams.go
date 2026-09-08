package store

import "database/sql"

// GetActiveSessionID returns the ID of the most recent in-progress draft session.
func (s *Store) GetActiveSessionID() (int, error) {
	var id int
	err := s.db.QueryRow(
		`SELECT id FROM draft_sessions WHERE status = 'in_progress' ORDER BY id DESC LIMIT 1`,
	).Scan(&id)
	return id, err
}

func (s *Store) LookupTeamByCode(codeHash string) (*Team, error) {
	var t Team
	err := s.db.QueryRow(`
		SELECT id, name, manager, cap_used,
		       COALESCE(transactions_used, 0), COALESCE(trades_used, 0)
		FROM fantasy_teams WHERE code_hash = ?
	`, codeHash).Scan(&t.ID, &t.Name, &t.Manager, &t.CapUsed, &t.TransactionsUsed, &t.TradesUsed)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (s *Store) GetTeam(id int) (*Team, error) {
	var t Team
	err := s.db.QueryRow(`
		SELECT id, name, manager, cap_used,
		       COALESCE(transactions_used, 0), COALESCE(trades_used, 0)
		FROM fantasy_teams WHERE id = ?
	`, id).Scan(&t.ID, &t.Name, &t.Manager, &t.CapUsed, &t.TransactionsUsed, &t.TradesUsed)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (s *Store) ListTeams() ([]Team, error) {
	rows, err := s.db.Query(`
		SELECT id, name, manager, cap_used,
		       COALESCE(transactions_used, 0), COALESCE(trades_used, 0)
		FROM fantasy_teams ORDER BY id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var teams []Team
	for rows.Next() {
		var t Team
		if err := rows.Scan(&t.ID, &t.Name, &t.Manager, &t.CapUsed, &t.TransactionsUsed, &t.TradesUsed); err != nil {
			return nil, err
		}
		teams = append(teams, t)
	}
	return teams, rows.Err()
}

// ListTeamsByLeague returns teams belonging to a specific league.
func (s *Store) ListTeamsByLeague(leagueID int) ([]Team, error) {
	rows, err := s.db.Query(`
		SELECT id, name, manager, cap_used,
		       COALESCE(transactions_used, 0), COALESCE(trades_used, 0)
		FROM fantasy_teams WHERE league_id = ? ORDER BY id
	`, leagueID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var teams []Team
	for rows.Next() {
		var t Team
		if err := rows.Scan(&t.ID, &t.Name, &t.Manager, &t.CapUsed, &t.TransactionsUsed, &t.TradesUsed); err != nil {
			return nil, err
		}
		teams = append(teams, t)
	}
	return teams, rows.Err()
}

// UpdateTeamEmail sets the notification email for a team.
func (s *Store) UpdateTeamEmail(teamID int, email string) error {
	_, err := s.db.Exec(`UPDATE fantasy_teams SET email = ? WHERE id = ?`, email, teamID)
	return err
}

// TeamEmailInfo holds the minimum info needed to send a notification.
type TeamEmailInfo struct {
	ID      int
	Name    string
	Manager string
	Email   string // may be empty
}

// GetLeagueTeamEmails returns all teams in a league with their emails.
func (s *Store) GetLeagueTeamEmails(leagueID int) ([]TeamEmailInfo, error) {
	rows, err := s.db.Query(`
		SELECT id, name, manager, COALESCE(email, '')
		FROM fantasy_teams WHERE league_id = ? ORDER BY id
	`, leagueID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []TeamEmailInfo
	for rows.Next() {
		var t TeamEmailInfo
		if err := rows.Scan(&t.ID, &t.Name, &t.Manager, &t.Email); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// GetPlayerOwnerTeam returns the TeamEmailInfo for the team that has a player
// on their roster in the given league (any slot type).
func (s *Store) GetPlayerOwnerTeam(leagueID, playerID int) (*TeamEmailInfo, error) {
	var t TeamEmailInfo
	err := s.db.QueryRow(`
		SELECT ft.id, ft.name, ft.manager, COALESCE(ft.email, '')
		FROM roster_slots rs
		JOIN fantasy_teams ft ON ft.id = rs.team_id
		WHERE rs.league_id = ? AND rs.player_id = ?
		LIMIT 1
	`, leagueID, playerID).Scan(&t.ID, &t.Name, &t.Manager, &t.Email)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// GetCommissionerTeamID returns the ID of the team with the lowest ID (commissioner).
func (s *Store) GetCommissionerTeamID() (int, error) {
	var id int
	err := s.db.QueryRow(`SELECT id FROM fantasy_teams ORDER BY id LIMIT 1`).Scan(&id)
	return id, err
}

// teamByIDFromTx loads a Team inside an existing transaction.
func teamByIDFromTx(tx *sql.Tx, id int) (Team, error) {
	var t Team
	err := tx.QueryRow(`
		SELECT id, name, manager, cap_used,
		       COALESCE(transactions_used, 0), COALESCE(trades_used, 0)
		FROM fantasy_teams WHERE id = ?
	`, id).Scan(&t.ID, &t.Name, &t.Manager, &t.CapUsed, &t.TransactionsUsed, &t.TradesUsed)
	return t, err
}
