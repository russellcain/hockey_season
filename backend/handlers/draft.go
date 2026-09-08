package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"

	"hockey_season/backend/hub"
	"hockey_season/backend/store"
)

type DraftHandler struct {
	store  *store.Store
	hub    *hub.Hub
	secret string
}

func NewDraft(s *store.Store, h *hub.Hub, secret string) *DraftHandler {
	return &DraftHandler{store: s, hub: h, secret: secret}
}

func (h *DraftHandler) Get(w http.ResponseWriter, r *http.Request) {
	teamID, ok := h.authorise(w, r)
	if !ok {
		return
	}
	sessionID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	full, err := h.store.GetDraftFull(sessionID, teamID)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(full)
}

func (h *DraftHandler) Pick(w http.ResponseWriter, r *http.Request) {
	teamID, ok := h.authorise(w, r)
	if !ok {
		return
	}
	sessionID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	var req struct {
		PlayerID int `json:"playerId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.PlayerID == 0 {
		http.Error(w, "missing playerId", http.StatusBadRequest)
		return
	}

	result, err := h.store.MakePick(sessionID, teamID, req.PlayerID)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, store.ErrNotYourTurn) {
			status = http.StatusForbidden
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		json.NewEncoder(w).Encode(map[string]any{"success": false, "error": err.Error()})
		return
	}

	draftID := strconv.Itoa(sessionID)
	h.broadcastPickMade(draftID, teamID, result)
	if result.DraftState.Status == "complete" {
		h.broadcastDraftComplete(draftID, result)
		// Finalise asynchronously: populate roster_slots, generate H2H schedule,
		// and advance league status. We don't block the HTTP response for this.
		go func() {
			if err := h.store.FinaliseDraft(sessionID); err != nil {
				log.Printf("FinaliseDraft session %d: %v", sessionID, err)
			} else {
				log.Printf("FinaliseDraft session %d: season setup complete", sessionID)
			}
		}()
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"success": true})
}

func (h *DraftHandler) broadcastPickMade(draftID string, teamID int, r *store.PickResult) {
	msg, err := json.Marshal(map[string]any{
		"type": "pick_made",
		"payload": map[string]any{
			"playerId":   r.Player.ID,
			"playerName": r.Player.Name,
			"teamId":     teamID,
			"draftState": r.DraftState,
			"teams":      r.Teams,
		},
	})
	if err == nil {
		h.hub.Broadcast(draftID, msg)
	}
}

func (h *DraftHandler) broadcastDraftComplete(draftID string, r *store.PickResult) {
	msg, err := json.Marshal(map[string]any{
		"type":    "draft_complete",
		"payload": map[string]any{"finalTeams": r.Teams},
	})
	if err == nil {
		h.hub.Broadcast(draftID, msg)
	}
}

func (h *DraftHandler) authorise(w http.ResponseWriter, r *http.Request) (int, bool) {
	teamID, err := ParseToken(ExtractToken(r), h.secret)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return 0, false
	}
	return teamID, true
}
