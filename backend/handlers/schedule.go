package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"hockey_season/backend/store"
)

type ScheduleHandler struct {
	store  *store.Store
	secret string
}

func NewSchedule(s *store.Store, secret string) *ScheduleHandler {
	return &ScheduleHandler{store: s, secret: secret}
}

func (h *ScheduleHandler) authorise(w http.ResponseWriter, r *http.Request) (int, bool) {
	teamID, err := ParseToken(ExtractToken(r), h.secret)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return 0, false
	}
	return teamID, true
}

// Generate handles POST /api/leagues/{id}/schedule/generate (commissioner only)
func (h *ScheduleHandler) Generate(w http.ResponseWriter, r *http.Request) {
	callerTeamID, ok := h.authorise(w, r)
	if !ok {
		return
	}
	commID, err := h.store.GetCommissionerTeamID()
	if err != nil || callerTeamID != commID {
		http.Error(w, "forbidden: commissioner only", http.StatusForbidden)
		return
	}

	leagueID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	if err := h.store.GenerateH2HSchedule(leagueID); err != nil {
		if errors.Is(err, store.ErrScheduleExists) {
			http.Error(w, "schedule already generated", http.StatusConflict)
			return
		}
		http.Error(w, "generate failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	matchups, _ := h.store.GetMatchups(leagueID)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(matchups)
}

// GetAll handles GET /api/leagues/{id}/schedule
func (h *ScheduleHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.authorise(w, r); !ok {
		return
	}
	leagueID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	matchups, err := h.store.GetMatchups(leagueID)
	if err != nil {
		http.Error(w, "query failed", http.StatusInternalServerError)
		return
	}
	if matchups == nil {
		matchups = []store.Matchup{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(matchups)
}

// GetWeek handles GET /api/leagues/{id}/schedule/week/{n}
func (h *ScheduleHandler) GetWeek(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.authorise(w, r); !ok {
		return
	}
	leagueID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	weekNum, err := strconv.Atoi(r.PathValue("n"))
	if err != nil {
		http.Error(w, "invalid week", http.StatusBadRequest)
		return
	}

	matchups, err := h.store.GetWeekMatchups(leagueID, weekNum)
	if err != nil {
		http.Error(w, "query failed", http.StatusInternalServerError)
		return
	}
	if matchups == nil {
		matchups = []store.Matchup{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(matchups)
}
