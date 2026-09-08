package store

import (
	"database/sql"
	"fmt"
	"log"
)

// FlagPlayerInjured marks a player as injured in injury_flags and updates their roster slot.
func (s *Store) FlagPlayerInjured(leagueID, playerID int, isLTIR bool) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	ltirVal := 0
	if isLTIR {
		ltirVal = 1
	}

	// Upsert injury flag
	if _, err := tx.Exec(`
		INSERT INTO injury_flags (player_id, is_ltir)
		VALUES (?, ?)
		ON CONFLICT DO NOTHING
	`, playerID, ltirVal); err != nil {
		return fmt.Errorf("flag injury: %w", err)
	}

	// Mark roster slot as injured for any team in this league
	if _, err := tx.Exec(`
		UPDATE roster_slots SET slot_type = 'injured'
		WHERE league_id = ? AND player_id = ? AND slot_type = 'active'
	`, leagueID, playerID); err != nil {
		return fmt.Errorf("update roster slot: %w", err)
	}

	log.Printf("[email deferred] Injury alert: player %d flagged as injured (league %d, ltir=%v)",
		playerID, leagueID, isLTIR)

	return tx.Commit()
}

// ResolvePlayerInjury clears the injury flag and restores the player's roster slot to active.
func (s *Store) ResolvePlayerInjury(leagueID, playerID int) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Resolve injury flag
	if _, err := tx.Exec(`
		UPDATE injury_flags SET resolved_at = CURRENT_TIMESTAMP
		WHERE player_id = ? AND resolved_at IS NULL
	`, playerID); err != nil {
		return fmt.Errorf("resolve flag: %w", err)
	}

	// Restore roster slot to active
	if _, err := tx.Exec(`
		UPDATE roster_slots SET slot_type = 'active'
		WHERE league_id = ? AND player_id = ? AND slot_type = 'injured'
	`, leagueID, playerID); err != nil {
		return fmt.Errorf("restore slot: %w", err)
	}

	// Remove any substitute for this player (returned to unclaimed pool)
	if _, err := tx.Exec(`
		DELETE FROM roster_slots
		WHERE league_id = ? AND slot_type = 'substitute' AND original_player_id = ?
	`, leagueID, playerID); err != nil {
		return fmt.Errorf("remove sub: %w", err)
	}

	log.Printf("[email deferred] Injury resolved: player %d back from injury (league %d)", playerID, leagueID)

	return tx.Commit()
}

// GetActiveInjuries returns all active injuries in a league with team and sub info.
func (s *Store) GetActiveInjuries(leagueID int) ([]InjuryInfo, error) {
	rows, err := s.db.Query(`
		SELECT rs.team_id,
		       np.id, np.name, np.nhl_team_name, np.nhl_team_code, np.position, np.salary_cap_hit, np.age,
		       rs.original_player_id
		FROM roster_slots rs
		JOIN nhl_players np ON np.id = rs.player_id
		WHERE rs.league_id = ? AND rs.slot_type = 'injured'
		ORDER BY np.name
	`, leagueID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var injuries []InjuryInfo
	for rows.Next() {
		var info InjuryInfo
		var origID sql.NullInt64
		var salaryText, ageText string
		if err := rows.Scan(
			&info.TeamID,
			&info.InjuredPlayer.ID, &info.InjuredPlayer.Name,
			&info.InjuredPlayer.NhlTeam, &info.InjuredPlayer.NhlTeamCode,
			&info.InjuredPlayer.Position, &salaryText, &ageText,
			&origID,
		); err != nil {
			return nil, err
		}
		info.InjuredPlayer.Salary = parseSalary(salaryText)
		info.InjuredPlayer.Age = parseAge(ageText)
		info.CapCeiling = info.InjuredPlayer.Salary

		// Find substitute if any
		sub, err := s.GetSubForInjured(leagueID, info.TeamID, info.InjuredPlayer.ID)
		if err == nil {
			info.SubstitutePlayer = &sub.Player
		}

		injuries = append(injuries, info)
	}
	return injuries, rows.Err()
}

// GetEligibleSubs returns unclaimed players eligible as substitutes for an injured player.
// Eligibility: same position, salary ≤ root injured player's cap hit.
func (s *Store) GetEligibleSubs(leagueID, teamID, injuredPlayerID int) ([]Player, error) {
	injSlot, err := s.GetRosterSlot(leagueID, teamID, injuredPlayerID)
	if err != nil {
		return nil, ErrNotOnRoster
	}

	rootSalary, err := s.getRootPlayerSalary(leagueID, teamID, injSlot)
	if err != nil {
		return nil, err
	}

	return s.GetAvailablePlayers(leagueID, injSlot.Player.Position, "", rootSalary)
}
