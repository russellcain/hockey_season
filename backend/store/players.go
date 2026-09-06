package store

import (
	"strconv"
	"strings"
)

func (s *Store) ListPlayers() ([]Player, error) {
	rows, err := s.db.Query(`
		SELECT id, name, nhl_team_name, nhl_team_code, position, salary_cap_hit, age
		FROM nhl_players
		ORDER BY name ASC
	`)
	if err != nil {
		return nil, err
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

func (s *Store) GetPlayer(id int) (*Player, error) {
	var p Player
	var salaryText, ageText string
	err := s.db.QueryRow(`
		SELECT id, name, nhl_team_name, nhl_team_code, position, salary_cap_hit, age
		FROM nhl_players WHERE id = ?
	`, id).Scan(&p.ID, &p.Name, &p.NhlTeam, &p.NhlTeamCode, &p.Position, &salaryText, &ageText)
	if err != nil {
		return nil, err
	}
	p.Salary = parseSalary(salaryText)
	p.Age = parseAge(ageText)
	return &p, nil
}

func parseSalary(s string) int64 {
	s = strings.ReplaceAll(s, ",", "")
	n, _ := strconv.ParseInt(s, 10, 64)
	return n
}

func parseAge(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}
