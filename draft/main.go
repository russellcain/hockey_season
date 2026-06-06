package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/gocolly/colly"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type Player struct {
	Name         string `json:"name"`
	Age          string `json:"age"`
	Pos          string `json:"pos"`
	SalaryCapHit string `json:"salary_cap_hit"`
	Team         string `json:"team"`
}

var teamPages = []string{
	"anaheim_ducks",
	"boston_bruins",
	"buffalo_sabres",
	"calgary_flames",
	"carolina_hurricanes",
	"chicago_blackhawks",
	"colorado_avalanche",
	"columbus_blue_jackets",
	"dallas_stars",
	"detroit_red_wings",
	"edmonton_oilers",
	"florida_panthers",
	"los_angeles_kings",
	"minnesota_wild",
	"montreal_canadiens",
	"nashville_predators",
	"new_jersey_devils",
	"new_york_islanders",
	"new_york_rangers",
	"ottawa_senators",
	"philadelphia_flyers",
	"pittsburgh_penguins",
	"san_jose_sharks",
	"seattle_kraken",
	"st_louis_blues",
	"tampa_bay_lightning",
	"toronto_maple_leafs",
	"utah_mammoth",
	"vancouver_canucks",
	"vegas_golden_knights",
	"washington_capitals",
	"winnipeg_jets",
}

const capwagesDomain = "capwages.com"

func teamSalaryURL(team string) string {
	return fmt.Sprintf("https://%s/teams/%s", capwagesDomain, team)
}

func formatTeamName(raw string) string {
	return cases.Title(language.English).String(strings.ReplaceAll(raw, "_", " "))
}

// capwages stores names as "Last, First" — invert to "First Last"
func formatPlayerName(raw string) string {
	if strings.Contains(raw, ",") {
		parts := strings.Split(raw, ", ")
		return fmt.Sprintf("%s %s", parts[1], parts[0])
	}
	return raw
}

// collapse multi-position strings to F, D, or G
func simplifyPosition(raw string) string {
	switch {
	case strings.Contains(raw, "G"):
		return "G"
	case strings.Contains(raw, "D"):
		return "D"
	default:
		return "F"
	}
}

func main() {
	c := colly.NewCollector()

	var players []Player
	var currentTeam string

	c.OnHTML("table.teamProfileRosterSection__table tbody tr", func(e *colly.HTMLElement) {
		if e.ChildText("td:nth-child(5)") == "Retired" {
			return
		}
		salaryText := e.ChildText("td:nth-child(10) div")
		parts := strings.Split(salaryText, "$")
		// parts[0] is the prefix (empty, "RFA", or "UFA"); parts[1] is the dollar amount
		if len(parts) < 2 || parts[0] == "RFA" || parts[0] == "UFA" {
			return
		}
		players = append(players, Player{
			Name:         formatPlayerName(e.ChildText("td:nth-child(1) a")),
			Age:          e.ChildText("td:nth-child(7)"),
			Pos:          simplifyPosition(e.ChildText("td:nth-child(4)")),
			Team:         formatTeamName(currentTeam),
			SalaryCapHit: parts[1],
		})
	})

	c.OnRequest(func(r *colly.Request) {
		fmt.Println("visiting", r.URL.String())
	})

	for _, team := range teamPages {
		currentTeam = team
		if err := c.Visit(teamSalaryURL(team)); err != nil {
			log.Fatal(err)
		}
	}

	out := "player-salaries.json"
	data, _ := json.MarshalIndent(players, "", "  ")
	if err := os.WriteFile(out, data, 0644); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("wrote %d players to %s\n", len(players), out)
}
