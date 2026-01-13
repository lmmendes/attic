package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/mendelui/attic/internal/auth"
	"github.com/mendelui/attic/internal/repository"
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
		http.Error(w, `{"error":"email/password login is disabled when OIDC is enabled"}`, http.StatusBadRequest)
		return
	}

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	if req.Email == "" || req.Password == "" {
		http.Error(w, `{"error":"email and password are required"}`, http.StatusBadRequest)
		return
	}

	user, err := h.userRepo.GetByEmail(r.Context(), req.Email)
	if err != nil {
		slog.Error("failed to get user", "error", err)
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	if user == nil {
		http.Error(w, `{"error":"invalid email or password"}`, http.StatusUnauthorized)
		return
	}

	if !user.HasPassword() {
		http.Error(w, `{"error":"invalid email or password"}`, http.StatusUnauthorized)
		return
	}

	if !auth.CheckPassword(req.Password, *user.PasswordHash) {
		http.Error(w, `{"error":"invalid email or password"}`, http.StatusUnauthorized)
		return
	}

	if err := h.sessionManager.CreateSession(w, r, user); err != nil {
		slog.Error("failed to create session", "error", err)
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
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
		http.Error(w, `{"error":"password change is disabled when OIDC is enabled"}`, http.StatusBadRequest)
		return
	}

	session, err := h.sessionManager.GetSession(r)
	if err != nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	var req ChangePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	if req.CurrentPassword == "" || req.NewPassword == "" {
		http.Error(w, `{"error":"current and new password are required"}`, http.StatusBadRequest)
		return
	}

	if err := auth.ValidatePassword(req.NewPassword, h.passwordMinLength); err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusBadRequest)
		return
	}

	user, err := h.userRepo.GetByID(r.Context(), session.UserID)
	if err != nil || user == nil {
		http.Error(w, `{"error":"user not found"}`, http.StatusNotFound)
		return
	}

	if !user.HasPassword() || !auth.CheckPassword(req.CurrentPassword, *user.PasswordHash) {
		http.Error(w, `{"error":"current password is incorrect"}`, http.StatusUnauthorized)
		return
	}

	hash, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		slog.Error("failed to hash password", "error", err)
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	if err := h.userRepo.UpdatePassword(r.Context(), user.ID, hash); err != nil {
		slog.Error("failed to update password", "error", err)
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
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
