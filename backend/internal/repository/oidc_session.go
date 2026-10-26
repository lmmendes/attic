package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lmmendes/attic/internal/domain"
)

// OIDCSessionRepository stores OIDC credentials behind hashed session IDs.
type OIDCSessionRepository struct{ pool *pgxpool.Pool }

func NewOIDCSessionRepository(pool *pgxpool.Pool) *OIDCSessionRepository {
	return &OIDCSessionRepository{pool: pool}
}

func (r *OIDCSessionRepository) Create(ctx context.Context, hash string, s *domain.OIDCSession) error {
	_, err := r.pool.Exec(ctx, `INSERT INTO oidc_sessions
 (session_id_hash, access_token, refresh_token, id_token, subject, email, name, expires_at)
 VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`, hash, s.AccessToken, s.RefreshToken, s.IDToken, s.Subject, s.Email, s.Name, s.ExpiresAt)
	return err
}

func (r *OIDCSessionRepository) Get(ctx context.Context, hash string) (*domain.OIDCSession, error) {
	var s domain.OIDCSession
	err := r.pool.QueryRow(ctx, `SELECT access_token, refresh_token, id_token, subject, email, name, expires_at
 FROM oidc_sessions WHERE session_id_hash = $1 AND expires_at > NOW()`, hash).Scan(
		&s.AccessToken, &s.RefreshToken, &s.IDToken, &s.Subject, &s.Email, &s.Name, &s.ExpiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *OIDCSessionRepository) Delete(ctx context.Context, hash string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM oidc_sessions WHERE session_id_hash = $1`, hash)
	return err
}

func (r *OIDCSessionRepository) DeleteExpired(ctx context.Context) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM oidc_sessions WHERE expires_at <= NOW()`)
	return err
}
