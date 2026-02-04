package auth

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/lmmendes/attic/internal/domain"
)

const (
	sessionCookieNameLocal = "attic_session"
)

// LocalSession represents a session for email/password auth
type LocalSession struct {
	UserID    uuid.UUID       `json:"user_id"`
	Email     string          `json:"email"`
	Name      string          `json:"name"`
	Role      domain.UserRole `json:"role"`
	ExpiresAt time.Time       `json:"expires_at"`
	Token     string          `json:"token"`
}

// SessionManager handles local session management
type SessionManager struct {
	secret          []byte
	durationHours   int
	cookieSecure    bool
}

// NewSessionManager creates a new session manager
func NewSessionManager(secret string, durationHours int) *SessionManager {
	secretBytes := []byte(secret)
	if len(secretBytes) < 32 {
		padded := make([]byte, 32)
		copy(padded, secretBytes)
		secretBytes = padded
	}
	return &SessionManager{
		secret:        secretBytes[:32],
		durationHours: durationHours,
	}
}

// CreateSession creates a new session for a user
func (m *SessionManager) CreateSession(w http.ResponseWriter, r *http.Request, user *domain.User) error {
	token := generateSecureToken(32)

	name := ""
	if user.DisplayName != nil {
		name = *user.DisplayName
	}

	session := LocalSession{
		UserID:    user.ID,
		Email:     user.Email,
		Name:      name,
		Role:      user.Role,
		ExpiresAt: time.Now().Add(time.Duration(m.durationHours) * time.Hour),
		Token:     token,
	}

	data, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("marshaling session: %w", err)
	}

	encoded := base64.StdEncoding.EncodeToString(data)

	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieNameLocal,
		Value:    encoded,
		Path:     "/",
		MaxAge:   m.durationHours * 3600,
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteLaxMode,
	})

	return nil
}

// GetSession retrieves the current session from cookie
func (m *SessionManager) GetSession(r *http.Request) (*LocalSession, error) {
	cookie, err := r.Cookie(sessionCookieNameLocal)
	if err != nil {
		return nil, err
	}

	data, err := base64.StdEncoding.DecodeString(cookie.Value)
	if err != nil {
		return nil, fmt.Errorf("decoding session: %w", err)
	}

	var session LocalSession
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, fmt.Errorf("unmarshaling session: %w", err)
	}

	if time.Now().After(session.ExpiresAt) {
		return nil, fmt.Errorf("session expired")
	}

	return &session, nil
}

// ClearSession removes the session cookie
func (m *SessionManager) ClearSession(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieNameLocal,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})
}

// GetSessionInfo returns session info for API response
func (m *SessionManager) GetSessionInfo(r *http.Request) map[string]any {
	session, err := m.GetSession(r)
	if err != nil || session == nil {
		return map[string]any{
			"authenticated": false,
		}
	}

	return map[string]any{
		"authenticated": true,
		"user": map[string]any{
			"id":    session.UserID.String(),
			"email": session.Email,
			"name":  session.Name,
			"role":  session.Role,
		},
		"expires_at": session.ExpiresAt,
	}
}

func generateSecureToken(length int) string {
	b := make([]byte, length)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)[:length]
}
