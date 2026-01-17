package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/mendelui/attic/internal/auth"
	"github.com/mendelui/attic/internal/domain"
)

// testCurrentUserHandler wraps user handler logic for testing
type testCurrentUserHandler struct{}

func newTestCurrentUserHandler() *testCurrentUserHandler {
	return &testCurrentUserHandler{}
}

func (h *testCurrentUserHandler) GetCurrentUser(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUser(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	response := CurrentUserResponse{
		ID:          user.ID.String(),
		Email:       user.Email,
		DisplayName: user.DisplayName,
	}

	writeJSON(w, http.StatusOK, response)
}

func withUserContext(r *http.Request, user *domain.User) *http.Request {
	ctx := context.WithValue(r.Context(), auth.DomainUserContextKey, user)
	return r.WithContext(ctx)
}

// Tests for GetCurrentUser

func Test_GetCurrentUser_Authenticated_ReturnsUser(t *testing.T) {
	h := newTestCurrentUserHandler()
	displayName := "Test User"
	user := &domain.User{
		ID:          uuid.New(),
		Email:       "test@example.com",
		DisplayName: &displayName,
		Role:        domain.UserRoleUser,
	}

	req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	req = withUserContext(req, user)
	rec := httptest.NewRecorder()

	h.GetCurrentUser(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var response CurrentUserResponse
	json.NewDecoder(rec.Body).Decode(&response)
	if response.Email != "test@example.com" {
		t.Errorf("expected email test@example.com, got %s", response.Email)
	}
	if response.DisplayName == nil || *response.DisplayName != "Test User" {
		t.Error("expected display name to be 'Test User'")
	}
}

func Test_GetCurrentUser_NotAuthenticated_ReturnsUnauthorized(t *testing.T) {
	h := newTestCurrentUserHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	rec := httptest.NewRecorder()

	h.GetCurrentUser(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", rec.Code)
	}
}

func Test_GetCurrentUser_WithoutDisplayName_ReturnsNil(t *testing.T) {
	h := newTestCurrentUserHandler()
	user := &domain.User{
		ID:    uuid.New(),
		Email: "test@example.com",
		Role:  domain.UserRoleUser,
	}

	req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	req = withUserContext(req, user)
	rec := httptest.NewRecorder()

	h.GetCurrentUser(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var response CurrentUserResponse
	json.NewDecoder(rec.Body).Decode(&response)
	if response.DisplayName != nil {
		t.Errorf("expected nil display name, got %v", response.DisplayName)
	}
}

