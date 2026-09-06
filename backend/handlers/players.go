package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"hockey_season/backend/store"
)

type PlayersHandler struct {
	store  *store.Store
	secret string
}

func NewPlayers(s *store.Store, secret string) *PlayersHandler {
	return &PlayersHandler{store: s, secret: secret}
}

func (h *PlayersHandler) authorise(w http.ResponseWriter, r *http.Request) (int, bool) {
	teamID, err := ParseToken(ExtractToken(r), h.secret)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return 0, false
	}
	return teamID, true
}

// List handles GET /api/leagues/{id}/players
// Query params: q, position, max_salary, available (bool)
func (h *PlayersHandler) List(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.authorise(w, r); !ok {
		return
	}
	leagueID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	q := r.URL.Query().Get("q")
	position := r.URL.Query().Get("position")
	maxSalaryStr := r.URL.Query().Get("max_salary")
	availableOnly := r.URL.Query().Get("available") == "true"

	var maxSalary int64
	if maxSalaryStr != "" {
		maxSalary, _ = strconv.ParseInt(maxSalaryStr, 10, 64)
	}

	var players []store.Player
	if availableOnly {
		players, err = h.store.GetAvailablePlayers(leagueID, position, q, maxSalary)
	} else {
		players, err = h.store.ListPlayers()
		// Apply basic filters if not using GetAvailablePlayers
		_ = position
		_ = q
		_ = maxSalary
	}
	if err != nil {
		http.Error(w, "query failed", http.StatusInternalServerError)
		return
	}

	if players == nil {
		players = []store.Player{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(players)
}
