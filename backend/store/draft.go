package store

import (
	"database/sql"
	"errors"
	"fmt"

	"hockey_season/backend/snake"
)

// FinaliseDraft is called once when a draft session transitions to "complete".
// It idempotently:
//  1. Copies all draft_picks for the session into roster_slots (slot_type='active')
//  2. Generates the H2H matchup schedule for the league
//  3. Advances the league status to 'in_season'
//
// Returns an error if the session has no associated league_id (i.e. the draft
// was started without linking it to a league — nothing is written in that case).
func (s *Store) FinaliseDraft(sessionID int) error {
	var leagueID sql.NullInt64
	if err := s.db.QueryRow(
		`SELECT league_id FROM draft_sessions WHERE id = ?`, sessionID,
	).Scan(&leagueID); err != nil {
		return fmt.Errorf("FinaliseDraft: get session: %w", err)
	}
	if !leagueID.Valid || leagueID.Int64 == 0 {
		return fmt.Errorf("FinaliseDraft: session %d has no league_id — skipping season setup", sessionID)
	}
	lid := int(leagueID.Int64)

	// Populate roster_slots from draft_picks. INSERT OR IGNORE makes this safe
	// to call more than once (e.g. if the server restarts mid-finalise).
	_, err := s.db.Exec(`
		INSERT OR IGNORE INTO roster_slots (team_id, player_id, league_id, slot_type)
		SELECT dp.team_id, dp.player_id, ?, 'active'
		FROM draft_picks dp
		WHERE dp.session_id = ?
	`, lid, sessionID)
	if err != nil {
		return fmt.Errorf("FinaliseDraft: populate roster_slots: %w", err)
	}

	// Generate H2H schedule. ErrScheduleExists means it already ran — fine.
	if err := s.GenerateH2HSchedule(lid); err != nil && !errors.Is(err, ErrScheduleExists) {
		return fmt.Errorf("FinaliseDraft: generate schedule: %w", err)
	}

	// Advance the league to in_season.
	if err := s.UpdateLeagueStatus(lid, "in_season"); err != nil {
		return fmt.Errorf("FinaliseDraft: update league status: %w", err)
	}

	return nil
}

var (
	ErrNotYourTurn    = errors.New("not your turn")
	ErrAlreadyDrafted = errors.New("player already drafted")
	ErrOverCap        = errors.New("over cap")
	ErrPositionFull   = errors.New("position full")
	ErrDraftNotActive = errors.New("draft not active")
)

func (s *Store) GetDraftFull(sessionID, myTeamID int) (*DraftFull, error) {
	var ds DraftState
	var capLimit int64
	err := s.db.QueryRow(`
		SELECT id, status, total_rounds, total_teams, current_round, current_pick, seconds_per_pick, cap_limit
		FROM draft_sessions WHERE id = ?
	`, sessionID).Scan(
		&ds.ID, &ds.Status, &ds.TotalRounds, &ds.TotalTeams,
		&ds.CurrentRound, &ds.CurrentPick, &ds.SecondsPerPick, &capLimit,
	)
	if err != nil {
		return nil, fmt.Errorf("get session: %w", err)
	}

	teams, err := s.teamsWithPicks(sessionID, ds.TotalRounds)
	if err != nil {
		return nil, err
	}
	for i := range teams {
		teams[i].IsMe = teams[i].ID == myTeamID
	}

	players, err := s.ListPlayers()
	if err != nil {
		return nil, fmt.Errorf("list players: %w", err)
	}

	return &DraftFull{
		DraftState: ds,
		Teams:      teams,
		Players:    players,
		Config: DraftConfig{
			CapLimit:    capLimit,
			SlotTargets: DefaultSlotTargets,
		},
		MyTeamID: myTeamID,
	}, nil
}

