package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"hockey_season/backend/store"
)

type RosterHandler struct {
	store  *store.Store
	secret string
}

func NewRoster(s *store.Store, secret string) *RosterHandler {
	return &RosterHandler{store: s, secret: secret}
}

func (h *RosterHandler) authorise(w http.ResponseWriter, r *http.Request) (int, bool) {
	teamID, err := ParseToken(ExtractToken(r), h.secret)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return 0, false
	}
	return teamID, true
}

func pathLeagueTeam(r *http.Request) (leagueID, teamID int, err error) {
	leagueID, err = strconv.Atoi(r.PathValue("id"))
	if err != nil {
		return
	}
	teamID, err = strconv.Atoi(r.PathValue("teamId"))
	return
}

// GetRoster handles GET /api/leagues/{id}/teams/{teamId}/roster
func (h *RosterHandler) GetRoster(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.authorise(w, r); !ok {
		return
	}
	leagueID, teamID, err := pathLeagueTeam(r)
	if err != nil {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}

	slots, err := h.store.GetRoster(leagueID, teamID)
	if err != nil {
		http.Error(w, "roster query failed", http.StatusInternalServerError)
		return
	}
	if slots == nil {
		slots = []store.RosterSlot{}
	}

	// Merge season stats into each player.
	if statsMap, err := h.store.GetRosterPlayerStats(leagueID, teamID); err == nil {
		for i := range slots {
			if st, ok := statsMap[slots[i].PlayerID]; ok {
				slots[i].Player.Stats = st
			}
		}
	}

	capUsed, _ := h.store.GetTeamCapUsed(leagueID, teamID)
	team, _ := h.store.GetTeam(teamID)
	league, _ := h.store.GetLeague(leagueID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"slots":            slots,
		"capUsed":          capUsed,
		"salaryCap":        league.SalaryCap,
		"transactionsUsed": team.TransactionsUsed,
		"tradesUsed":       team.TradesUsed,
	})
}

// MakeTransaction handles POST /api/leagues/{id}/teams/{teamId}/transactions
func (h *RosterHandler) MakeTransaction(w http.ResponseWriter, r *http.Request) {
	callerTeamID, ok := h.authorise(w, r)
	if !ok {
		return
	}
	leagueID, teamID, err := pathLeagueTeam(r)
	if err != nil || callerTeamID != teamID {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	var req struct {
		DropPlayerID int `json:"dropPlayerId"`
		AddPlayerID  int `json:"addPlayerId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.DropPlayerID == 0 || req.AddPlayerID == 0 {
		http.Error(w, "missing dropPlayerId or addPlayerId", http.StatusBadRequest)
		return
	}

	if err := h.store.ExecuteTransaction(leagueID, teamID, req.DropPlayerID, req.AddPlayerID); err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, store.ErrOverTransactions) {
			status = http.StatusForbidden
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		json.NewEncoder(w).Encode(map[string]any{"success": false, "error": err.Error()})
		return
	}

	slots, _ := h.store.GetRoster(leagueID, teamID)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"success": true, "roster": slots})
}

// MakeInjurySub handles POST /api/leagues/{id}/teams/{teamId}/injury-subs
func (h *RosterHandler) MakeInjurySub(w http.ResponseWriter, r *http.Request) {
	callerTeamID, ok := h.authorise(w, r)
	if !ok {
		return
	}
	leagueID, teamID, err := pathLeagueTeam(r)
	if err != nil || callerTeamID != teamID {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	var req struct {
		InjuredPlayerID int `json:"injuredPlayerId"`
		SubPlayerID     int `json:"subPlayerId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.InjuredPlayerID == 0 || req.SubPlayerID == 0 {
		http.Error(w, "missing injuredPlayerId or subPlayerId", http.StatusBadRequest)
		return
	}

	if err := h.store.ExecuteInjurySub(leagueID, teamID, req.InjuredPlayerID, req.SubPlayerID); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]any{"success": false, "error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"success": true})
}

// GetEligibleSubs handles GET /api/leagues/{id}/teams/{teamId}/injury-subs/available
func (h *RosterHandler) GetEligibleSubs(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.authorise(w, r); !ok {
		return
	}
	leagueID, teamID, err := pathLeagueTeam(r)
	if err != nil {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}

	injuredIDStr := r.URL.Query().Get("injuredPlayerId")
	injuredID, err := strconv.Atoi(injuredIDStr)
	if err != nil {
		http.Error(w, "missing injuredPlayerId", http.StatusBadRequest)
		return
	}

	players, err := h.store.GetEligibleSubs(leagueID, teamID, injuredID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if players == nil {
		players = []store.Player{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(players)
}

// GetTransactionLog handles GET /api/leagues/{id}/transactions
// Returns all elective + injury-sub transactions for the league, newest first.
// Any authenticated member of the league can read this.
func (h *RosterHandler) GetTransactionLog(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.authorise(w, r); !ok {
		return
	}
	leagueID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	records, err := h.store.GetLeagueTransactions(leagueID)
	if err != nil {
		http.Error(w, "query failed", http.StatusInternalServerError)
		return
	}
	if records == nil {
		records = []store.TransactionRecord{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(records)
}

// CutPlayer handles POST /api/leagues/{id}/teams/{teamId}/cut
func (h *RosterHandler) CutPlayer(w http.ResponseWriter, r *http.Request) {
	callerTeamID, ok := h.authorise(w, r)
	if !ok {
		return
	}
	leagueID, teamID, err := pathLeagueTeam(r)
	if err != nil || callerTeamID != teamID {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	var req struct {
		PlayerID int `json:"playerId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.PlayerID == 0 {
		http.Error(w, "missing playerId", http.StatusBadRequest)
		return
	}

	if err := h.store.CutInjuredPlayer(leagueID, teamID, req.PlayerID); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]any{"success": false, "error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"success": true})
}
