package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"hockey_season/backend/store"
)

type StandingsHandler struct {
	store  *store.Store
	secret string
}

func NewStandings(s *store.Store, secret string) *StandingsHandler {
	return &StandingsHandler{store: s, secret: secret}
}

func (h *StandingsHandler) authorise(w http.ResponseWriter, r *http.Request) (int, bool) {
	teamID, err := ParseToken(ExtractToken(r), h.secret)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return 0, false
	}
	return teamID, true
}

// Get handles GET /api/leagues/{id}/standings
func (h *StandingsHandler) Get(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.authorise(w, r); !ok {
		return
	}
	leagueID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	aggregate, h2h, err := h.store.GetStandings(leagueID)
	if err != nil {
		http.Error(w, "standings query failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if aggregate == nil {
		aggregate = []store.TeamStanding{}
	}
	if h2h == nil {
		h2h = []store.TeamStanding{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"aggregate": aggregate,
		"h2h":       h2h,
	})
}
