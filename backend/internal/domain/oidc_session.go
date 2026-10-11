package domain

import "time"

// OIDCSession contains provider credentials and claims for a server-side session.
type OIDCSession struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	IDToken      string    `json:"id_token,omitempty"`
	ExpiresAt    time.Time `json:"expires_at"`
	Subject      string    `json:"sub"`
	Email        string    `json:"email"`
	Name         string    `json:"name"`
}
