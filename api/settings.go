package api

import (
	"net/http"
	"strings"

	"github.com/hjordan6/go_budget/models"
)

// settingsResponse is the client-safe view of a user's settings. The Lunch Money
// token itself is never returned — only whether one is configured.
type settingsResponse struct {
	LunchMoneyConnected bool `json:"lunch_money_connected"`
}

func (h *Handler) getSettings(w http.ResponseWriter, r *http.Request) {
	var user models.User
	if err := h.DB.First(&user, userID(r)).Error; err != nil {
		writeError(w, http.StatusInternalServerError, "could not load settings")
		return
	}
	writeJSON(w, http.StatusOK, settingsResponse{LunchMoneyConnected: user.LunchMoneyToken != ""})
}

type updateSettingsRequest struct {
	// LunchMoneyToken, when present, sets the token (empty string disconnects).
	// A nil pointer leaves it unchanged.
	LunchMoneyToken *string `json:"lunch_money_token"`
}

func (h *Handler) updateSettings(w http.ResponseWriter, r *http.Request) {
	var req updateSettingsRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	uid := userID(r)

	if req.LunchMoneyToken != nil {
		token := strings.TrimSpace(*req.LunchMoneyToken)
		if err := h.DB.Model(&models.User{}).Where("id = ?", uid).
			Update("lunch_money_token", token).Error; err != nil {
			writeError(w, http.StatusInternalServerError, "could not save settings")
			return
		}
	}

	var user models.User
	if err := h.DB.First(&user, uid).Error; err != nil {
		writeError(w, http.StatusInternalServerError, "could not load settings")
		return
	}
	writeJSON(w, http.StatusOK, settingsResponse{LunchMoneyConnected: user.LunchMoneyToken != ""})
}