func (s *Store) MakePick(sessionID, callerTeamID, playerID int) (*PickResult, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var ds DraftState
	var capLimit int64
	err = tx.QueryRow(`
		SELECT id, status, total_rounds, total_teams, current_round, current_pick, seconds_per_pick, cap_limit
		FROM draft_sessions WHERE id = ?
	`, sessionID).Scan(
		&ds.ID, &ds.Status, &ds.TotalRounds, &ds.TotalTeams,
		&ds.CurrentRound, &ds.CurrentPick, &ds.SecondsPerPick, &capLimit,
	)
	if err != nil {
		return nil, fmt.Errorf("get session: %w", err)
	}
	if ds.Status != "in_progress" {
		return nil, ErrDraftNotActive
	}

	teamIDs, err := scanTeamIDsFromTx(tx)
	if err != nil {
		return nil, err
	}
	expectedIdx := snake.TeamAt(ds.CurrentRound, ds.CurrentPick, ds.TotalTeams)
	if expectedIdx >= len(teamIDs) || teamIDs[expectedIdx] != callerTeamID {
		return nil, ErrNotYourTurn
	}

	var taken int
	tx.QueryRow(`SELECT COUNT(*) FROM draft_picks WHERE session_id = ? AND player_id = ?`,
		sessionID, playerID).Scan(&taken)
	if taken > 0 {
		return nil, ErrAlreadyDrafted
	}

	var player Player
	var salaryText, ageText string
	err = tx.QueryRow(`
		SELECT id, name, nhl_team_name, nhl_team_code, position, salary_cap_hit, age
		FROM nhl_players WHERE id = ?
	`, playerID).Scan(
		&player.ID, &player.Name, &player.NhlTeam, &player.NhlTeamCode,
		&player.Position, &salaryText, &ageText,
	)
	if err != nil {
		return nil, fmt.Errorf("get player: %w", err)
	}
	player.Salary = parseSalary(salaryText)
	player.Age = parseAge(ageText)

	var capUsed int64
	tx.QueryRow(`SELECT cap_used FROM fantasy_teams WHERE id = ?`, callerTeamID).Scan(&capUsed)
	if player.Salary > capLimit-capUsed {
		return nil, ErrOverCap
	}

	var posCount int
	tx.QueryRow(`
		SELECT COUNT(*) FROM draft_picks dp
		JOIN nhl_players np ON np.id = dp.player_id
		WHERE dp.session_id = ? AND dp.team_id = ? AND np.position = ?
	`, sessionID, callerTeamID, player.Position).Scan(&posCount)
	if posCount >= DefaultSlotTargets[player.Position] {
		return nil, ErrPositionFull
	}

	if _, err = tx.Exec(`
		INSERT INTO draft_picks (session_id, team_id, player_id, round, pick_number)
		VALUES (?, ?, ?, ?, ?)
	`, sessionID, callerTeamID, playerID, ds.CurrentRound, ds.CurrentPick); err != nil {
		return nil, fmt.Errorf("insert pick: %w", err)
	}

	if _, err = tx.Exec(`UPDATE fantasy_teams SET cap_used = cap_used + ? WHERE id = ?`,
		player.Salary, callerTeamID); err != nil {
		return nil, fmt.Errorf("update cap: %w", err)
	}

	nextRound, nextPick := advancePick(ds.CurrentRound, ds.CurrentPick, ds.TotalTeams, ds.TotalRounds)
	newStatus := "in_progress"
	if nextRound > ds.TotalRounds {
		newStatus = "complete"
	}
	if _, err = tx.Exec(`
		UPDATE draft_sessions SET current_round = ?, current_pick = ?, status = ? WHERE id = ?
	`, nextRound, nextPick, newStatus, sessionID); err != nil {
		return nil, fmt.Errorf("advance pick: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	updatedDS := ds
	updatedDS.CurrentRound = nextRound
	updatedDS.CurrentPick = nextPick
	updatedDS.Status = newStatus

	teams, err := s.teamsWithPicks(sessionID, ds.TotalRounds)
	if err != nil {
		return nil, err
	}

	return &PickResult{
		Player:     player,
		DraftState: updatedDS,
		Teams:      teams,
	}, nil
}

func (s *Store) teamsWithPicks(sessionID, totalRounds int) ([]Team, error) {
	teams, err := s.ListTeams()
	if err != nil {
		return nil, err
	}
	for i := range teams {
		picks, err := s.teamPicksForSession(sessionID, teams[i].ID, totalRounds)
		if err != nil {
			return nil, fmt.Errorf("picks for team %d: %w", teams[i].ID, err)
		}
		teams[i].Picks = picks
	}
	return teams, nil
}

func (s *Store) teamPicksForSession(sessionID, teamID, totalRounds int) ([]*Player, error) {
	picks := make([]*Player, totalRounds)

	rows, err := s.db.Query(`
		SELECT dp.round, np.id, np.name, np.nhl_team_name, np.nhl_team_code, np.position, np.salary_cap_hit, np.age
		FROM draft_picks dp
		JOIN nhl_players np ON np.id = dp.player_id
		WHERE dp.session_id = ? AND dp.team_id = ?
		ORDER BY dp.round
	`, sessionID, teamID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var round int
		var p Player
		var salaryText, ageText string
		if err := rows.Scan(&round, &p.ID, &p.Name, &p.NhlTeam, &p.NhlTeamCode, &p.Position, &salaryText, &ageText); err != nil {
			return nil, err
		}
		p.Salary = parseSalary(salaryText)
		p.Age = parseAge(ageText)
		if round >= 1 && round <= totalRounds {
			picks[round-1] = &p
		}
	}
	return picks, rows.Err()
}

func scanTeamIDsFromTx(tx *sql.Tx) ([]int, error) {
	rows, err := tx.Query(`SELECT id FROM fantasy_teams ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func advancePick(round, pick, totalTeams, totalRounds int) (nextRound, nextPick int) {
	pick++
	if pick > totalTeams {
		pick = 1
		round++
	}
	return round, pick
}
