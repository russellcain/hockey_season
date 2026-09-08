package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/robfig/cron/v3"

	"hockey_season/backend/email"
	"hockey_season/backend/handlers"
	"hockey_season/backend/hub"
	"hockey_season/backend/jobs"
	"hockey_season/backend/mockdata"
	"hockey_season/backend/store"
)

func main() {
	mock := flag.Bool("mock", false, "seed mock draft data on startup and clean it up on shutdown")
	dev := flag.Bool("dev", false, "enable dev-only endpoints (advance-week, etc.)")
	flag.Parse()

	secret := os.Getenv("DRAFT_SECRET")
	if secret == "" {
		log.Fatal("DRAFT_SECRET environment variable is required")
	}

	dbPath := getEnv("DB_PATH", filepath.Join("..", "data", "hockey_season.db"))
	migrationsDir := getEnv("MIGRATIONS_DIR", filepath.Join("..", "data", "migrations"))

	st, err := store.Open(dbPath, migrationsDir)
	if err != nil {
		log.Fatalf("open store: %v", err)
	}
	defer st.Close()

	var seedResult *mockdata.SeedResult
	if *mock {
		seedResult, err = mockdata.Seed(st.DB(), secret)
		if err != nil {
			log.Fatalf("mock seed: %v", err)
		}
	}

	// Email sender — real if RESEND_API_KEY is set, no-op otherwise.
	var mailer email.Sender
	resendKey := os.Getenv("RESEND_API_KEY")
	resendFrom := getEnv("RESEND_FROM", "Draftr <noreply@draftr.local>")
	if resendKey != "" {
		mailer = email.NewResend(resendKey, resendFrom)
		log.Printf("email: Resend enabled (from=%s)", resendFrom)
	} else {
		mailer = email.NewNoop()
		log.Println("email: no RESEND_API_KEY set — using noop sender")
	}

	h := hub.New()

	// Existing handlers
	authH := handlers.NewAuth(st, secret)
	draftH := handlers.NewDraft(st, h, secret)
	wsH := handlers.NewWS(h, secret)

	// New season handlers
	leagueH := handlers.NewLeague(st, secret)
	playersH := handlers.NewPlayers(st, secret)
	rosterH := handlers.NewRoster(st, secret)
	scheduleH := handlers.NewSchedule(st, secret)
	standingsH := handlers.NewStandings(st, secret)
	tradesH := handlers.NewTrades(st, secret)
	injuriesH := handlers.NewInjuries(st, secret)

	mux := http.NewServeMux()

	// Auth
	mux.HandleFunc("POST /api/auth/join", authH.Join)

	// Draft (existing)
	mux.HandleFunc("GET /api/draft/{id}", draftH.Get)
	mux.HandleFunc("POST /api/draft/{id}/pick", draftH.Pick)
	mux.HandleFunc("GET /ws/draft/{id}", wsH.Handle)

	// Leagues
	mux.HandleFunc("POST /api/leagues", leagueH.Create)
	mux.HandleFunc("GET /api/leagues/{id}", leagueH.Get)
	mux.HandleFunc("PATCH /api/leagues/{id}/status", leagueH.UpdateStatus)

	// Players
	mux.HandleFunc("GET /api/leagues/{id}/players", playersH.List)

	// Roster & transactions
	mux.HandleFunc("GET /api/leagues/{id}/teams/{teamId}/roster", rosterH.GetRoster)
	mux.HandleFunc("POST /api/leagues/{id}/teams/{teamId}/transactions", rosterH.MakeTransaction)
	mux.HandleFunc("POST /api/leagues/{id}/teams/{teamId}/injury-subs", rosterH.MakeInjurySub)
	mux.HandleFunc("GET /api/leagues/{id}/teams/{teamId}/injury-subs/available", rosterH.GetEligibleSubs)
	mux.HandleFunc("POST /api/leagues/{id}/teams/{teamId}/cut", rosterH.CutPlayer)

	// Transaction log (league-wide, read-only for all members)
	mux.HandleFunc("GET /api/leagues/{id}/transactions", rosterH.GetTransactionLog)

	// Schedule
	mux.HandleFunc("POST /api/leagues/{id}/schedule/generate", scheduleH.Generate)
	mux.HandleFunc("GET /api/leagues/{id}/schedule", scheduleH.GetAll)
	mux.HandleFunc("GET /api/leagues/{id}/schedule/week/{n}", scheduleH.GetWeek)

	// Standings
	mux.HandleFunc("GET /api/leagues/{id}/standings", standingsH.Get)

	// Trades
	mux.HandleFunc("POST /api/leagues/{id}/trades", tradesH.Propose)
	mux.HandleFunc("GET /api/leagues/{id}/trades", tradesH.List)
	mux.HandleFunc("POST /api/trades/{id}/review", tradesH.Review)

	// Injuries
	mux.HandleFunc("GET /api/leagues/{id}/injuries", injuriesH.GetInjuries)

	// Team emails (commissioner only)
	mux.HandleFunc("GET /api/leagues/{id}/teams", leagueH.GetTeams)
	mux.HandleFunc("PATCH /api/leagues/{id}/teams/{teamId}/email", leagueH.UpdateTeamEmail)

	// Dev-only: advance simulated season one week at a time.
	if *dev {
		devH := handlers.NewDev(st)
		mux.HandleFunc("POST /api/dev/leagues/{id}/advance-week", devH.AdvanceWeek)
		log.Println("dev mode: POST /api/dev/leagues/{id}/advance-week enabled")
	}

	// Cron jobs
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		log.Printf("could not load America/New_York timezone, falling back to UTC: %v", err)
		loc = time.UTC
	}
	c := cron.New(cron.WithLocation(loc))
	c.AddFunc("0 2 * * *", func() { jobs.SyncStats(st, mailer) })     // 2AM ET daily
	c.AddFunc("0 23 * * *", func() { jobs.DigestInjuries(st, mailer) }) // 11PM ET daily
	c.Start()
	defer c.Stop()

	srv := &http.Server{
		Addr:         ":8080",
		Handler:      corsMiddleware(mux),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		log.Println("draft server listening on :8080")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("shutdown: %v", err)
	}

	if *mock {
		mockdata.Cleanup(st.DB(), seedResult)
	}
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin == "" {
			origin = "*"
		}
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
