package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"hockey_season/backend/store"
)

type InjuriesHandler struct {
	store  *store.Store
	secret string
}

func NewInjuries(s *store.Store, secret string) *InjuriesHandler {
	return &InjuriesHandler{store: s, secret: secret}
}

func (h *InjuriesHandler) authorise(w http.ResponseWriter, r *http.Request) (int, bool) {
	teamID, err := ParseToken(ExtractToken(r), h.secret)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return 0, false
	}
	return teamID, true
}

// GetInjuries handles GET /api/leagues/{id}/injuries
func (h *InjuriesHandler) GetInjuries(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.authorise(w, r); !ok {
		return
	}
	leagueID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	injuries, err := h.store.GetActiveInjuries(leagueID)
	if err != nil {
		http.Error(w, "query failed", http.StatusInternalServerError)
		return
	}
	if injuries == nil {
		injuries = []store.InjuryInfo{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(injuries)
}
