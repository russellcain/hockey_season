package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"hockey_season/backend/store"
)

type LeagueHandler struct {
	store  *store.Store
	secret string
}

func NewLeague(s *store.Store, secret string) *LeagueHandler {
	return &LeagueHandler{store: s, secret: secret}
}

func (h *LeagueHandler) authorise(w http.ResponseWriter, r *http.Request) (int, bool) {
	teamID, err := ParseToken(ExtractToken(r), h.secret)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return 0, false
	}
	return teamID, true
}

// Create handles POST /api/leagues
func (h *LeagueHandler) Create(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.authorise(w, r); !ok {
		return
	}
	var req struct {
		Name      string `json:"name"`
		SalaryCap int64  `json:"salaryCap"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
		http.Error(w, "missing name", http.StatusBadRequest)
		return
	}
	if req.SalaryCap == 0 {
		req.SalaryCap = 104_000_000
	}
	league, err := h.store.CreateLeague(req.Name, req.SalaryCap)
	if err != nil {
		http.Error(w, "create failed", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(league)
}

// Get handles GET /api/leagues/{id}
func (h *LeagueHandler) Get(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.authorise(w, r); !ok {
		return
	}
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	league, err := h.store.GetLeague(id)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(league)
}

// UpdateStatus handles PATCH /api/leagues/{id}/status
// Commissioner only (lowest team ID).
func (h *LeagueHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	callerTeamID, ok := h.authorise(w, r)
	if !ok {
		return
	}
	commID, err := h.store.GetCommissionerTeamID()
	if err != nil || callerTeamID != commID {
		http.Error(w, "forbidden: commissioner only", http.StatusForbidden)
		return
	}

	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	var req struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Status == "" {
		http.Error(w, "missing status", http.StatusBadRequest)
		return
	}

	validStatuses := map[string]bool{"setup": true, "draft_ready": true, "drafting": true, "in_season": true, "complete": true}
	if !validStatuses[req.Status] {
		http.Error(w, "invalid status", http.StatusBadRequest)
		return
	}

	if err := h.store.UpdateLeagueStatus(id, req.Status); err != nil {
		http.Error(w, "update failed", http.StatusInternalServerError)
		return
	}
	league, _ := h.store.GetLeague(id)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(league)
}
