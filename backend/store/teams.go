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
