package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/lmmendes/attic/internal/auth"
	"github.com/lmmendes/attic/internal/repository"
)

// AuthHandler handles authentication endpoints
type AuthHandler struct {
	userRepo          *repository.UserRepository
	sessionManager    *auth.SessionManager
	passwordMinLength int
	oidcEnabled       bool
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(userRepo *repository.UserRepository, sessionManager *auth.SessionManager, passwordMinLength int, oidcEnabled bool) *AuthHandler {
	return &AuthHandler{
		userRepo:          userRepo,
		sessionManager:    sessionManager,
		passwordMinLength: passwordMinLength,
		oidcEnabled:       oidcEnabled,
	}
}

// LoginRequest represents login credentials
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Login handles email/password login
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	if h.oidcEnabled {
		writeError(w, http.StatusBadRequest, "email/password login is disabled when OIDC is enabled")
		return
	}

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Email == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "email and password are required")
		return
	}

	user, err := h.userRepo.GetByEmail(r.Context(), req.Email)
	if err != nil {
		slog.Error("failed to get user", "error", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	if user == nil {
		writeError(w, http.StatusUnauthorized, "invalid email or password")
		return
	}

	if !user.HasPassword() {
		writeError(w, http.StatusUnauthorized, "invalid email or password")
		return
	}

	if !auth.CheckPassword(req.Password, *user.PasswordHash) {
		writeError(w, http.StatusUnauthorized, "invalid email or password")
		return
	}

	if err := h.sessionManager.CreateSession(w, r, user); err != nil {
		slog.Error("failed to create session", "error", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"success": true,
		"user": map[string]any{
			"id":    user.ID.String(),
			"email": user.Email,
			"name":  user.DisplayName,
			"role":  user.Role,
		},
	})
}

// Logout clears the session
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	h.sessionManager.ClearSession(w)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"success": true})
}

// GetSession returns current session info
func (h *AuthHandler) GetSession(w http.ResponseWriter, r *http.Request) {
	info := h.sessionManager.GetSessionInfo(r)
	info["oidc_enabled"] = h.oidcEnabled
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(info)
}

// ChangePasswordRequest represents a password change request
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

// ChangePassword allows a user to change their own password
func (h *AuthHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	if h.oidcEnabled {
		writeError(w, http.StatusBadRequest, "password change is disabled when OIDC is enabled")
		return
	}

	session, err := h.sessionManager.GetSession(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req ChangePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.CurrentPassword == "" || req.NewPassword == "" {
		writeError(w, http.StatusBadRequest, "current and new password are required")
		return
	}

	if err := auth.ValidatePassword(req.NewPassword, h.passwordMinLength); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	user, err := h.userRepo.GetByID(r.Context(), session.UserID)
	if err != nil || user == nil {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}

	if !user.HasPassword() || !auth.CheckPassword(req.CurrentPassword, *user.PasswordHash) {
		writeError(w, http.StatusUnauthorized, "current password is incorrect")
		return
	}

	hash, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		slog.Error("failed to hash password", "error", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	if err := h.userRepo.UpdatePassword(r.Context(), user.ID, hash); err != nil {
		slog.Error("failed to update password", "error", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"success": true})
}

// GetAuthMode returns the current authentication mode
func (h *AuthHandler) GetAuthMode(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"oidc_enabled": h.oidcEnabled,
	})
}
