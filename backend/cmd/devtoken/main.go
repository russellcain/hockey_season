// Package main prints browser-ready localStorage snippets for every team in the
// database. Use this in development to skip the Join flow when the DB was
// seeded by the simulator (which stores raw code_hash values, not HMAC'd ones).
//
// Usage (from backend/):
//
//	DRAFT_SECRET=dev go run ./cmd/devtoken [--db ../data/hockey_season.db]
//
// Output is a JS snippet you can paste into the browser console to log in as
// any team.
package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	dbPath := flag.String("db", "../data/hockey_season.db", "path to SQLite DB")
	flag.Parse()

	secret := os.Getenv("DRAFT_SECRET")
	if secret == "" {
		log.Fatal("DRAFT_SECRET env var is required")
	}

	db, err := sql.Open("sqlite3", *dbPath+"?_journal_mode=WAL")
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()

	rows, err := db.Query(`
		SELECT ft.id, ft.name, ft.manager, ft.league_id,
		       COALESCE((SELECT id FROM draft_sessions WHERE league_id = ft.league_id ORDER BY id DESC LIMIT 1), 0)
		FROM fantasy_teams ft
		ORDER BY ft.league_id, ft.id
	`)
	if err != nil {
		log.Fatalf("query teams: %v", err)
	}
	defer rows.Close()

	fmt.Println("=== Draftr Dev Tokens ===")
	fmt.Println("Paste any snippet into the browser console, then navigate to the app.\n")

	for rows.Next() {
		var teamID, leagueID, draftID int
		var name, manager string
		if err := rows.Scan(&teamID, &name, &manager, &leagueID, &draftID); err != nil {
			log.Printf("scan: %v", err)
			continue
		}
		token := signToken(teamID, secret)
		fmt.Printf("── Team %d: %s (%s) · League %d\n", teamID, name, manager, leagueID)
		fmt.Printf("   localStorage.setItem('draft_token','%s'); localStorage.setItem('team_id','%d'); localStorage.setItem('league_id','%d'); localStorage.setItem('draft_id','%d'); location.href='/standings';\n\n",
			token, teamID, leagueID, draftID)
	}
}

func signToken(teamID int, secret string) string {
	expiry := time.Now().Add(24 * time.Hour).Unix()
	payload := fmt.Sprintf("%d:%d", teamID, expiry)
	sig := computeHMAC(payload, secret)
	return fmt.Sprintf("%s:%s", payload, sig)
}

func computeHMAC(data, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(data))
	return hex.EncodeToString(mac.Sum(nil))
}
