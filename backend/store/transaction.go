package store

import (
	"database/sql"
	"errors"
	"fmt"
)

const maxElectiveTransactions = 15

var (
	ErrSamePosition      = errors.New("dropped and added player must be same position")
	ErrNotInjured        = errors.New("player is not injured")
	ErrSubCapExceeded    = errors.New("substitute salary exceeds injured player cap ceiling")
	ErrNoSubstitute      = errors.New("no active substitute found")
)

// TransactionRecord is one row from the transactions log, enriched with names.
type TransactionRecord struct {
	ID            int    `json:"id"`
	TeamID        int    `json:"teamId"`
	TeamName      string `json:"teamName"`
	DroppedPlayer string `json:"droppedPlayer"`
	AddedPlayer   string `json:"addedPlayer"`
	TxnType       string `json:"txnType"`
	CreatedAt     string `json:"createdAt"`
}

// GetLeagueTransactions returns all transactions for a league, newest first.
func (s *Store) GetLeagueTransactions(leagueID int) ([]TransactionRecord, error) {
	rows, err := s.db.Query(`
		SELECT t.id, t.team_id, ft.name,
		       dp.name, ap.name,
		       t.txn_type, t.created_at
		FROM transactions t
		JOIN fantasy_teams ft ON ft.id = t.team_id
		JOIN nhl_players dp   ON dp.id = t.dropped_player_id
		JOIN nhl_players ap   ON ap.id = t.added_player_id
		WHERE t.league_id = ?
		ORDER BY t.created_at DESC
		LIMIT 200
	`, leagueID)
	if err != nil {
		return nil, fmt.Errorf("GetLeagueTransactions: %w", err)
	}
	defer rows.Close()

	var out []TransactionRecord
	for rows.Next() {
		var r TransactionRecord
		if err := rows.Scan(&r.ID, &r.TeamID, &r.TeamName,
			&r.DroppedPlayer, &r.AddedPlayer, &r.TxnType, &r.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// ValidateTransaction checks all rules for a drop/add transaction without executing.
func (s *Store) ValidateTransaction(leagueID, teamID, dropID, addID int) error {
	// Check transaction limit
	var txnUsed int
	if err := s.db.QueryRow(`SELECT COALESCE(transactions_used, 0) FROM fantasy_teams WHERE id = ?`,
		teamID).Scan(&txnUsed); err != nil {
		return fmt.Errorf("get team: %w", err)
	}
	if txnUsed >= maxElectiveTransactions {
		return ErrOverTransactions
	}

	// Dropped player must be on roster
	dropSlot, err := s.GetRosterSlot(leagueID, teamID, dropID)
	if err != nil {
		if err == sql.ErrNoRows {
			return ErrNotOnRoster
		}
		return err
	}

	// Added player must not be rostered
	rostered, err := s.IsPlayerRostered(leagueID, addID)
	if err != nil {
		return err
	}
	if rostered {
		return ErrAlreadyRostered
	}

	// Same position check
	addPlayer, err := s.GetPlayer(addID)
	if err != nil {
		return fmt.Errorf("get add player: %w", err)
	}
	if dropSlot.Player.Position != addPlayer.Position {
		return ErrSamePosition
	}

	// Cap check: after dropping, can we afford the add?
	capUsed, err := s.GetTeamCapUsed(leagueID, teamID)
	if err != nil {
		return err
	}

	league, err := s.GetLeague(leagueID)
	if err != nil {
		return err
	}

	capAfterDrop := capUsed - dropSlot.Player.Salary
	if capAfterDrop+addPlayer.Salary > league.SalaryCap {
		return ErrOverCap
	}
	return nil
}

// ExecuteTransaction atomically drops a player and adds another.
func (s *Store) ExecuteTransaction(leagueID, teamID, dropID, addID int) error {
	if err := s.ValidateTransaction(leagueID, teamID, dropID, addID); err != nil {
		return err
	}

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Get add player salary
	addPlayer, err := s.GetPlayer(addID)
	if err != nil {
		return err
	}
	dropSlot, err := s.GetRosterSlot(leagueID, teamID, dropID)
	if err != nil {
		return err
	}

	// Remove dropped player
	if _, err := tx.Exec(`
		DELETE FROM roster_slots WHERE league_id = ? AND team_id = ? AND player_id = ?
	`, leagueID, teamID, dropID); err != nil {
		return fmt.Errorf("remove drop player: %w", err)
	}

	// Add new player
	if _, err := tx.Exec(`
		INSERT INTO roster_slots (team_id, player_id, league_id, slot_type)
		VALUES (?, ?, ?, 'active')
	`, teamID, addID, leagueID); err != nil {
		return fmt.Errorf("add player: %w", err)
	}

	// Log transaction
	if _, err := tx.Exec(`
		INSERT INTO transactions (team_id, league_id, dropped_player_id, added_player_id, txn_type)
		VALUES (?, ?, ?, ?, 'elective')
	`, teamID, leagueID, dropID, addID); err != nil {
		return fmt.Errorf("log transaction: %w", err)
	}

	// Update cap_used: +addSalary -dropSalary
	capDelta := addPlayer.Salary - dropSlot.Player.Salary
	if _, err := tx.Exec(`
		UPDATE fantasy_teams SET cap_used = cap_used + ?, transactions_used = transactions_used + 1
		WHERE id = ?
	`, capDelta, teamID); err != nil {
		return fmt.Errorf("update cap: %w", err)
	}

	return tx.Commit()
}

// ValidateInjurySub checks rules for placing an injury substitute.
func (s *Store) ValidateInjurySub(leagueID, teamID, injuredID, subID int) error {
	// Injured player must be on roster with slot_type = 'injured'
	injSlot, err := s.GetRosterSlot(leagueID, teamID, injuredID)
	if err != nil {
		if err == sql.ErrNoRows {
			return ErrNotOnRoster
		}
		return err
	}
	if injSlot.SlotType != "injured" {
		return ErrNotInjured
	}

	// Sub must not be rostered
	rostered, err := s.IsPlayerRostered(leagueID, subID)
	if err != nil {
		return err
	}
	if rostered {
		return ErrAlreadyRostered
	}

	// Sub must be same position
	subPlayer, err := s.GetPlayer(subID)
	if err != nil {
		return err
	}
	if subPlayer.Position != injSlot.Player.Position {
		return ErrSamePosition
	}

	// Sub salary must not exceed root injured player's cap ceiling
	rootSalary, err := s.getRootPlayerSalary(leagueID, teamID, injSlot)
	if err != nil {
		return err
	}
	if subPlayer.Salary > rootSalary {
		return ErrSubCapExceeded
	}

	return nil
}

// ExecuteInjurySub atomically places an injury substitute.
func (s *Store) ExecuteInjurySub(leagueID, teamID, injuredID, subID int) error {
	if err := s.ValidateInjurySub(leagueID, teamID, injuredID, subID); err != nil {
		return err
	}

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Determine root player ID for original_player_id chain
	injSlot, err := s.GetRosterSlot(leagueID, teamID, injuredID)
	if err != nil {
		return err
	}
	rootID := injuredID
	if injSlot.OriginalPlayerID != nil {
		rootID = *injSlot.OriginalPlayerID
	}

	// Insert substitute slot
	if _, err := tx.Exec(`
		INSERT INTO roster_slots (team_id, player_id, league_id, slot_type, original_player_id)
		VALUES (?, ?, ?, 'substitute', ?)
	`, teamID, subID, leagueID, rootID); err != nil {
		return fmt.Errorf("add sub: %w", err)
	}

	// Log as injury_sub transaction
	if _, err := tx.Exec(`
		INSERT INTO transactions (team_id, league_id, dropped_player_id, added_player_id, txn_type)
		VALUES (?, ?, ?, ?, 'injury_sub')
	`, teamID, leagueID, injuredID, subID); err != nil {
		return fmt.Errorf("log injury sub: %w", err)
	}

	return tx.Commit()
}

// CutInjuredPlayer removes an injured player (and their sub) from the roster.
// This consumes one elective transaction.
func (s *Store) CutInjuredPlayer(leagueID, teamID, playerID int) error {
	// Check transaction limit
	var txnUsed int
	if err := s.db.QueryRow(`SELECT COALESCE(transactions_used, 0) FROM fantasy_teams WHERE id = ?`,
		teamID).Scan(&txnUsed); err != nil {
		return fmt.Errorf("get team: %w", err)
	}
	if txnUsed >= maxElectiveTransactions {
		return ErrOverTransactions
	}

	injSlot, err := s.GetRosterSlot(leagueID, teamID, playerID)
	if err != nil {
		return ErrNotOnRoster
	}

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Remove substitute if one exists
	if _, err := tx.Exec(`
		DELETE FROM roster_slots
		WHERE league_id = ? AND team_id = ? AND slot_type = 'substitute' AND original_player_id = ?
	`, leagueID, teamID, playerID); err != nil {
		return fmt.Errorf("remove sub: %w", err)
	}

	// Remove injured player
	if _, err := tx.Exec(`
		DELETE FROM roster_slots WHERE league_id = ? AND team_id = ? AND player_id = ?
	`, leagueID, teamID, playerID); err != nil {
		return fmt.Errorf("remove injured: %w", err)
	}

	// Update team: cap_used -= injured player salary, transactions_used++
	if _, err := tx.Exec(`
		UPDATE fantasy_teams
		SET cap_used = cap_used - ?, transactions_used = transactions_used + 1
		WHERE id = ?
	`, injSlot.Player.Salary, teamID); err != nil {
		return fmt.Errorf("update cap: %w", err)
	}

	return tx.Commit()
}

// getRootPlayerSalary follows the original_player_id chain to find the root salary.
func (s *Store) getRootPlayerSalary(leagueID, teamID int, slot *RosterSlot) (int64, error) {
	if slot.OriginalPlayerID == nil {
		return slot.Player.Salary, nil
	}
	rootID := *slot.OriginalPlayerID
	root, err := s.GetPlayer(rootID)
	if err != nil {
		return 0, fmt.Errorf("get root player: %w", err)
	}
	return root.Salary, nil
}
