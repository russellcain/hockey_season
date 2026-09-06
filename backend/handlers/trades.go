package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"hockey_season/backend/store"
)

type TradesHandler struct {
	store  *store.Store
	secret string
}

func NewTrades(s *store.Store, secret string) *TradesHandler {
	return &TradesHandler{store: s, secret: secret}
}

func (h *TradesHandler) authorise(w http.ResponseWriter, r *http.Request) (int, bool) {
	teamID, err := ParseToken(ExtractToken(r), h.secret)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return 0, false
	}
	return teamID, true
}

// Propose handles POST /api/leagues/{id}/trades
func (h *TradesHandler) Propose(w http.ResponseWriter, r *http.Request) {
	callerTeamID, ok := h.authorise(w, r)
	if !ok {
		return
	}
	leagueID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	var req struct {
		ToTeamID      int   `json:"toTeamId"`
		FromPlayerIDs []int `json:"fromPlayerIds"`
		ToPlayerIDs   []int `json:"toPlayerIds"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ToTeamID == 0 {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	tradeID, err := h.store.ProposeTrade(leagueID, callerTeamID, req.ToTeamID, req.FromPlayerIDs, req.ToPlayerIDs)
	if err != nil {
		code := http.StatusBadRequest
		if errors.Is(err, store.ErrTradeLimitReached) {
			code = http.StatusForbidden
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(code)
		json.NewEncoder(w).Encode(map[string]any{"success": false, "error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]any{"success": true, "tradeId": tradeID})
}

// List handles GET /api/leagues/{id}/trades
func (h *TradesHandler) List(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.authorise(w, r); !ok {
		return
	}
	leagueID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	trades, err := h.store.GetTradeHistory(leagueID)
	if err != nil {
		http.Error(w, "query failed", http.StatusInternalServerError)
		return
	}
	if trades == nil {
		trades = []store.TradeDetail{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(trades)
}

// Review handles POST /api/trades/{id}/review
// Commissioner only for now.
func (h *TradesHandler) Review(w http.ResponseWriter, r *http.Request) {
	callerTeamID, ok := h.authorise(w, r)
	if !ok {
		return
	}
	commID, err := h.store.GetCommissionerTeamID()
	if err != nil || callerTeamID != commID {
		http.Error(w, "forbidden: commissioner only", http.StatusForbidden)
		return
	}

	tradeID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	var req struct {
		Decision string `json:"decision"`
		Notes    string `json:"notes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}

	switch req.Decision {
	case "approved":
		err = h.store.ApproveTrade(tradeID, callerTeamID)
	case "rejected":
		err = h.store.RejectTrade(tradeID, callerTeamID, req.Notes)
	default:
		http.Error(w, "decision must be 'approved' or 'rejected'", http.StatusBadRequest)
		return
	}

	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]any{"success": false, "error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"success": true})
}
