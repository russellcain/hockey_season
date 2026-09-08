package handlers

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"hockey_season/backend/store"
)

type AuthHandler struct {
	store  *store.Store
	secret string
}

func NewAuth(s *store.Store, secret string) *AuthHandler {
	return &AuthHandler{store: s, secret: secret}
}

func (h *AuthHandler) Join(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Code == "" {
		http.Error(w, "missing code", http.StatusBadRequest)
		return
	}

	codeHash := computeHMAC(req.Code, h.secret)
	team, err := h.store.LookupTeamByCode(codeHash)
	if err != nil {
		http.Error(w, "invalid code", http.StatusUnauthorized)
		return
	}

	sessionID, err := h.store.GetActiveSessionID()
	if err != nil {
		http.Error(w, "no active draft session", http.StatusServiceUnavailable)
		return
	}

	token := SignToken(team.ID, h.secret)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"token":   token,
		"team":    team,
		"draftId": sessionID,
	})
}

// SignToken creates a self-contained token with format "teamID:expiry:hmac".
func SignToken(teamID int, secret string) string {
	expiry := time.Now().Add(24 * time.Hour).Unix()
	payload := fmt.Sprintf("%d:%d", teamID, expiry)
	sig := computeHMAC(payload, secret)
	return fmt.Sprintf("%s:%s", payload, sig)
}

// ParseToken validates the token and returns the encoded team ID.
func ParseToken(token, secret string) (int, error) {
	parts := strings.Split(token, ":")
	if len(parts) != 3 {
		return 0, errors.New("malformed token")
	}
	payload := parts[0] + ":" + parts[1]
	if computeHMAC(payload, secret) != parts[2] {
		return 0, errors.New("invalid signature")
	}
	expiry, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil || time.Now().Unix() > expiry {
		return 0, errors.New("token expired")
	}
	teamID, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, errors.New("invalid team id")
	}
	return teamID, nil
}

// ExtractToken reads the bearer token from the Authorization header or ?token= query param.
func ExtractToken(r *http.Request) string {
	if auth := r.Header.Get("Authorization"); strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}
	return r.URL.Query().Get("token")
}

func computeHMAC(data, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(data))
	return hex.EncodeToString(mac.Sum(nil))
}
