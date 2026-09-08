package store

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
)

const maxTrades = 3

var (
	ErrTradeNotFound   = errors.New("trade not found")
	ErrTradeLimitReached = errors.New("trade limit reached (max 3)")
	ErrPlayerInBothSides = errors.New("player appears in both sides of trade")
	ErrTradeNotPending = errors.New("trade is not pending")
)

// ProposeTrade creates a pending trade between two teams.
// fromPlayerIDs are moved from fromTeamID to toTeamID, toPlayerIDs vice versa.
func (s *Store) ProposeTrade(leagueID, fromTeamID, toTeamID int, fromPlayerIDs, toPlayerIDs []int) (int, error) {
	// Validate trade limits
	fromTeam, err := s.GetTeam(fromTeamID)
	if err != nil {
		return 0, err
	}
	if fromTeam.TradesUsed >= maxTrades {
		return 0, ErrTradeLimitReached
	}

	// Check for overlap between sides
	fromSet := make(map[int]bool)
	for _, id := range fromPlayerIDs {
		fromSet[id] = true
	}
	for _, id := range toPlayerIDs {
		if fromSet[id] {
			return 0, ErrPlayerInBothSides
		}
	}

	// Validate all fromPlayerIDs are on fromTeam
	for _, pid := range fromPlayerIDs {
		if _, err := s.GetRosterSlot(leagueID, fromTeamID, pid); err != nil {
			return 0, fmt.Errorf("player %d not on from-team: %w", pid, ErrNotOnRoster)
		}
	}

	// Validate all toPlayerIDs are on toTeam
	for _, pid := range toPlayerIDs {
		if _, err := s.GetRosterSlot(leagueID, toTeamID, pid); err != nil {
			return 0, fmt.Errorf("player %d not on to-team: %w", pid, ErrNotOnRoster)
		}
	}

	tx, err := s.db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	res, err := tx.Exec(`
		INSERT INTO trades (league_id, status, submitted_by_team_id)
		VALUES (?, 'pending', ?)
	`, leagueID, fromTeamID)
	if err != nil {
		return 0, fmt.Errorf("create trade: %w", err)
	}
	tradeID64, _ := res.LastInsertId()
	tradeID := int(tradeID64)

	// Insert legs: fromPlayerIDs go from fromTeam to toTeam
	for _, pid := range fromPlayerIDs {
		if _, err := tx.Exec(`
			INSERT INTO trade_legs (trade_id, from_team_id, to_team_id, player_id)
			VALUES (?, ?, ?, ?)
		`, tradeID, fromTeamID, toTeamID, pid); err != nil {
			return 0, fmt.Errorf("insert leg: %w", err)
		}
	}
	// toPlayerIDs go from toTeam to fromTeam
	for _, pid := range toPlayerIDs {
		if _, err := tx.Exec(`
			INSERT INTO trade_legs (trade_id, from_team_id, to_team_id, player_id)
			VALUES (?, ?, ?, ?)
		`, tradeID, toTeamID, fromTeamID, pid); err != nil {
			return 0, fmt.Errorf("insert leg: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}

	log.Printf("[email deferred] Trade proposed: trade %d in league %d from team %d to team %d",
		tradeID, leagueID, fromTeamID, toTeamID)

	return tradeID, nil
}

// ApproveTrade executes an approved trade: moves players and increments trade counts.
func (s *Store) ApproveTrade(tradeID, reviewerTeamID int) error {
	// Load trade
	var trade struct {
		leagueID        int
		submittedByTeam int
		status          string
	}
	err := s.db.QueryRow(`
		SELECT league_id, submitted_by_team_id, status FROM trades WHERE id = ?
	`, tradeID).Scan(&trade.leagueID, &trade.submittedByTeam, &trade.status)
	if err != nil {
		if err == sql.ErrNoRows {
			return ErrTradeNotFound
		}
		return err
	}
	if trade.status != "pending" {
		return ErrTradeNotPending
	}

	// Load legs
	legs, err := s.tradeLegs(tradeID)
	if err != nil {
		return err
	}

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	affectedTeams := map[int]bool{}
	for _, leg := range legs {
		// Move player from leg.FromTeam.ID to leg.ToTeam.ID
		if _, err := tx.Exec(`
			UPDATE roster_slots
			SET team_id = ?
			WHERE league_id = ? AND team_id = ? AND player_id = ?
		`, leg.ToTeam.ID, trade.leagueID, leg.FromTeam.ID, leg.Player.ID); err != nil {
			return fmt.Errorf("move player %d: %w", leg.Player.ID, err)
		}
		affectedTeams[leg.FromTeam.ID] = true
		affectedTeams[leg.ToTeam.ID] = true
	}

	// Increment trades_used for all affected teams
	for teamID := range affectedTeams {
		if _, err := tx.Exec(`
			UPDATE fantasy_teams SET trades_used = trades_used + 1 WHERE id = ?
		`, teamID); err != nil {
			return err
		}
	}

	// Update trade status
	if _, err := tx.Exec(`
		UPDATE trades SET status = 'approved', reviewed_by_team_id = ? WHERE id = ?
	`, reviewerTeamID, tradeID); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	log.Printf("[email deferred] Trade %d approved by team %d", tradeID, reviewerTeamID)
	return nil
}

// RejectTrade marks a trade as rejected with optional notes.
func (s *Store) RejectTrade(tradeID, reviewerTeamID int, notes string) error {
	var status string
	err := s.db.QueryRow(`SELECT status FROM trades WHERE id = ?`, tradeID).Scan(&status)
	if err != nil {
		if err == sql.ErrNoRows {
			return ErrTradeNotFound
		}
		return err
	}
	if status != "pending" {
		return ErrTradeNotPending
	}

	_, err = s.db.Exec(`
		UPDATE trades SET status = 'rejected', reviewed_by_team_id = ?, notes = ? WHERE id = ?
	`, reviewerTeamID, notes, tradeID)

	log.Printf("[email deferred] Trade %d rejected by team %d: %s", tradeID, reviewerTeamID, notes)
	return err
}

// GetPendingTrades returns all pending trades for a league.
func (s *Store) GetPendingTrades(leagueID int) ([]TradeDetail, error) {
	return s.queryTrades(leagueID, "pending")
}

// GetTradeHistory returns all trades (any status) for a league.
func (s *Store) GetTradeHistory(leagueID int) ([]TradeDetail, error) {
	return s.queryTrades(leagueID, "")
}

func (s *Store) queryTrades(leagueID int, status string) ([]TradeDetail, error) {
	query := `
		SELECT t.id, t.status, t.submitted_by_team_id,
		       ft.name, ft.manager, COALESCE(ft.cap_used,0),
		       COALESCE(ft.transactions_used,0), COALESCE(ft.trades_used,0),
		       COALESCE(t.notes,''), t.created_at
		FROM trades t
		JOIN fantasy_teams ft ON ft.id = t.submitted_by_team_id
		WHERE t.league_id = ?
	`
	args := []any{leagueID}
	if status != "" {
		query += " AND t.status = ?"
		args = append(args, status)
	}
	query += " ORDER BY t.created_at DESC"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var trades []TradeDetail
	for rows.Next() {
		var td TradeDetail
		if err := rows.Scan(
			&td.ID, &td.Status, &td.SubmittedByTeam.ID,
			&td.SubmittedByTeam.Name, &td.SubmittedByTeam.Manager,
			&td.SubmittedByTeam.CapUsed,
			&td.SubmittedByTeam.TransactionsUsed, &td.SubmittedByTeam.TradesUsed,
			&td.Notes, &td.CreatedAt,
		); err != nil {
			return nil, err
		}
		legs, err := s.tradeLegs(td.ID)
		if err != nil {
			return nil, err
		}
		td.Legs = legs
		trades = append(trades, td)
	}
	return trades, rows.Err()
}

func (s *Store) tradeLegs(tradeID int) ([]TradeLeg, error) {
	rows, err := s.db.Query(`
		SELECT tl.from_team_id, ft.name, ft.manager, COALESCE(ft.cap_used,0),
		       COALESCE(ft.transactions_used,0), COALESCE(ft.trades_used,0),
		       tl.to_team_id, tt.name, tt.manager, COALESCE(tt.cap_used,0),
		       COALESCE(tt.transactions_used,0), COALESCE(tt.trades_used,0),
		       tl.player_id, np.name, np.nhl_team_name, np.nhl_team_code,
		       np.position, np.salary_cap_hit, np.age
		FROM trade_legs tl
		JOIN fantasy_teams ft ON ft.id = tl.from_team_id
		JOIN fantasy_teams tt ON tt.id = tl.to_team_id
		JOIN nhl_players np ON np.id = tl.player_id
		WHERE tl.trade_id = ?
	`, tradeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var legs []TradeLeg
	for rows.Next() {
		var leg TradeLeg
		var salaryText, ageText string
		if err := rows.Scan(
			&leg.FromTeam.ID, &leg.FromTeam.Name, &leg.FromTeam.Manager,
			&leg.FromTeam.CapUsed, &leg.FromTeam.TransactionsUsed, &leg.FromTeam.TradesUsed,
			&leg.ToTeam.ID, &leg.ToTeam.Name, &leg.ToTeam.Manager,
			&leg.ToTeam.CapUsed, &leg.ToTeam.TransactionsUsed, &leg.ToTeam.TradesUsed,
			&leg.Player.ID, &leg.Player.Name, &leg.Player.NhlTeam, &leg.Player.NhlTeamCode,
			&leg.Player.Position, &salaryText, &ageText,
		); err != nil {
			return nil, err
		}
		leg.Player.Salary = parseSalary(salaryText)
		leg.Player.Age = parseAge(ageText)
		legs = append(legs, leg)
	}
	return legs, rows.Err()
}
