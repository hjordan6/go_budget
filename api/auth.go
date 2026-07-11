package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/hjordan6/go_budget/models"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type ctxKey string

const userCtxKey ctxKey = "userID"

const sessionCookie = "session"
const sessionTTL = 30 * 24 * time.Hour

// requireAuth wraps a handler so it only runs for a request carrying a valid,
// unexpired session cookie. The authenticated user ID is stashed on the
// request context for the wrapped handler to read via userID().
func (h *Handler) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		uid, ok := h.authenticate(r)
		if !ok {
			writeError(w, http.StatusUnauthorized, "not authenticated")
			return
		}
		ctx := context.WithValue(r.Context(), userCtxKey, uid)
		next(w, r.WithContext(ctx))
	}
}

// authenticate resolves the session cookie to a user ID. Expired sessions are
// deleted. It returns ok=false when there is no valid session.
func (h *Handler) authenticate(r *http.Request) (uint, bool) {
	c, err := r.Cookie(sessionCookie)
	if err != nil || c.Value == "" {
		return 0, false
	}
	var s models.Session
	if err := h.DB.Where("token = ?", c.Value).First(&s).Error; err != nil {
		return 0, false
	}
	if time.Now().After(s.ExpiresAt) {
		h.DB.Delete(&s)
		return 0, false
	}
	return s.UserID, true
}

func userID(r *http.Request) uint {
	return r.Context().Value(userCtxKey).(uint)
}

// startSession creates a session row and sets the cookie.
func (h *Handler) startSession(w http.ResponseWriter, uid uint) error {
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return err
	}
	token := hex.EncodeToString(tokenBytes)
	s := models.Session{Token: token, UserID: uid, ExpiresAt: time.Now().Add(sessionTTL)}
	if err := h.DB.Create(&s).Error; err != nil {
		return err
	}
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Expires:  s.ExpiresAt,
	})
	return nil
}

type registerRequest struct {
	Email    string `json:"email"`
	Name     string `json:"name"`
	Password string `json:"password"`
}

func (h *Handler) register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	if req.Email == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "email and password are required")
		return
	}

	// Reject duplicate emails up front for a clean error message.
	var existing models.User
	err := h.DB.Where("email = ?", req.Email).First(&existing).Error
	if err == nil {
		writeError(w, http.StatusConflict, "email already registered")
		return
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not hash password")
		return
	}
	user := models.User{Email: req.Email, Name: req.Name, PasswordHash: string(hash)}
	if err := h.DB.Create(&user).Error; err != nil {
		writeError(w, http.StatusInternalServerError, "could not create user")
		return
	}
	if err := h.startSession(w, user.ID); err != nil {
		writeError(w, http.StatusInternalServerError, "could not start session")
		return
	}
	writeJSON(w, http.StatusCreated, user)
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))

	var user models.User
	if err := h.DB.Where("email = ?", req.Email).First(&user).Error; err != nil {
		writeError(w, http.StatusUnauthorized, "invalid email or password")
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)) != nil {
		writeError(w, http.StatusUnauthorized, "invalid email or password")
		return
	}
	if err := h.startSession(w, user.ID); err != nil {
		writeError(w, http.StatusInternalServerError, "could not start session")
		return
	}
	writeJSON(w, http.StatusOK, user)
}

func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(sessionCookie); err == nil && c.Value != "" {
		h.DB.Where("token = ?", c.Value).Delete(&models.Session{})
	}
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
	writeJSON(w, http.StatusOK, map[string]string{"status": "logged out"})
}

// me returns the current user, or 401 when unauthenticated, so the SPA can
// bootstrap its auth state on load.
func (h *Handler) me(w http.ResponseWriter, r *http.Request) {
	uid, ok := h.authenticate(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}
	var user models.User
	if err := h.DB.First(&user, uid).Error; err != nil {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}
	writeJSON(w, http.StatusOK, user)
}
