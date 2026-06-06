// Seed script — run once per season from the draft/ directory:
//
//	go run ./seed
//
// Reads ../data/player-salaries.json, fetches NHL roster IDs from the NHL API
// (one request per team), and hydrates ../data/hockey_season.db.
// Runs the migration in ../data/migrations/001_create_nhl_players.sql first,
// so the script is safe to re-run; existing rows are skipped via INSERT OR IGNORE.
package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

// Update this each season (e.g. 20262027 for the 2026-27 season).
const nhlSeason = "20252026"

const (
	dbPath        = "../data/hockey_season.db"
	salaryPath    = "../data/player-salaries.json"
	migrationPath = "../data/migrations/001_create_nhl_players.sql"
)

// teamTricodes maps capwages team names (as formatted by the salary scraper)
// to the NHL API tricode used to fetch rosters.
var teamTricodes = map[string]string{
	"Anaheim Ducks":         "ANA",
	"Boston Bruins":         "BOS",
	"Buffalo Sabres":        "BUF",
	"Calgary Flames":        "CGY",
	"Carolina Hurricanes":   "CAR",
	"Chicago Blackhawks":    "CHI",
	"Colorado Avalanche":    "COL",
	"Columbus Blue Jackets": "CBJ",
	"Dallas Stars":          "DAL",
	"Detroit Red Wings":     "DET",
	"Edmonton Oilers":       "EDM",
	"Florida Panthers":      "FLA",
	"Los Angeles Kings":     "LAK",
	"Minnesota Wild":        "MIN",
	"Montreal Canadiens":    "MTL",
	"Nashville Predators":   "NSH",
	"New Jersey Devils":     "NJD",
	"New York Islanders":    "NYI",
	"New York Rangers":      "NYR",
	"Ottawa Senators":       "OTT",
	"Philadelphia Flyers":   "PHI",
	"Pittsburgh Penguins":   "PIT",
	"San Jose Sharks":       "SJS",
	"Seattle Kraken":        "SEA",
	"St Louis Blues":        "STL",
	"Tampa Bay Lightning":   "TBL",
	"Toronto Maple Leafs":   "TOR",
	"Utah Mammoth":          "UTA",
	"Vancouver Canucks":     "VAN",
	"Vegas Golden Knights":  "VGK",
	"Washington Capitals":   "WSH",
	"Winnipeg Jets":         "WPG",
}

type salaryEntry struct {
	Name         string `json:"name"`
	Age          string `json:"age"`
	Pos          string `json:"pos"`
	SalaryCapHit string `json:"salary_cap_hit"`
	Team         string `json:"team"`
}

type localeName struct {
	Default string `json:"default"`
}
type rosterPlayer struct {
	ID        int64      `json:"id"`
	FirstName localeName `json:"firstName"`
	LastName  localeName `json:"lastName"`
}
type rosterResponse struct {
	Forwards   []rosterPlayer `json:"forwards"`
	Defensemen []rosterPlayer `json:"defensemen"`
	Goalies    []rosterPlayer `json:"goalies"`
}

func normName(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

// fetchNHLIDs returns a normalized-name → NHL player ID map for one team.
func fetchNHLIDs(tricode string) (map[string]int64, error) {
	url := fmt.Sprintf("https://api-web.nhle.com/v1/roster/%s/%s", tricode, nhlSeason)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var roster rosterResponse
	if err := json.Unmarshal(body, &roster); err != nil {
		return nil, err
	}
	ids := make(map[string]int64)
	all := append(append(roster.Forwards, roster.Defensemen...), roster.Goalies...)
	for _, p := range all {
		key := normName(p.FirstName.Default + " " + p.LastName.Default)
		ids[key] = p.ID
	}
	return ids, nil
}

func main() {
	migration, err := os.ReadFile(migrationPath)
	if err != nil {
		log.Fatal("could not read migration:", err)
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if _, err := db.Exec(string(migration)); err != nil {
		log.Fatal("migration failed:", err)
	}

	data, err := os.ReadFile(salaryPath)
	if err != nil {
		log.Fatal(err)
	}
	var players []salaryEntry
	if err := json.Unmarshal(data, &players); err != nil {
		log.Fatal(err)
	}

	// One NHL API call per team, cached for the rest of that team's players.
	rosterCache := make(map[string]map[string]int64)
	for _, p := range players {
		tricode, ok := teamTricodes[p.Team]
		if !ok || tricode == "" {
			continue
		}
		if _, already := rosterCache[tricode]; already {
			continue
		}
		fmt.Printf("fetching roster: %s (%s)\n", p.Team, tricode)
		ids, err := fetchNHLIDs(tricode)
		if err != nil {
			log.Printf("warning: roster fetch failed for %s: %v", tricode, err)
			ids = map[string]int64{}
		}
		rosterCache[tricode] = ids
	}

	stmt, err := db.Prepare(`
		INSERT OR IGNORE INTO nhl_players
			(nhl_id, name, nhl_team_name, nhl_team_code, position, salary_cap_hit, age)
		VALUES (?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()

	var inserted, skipped, unmatched int
	for _, p := range players {
		tricode := teamTricodes[p.Team]
		roster := rosterCache[tricode]

		var nhlID interface{} // nil → stored as SQL NULL
		if id := roster[normName(p.Name)]; id != 0 {
			nhlID = id
		} else {
			unmatched++
			log.Printf("unmatched (nhl_id will be null): %s — %s", p.Name, p.Team)
		}

		res, err := stmt.Exec(nhlID, p.Name, p.Team, tricode, p.Pos, p.SalaryCapHit, p.Age)
		if err != nil {
			log.Printf("insert error for %s: %v", p.Name, err)
			continue
		}
		if rows, _ := res.RowsAffected(); rows == 0 {
			skipped++ // already exists, INSERT OR IGNORE no-op'd
		} else {
			inserted++
		}
	}

	fmt.Printf("\ndone: %d inserted, %d skipped (already existed), %d unmatched (nhl_id is null)\n",
		inserted, skipped, unmatched)
}
