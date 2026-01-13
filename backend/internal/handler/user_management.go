package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/mendelui/attic/internal/auth"
	"github.com/mendelui/attic/internal/domain"
	"github.com/mendelui/attic/internal/repository"
)

// UserManagementHandler handles user management endpoints (admin only)
type UserManagementHandler struct {
	userRepo          *repository.UserRepository
	sessionManager    *auth.SessionManager
	passwordMinLength int
	defaultOrgID      uuid.UUID
}

// NewUserManagementHandler creates a new user management handler
func NewUserManagementHandler(userRepo *repository.UserRepository, sessionManager *auth.SessionManager, passwordMinLength int, defaultOrgID uuid.UUID) *UserManagementHandler {
	return &UserManagementHandler{
		userRepo:          userRepo,
		sessionManager:    sessionManager,
		passwordMinLength: passwordMinLength,
		defaultOrgID:      defaultOrgID,
	}
}

// RequireAdmin middleware checks if user is admin
func (h *UserManagementHandler) RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, err := h.sessionManager.GetSession(r)
		if err != nil {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}

		if session.Role != domain.UserRoleAdmin {
			http.Error(w, `{"error":"admin access required"}`, http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// UserResponse represents a user in API responses
type UserResponse struct {
	ID          string  `json:"id"`
	Email       string  `json:"email"`
	Name        *string `json:"name"`
	Role        string  `json:"role"`
	HasPassword bool    `json:"has_password"`
	HasOIDC     bool    `json:"has_oidc"`
	CreatedAt   string  `json:"created_at"`
}

func toUserResponse(u *domain.User) UserResponse {
	return UserResponse{
		ID:          u.ID.String(),
		Email:       u.Email,
		Name:        u.DisplayName,
		Role:        string(u.Role),
		HasPassword: u.HasPassword(),
		HasOIDC:     u.OIDCSubject != nil && *u.OIDCSubject != "",
		CreatedAt:   u.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}
}

// ListUsers returns all users
func (h *UserManagementHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.userRepo.List(r.Context(), h.defaultOrgID)
	if err != nil {
		slog.Error("failed to list users", "error", err)
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	response := make([]UserResponse, len(users))
	for i, u := range users {
		response[i] = toUserResponse(&u)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetUser returns a single user
func (h *UserManagementHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, `{"error":"invalid user ID"}`, http.StatusBadRequest)
		return
	}

	user, err := h.userRepo.GetByID(r.Context(), id)
	if err != nil {
		slog.Error("failed to get user", "error", err)
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	if user == nil {
		http.Error(w, `{"error":"user not found"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(toUserResponse(user))
}

// CreateUserRequest represents a request to create a user
type CreateUserRequest struct {
	Email    string `json:"email"`
	Name     string `json:"name"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

// CreateUser creates a new user
func (h *UserManagementHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var req CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	if req.Email == "" {
		http.Error(w, `{"error":"email is required"}`, http.StatusBadRequest)
		return
	}

	if req.Password == "" {
		http.Error(w, `{"error":"password is required"}`, http.StatusBadRequest)
		return
	}

	if err := auth.ValidatePassword(req.Password, h.passwordMinLength); err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusBadRequest)
		return
	}

	// Check if email already exists
	existing, err := h.userRepo.GetByEmail(r.Context(), req.Email)
	if err != nil {
		slog.Error("failed to check existing user", "error", err)
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}
	if existing != nil {
		http.Error(w, `{"error":"email already in use"}`, http.StatusConflict)
		return
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		slog.Error("failed to hash password", "error", err)
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	role := domain.UserRoleUser
	if req.Role == "admin" {
		role = domain.UserRoleAdmin
	}

	user := &domain.User{
		OrganizationID: h.defaultOrgID,
		Email:          req.Email,
		PasswordHash:   &hash,
		Role:           role,
	}
	if req.Name != "" {
		user.DisplayName = &req.Name
	}

	if err := h.userRepo.Create(r.Context(), user); err != nil {
		slog.Error("failed to create user", "error", err)
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(toUserResponse(user))
}

// UpdateUserRequest represents a request to update a user
type UpdateUserRequest struct {
	Email string `json:"email"`
	Name  string `json:"name"`
	Role  string `json:"role"`
}

// UpdateUser updates an existing user
func (h *UserManagementHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, `{"error":"invalid user ID"}`, http.StatusBadRequest)
		return
	}

	user, err := h.userRepo.GetByID(r.Context(), id)
	if err != nil {
		slog.Error("failed to get user", "error", err)
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	if user == nil {
		http.Error(w, `{"error":"user not found"}`, http.StatusNotFound)
		return
	}

	var req UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	if req.Email != "" && req.Email != user.Email {
		// Check if new email is already in use
		existing, err := h.userRepo.GetByEmail(r.Context(), req.Email)
		if err != nil {
			slog.Error("failed to check existing user", "error", err)
			http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
			return
		}
		if existing != nil && existing.ID != user.ID {
			http.Error(w, `{"error":"email already in use"}`, http.StatusConflict)
			return
		}
		user.Email = req.Email
	}

	if req.Name != "" {
		user.DisplayName = &req.Name
	}

	if req.Role != "" {
		if req.Role == "admin" {
			user.Role = domain.UserRoleAdmin
		} else {
			user.Role = domain.UserRoleUser
		}
	}

	if err := h.userRepo.Update(r.Context(), user); err != nil {
		slog.Error("failed to update user", "error", err)
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(toUserResponse(user))
}

// DeleteUser deletes a user
func (h *UserManagementHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, `{"error":"invalid user ID"}`, http.StatusBadRequest)
		return
	}

	// Get current session to prevent self-deletion
	session, _ := h.sessionManager.GetSession(r)
	if session != nil && session.UserID == id {
		http.Error(w, `{"error":"cannot delete your own account"}`, http.StatusBadRequest)
		return
	}

	user, err := h.userRepo.GetByID(r.Context(), id)
	if err != nil {
		slog.Error("failed to get user", "error", err)
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	if user == nil {
		http.Error(w, `{"error":"user not found"}`, http.StatusNotFound)
		return
	}

	if err := h.userRepo.Delete(r.Context(), id); err != nil {
		slog.Error("failed to delete user", "error", err)
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"success": true})
}

// ResetPasswordRequest represents a password reset request
type ResetPasswordRequest struct {
	Password string `json:"password"`
}

// ResetPassword resets a user's password (admin only)
func (h *UserManagementHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, `{"error":"invalid user ID"}`, http.StatusBadRequest)
		return
	}

	var req ResetPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	if req.Password == "" {
		http.Error(w, `{"error":"password is required"}`, http.StatusBadRequest)
		return
	}

	if err := auth.ValidatePassword(req.Password, h.passwordMinLength); err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusBadRequest)
		return
	}

	user, err := h.userRepo.GetByID(r.Context(), id)
	if err != nil {
		slog.Error("failed to get user", "error", err)
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	if user == nil {
		http.Error(w, `{"error":"user not found"}`, http.StatusNotFound)
		return
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		slog.Error("failed to hash password", "error", err)
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	if err := h.userRepo.UpdatePassword(r.Context(), id, hash); err != nil {
		slog.Error("failed to update password", "error", err)
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"success": true})
}
