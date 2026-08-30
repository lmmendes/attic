package repository

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lmmendes/attic/internal/domain"
)

type AuthorizationCode struct {
	UserID        uuid.UUID
	ClientID      string
	RedirectURI   string
	CodeChallenge string
	Scope         string
}

type OAuthSession struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	ClientID  string
	Scope     string
	ExpiresAt time.Time
}

type OAuthRepository struct {
	pool *pgxpool.Pool
}

func NewOAuthRepository(pool *pgxpool.Pool) *OAuthRepository {
	return &OAuthRepository{pool: pool}
}

func (r *OAuthRepository) CreateAuthorizationCode(ctx context.Context, rawCode string, code AuthorizationCode, expiresAt time.Time) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO oauth_authorization_codes
			(code_hash, user_id, client_id, redirect_uri, code_challenge, scope, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, hashToken(rawCode), code.UserID, code.ClientID, code.RedirectURI, code.CodeChallenge, code.Scope, expiresAt)
	return err
}

func (r *OAuthRepository) ConsumeAuthorizationCode(ctx context.Context, rawCode string) (*AuthorizationCode, error) {
	var code AuthorizationCode
	err := r.pool.QueryRow(ctx, `
		UPDATE oauth_authorization_codes
		SET consumed_at = NOW()
		WHERE code_hash = $1
			AND consumed_at IS NULL
			AND expires_at > NOW()
		RETURNING user_id, client_id, redirect_uri, code_challenge, scope
	`, hashToken(rawCode)).Scan(&code.UserID, &code.ClientID, &code.RedirectURI, &code.CodeChallenge, &code.Scope)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &code, nil
}

func (r *OAuthRepository) CreateSession(ctx context.Context, userID uuid.UUID, clientID, scope, accessToken, refreshToken string, accessExpiresAt, refreshExpiresAt time.Time) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO oauth_sessions
			(user_id, client_id, scope, access_token_hash, access_expires_at, refresh_token_hash, refresh_expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, userID, clientID, scope, hashToken(accessToken), accessExpiresAt, hashToken(refreshToken), refreshExpiresAt)
	return err
}

func (r *OAuthRepository) GetUserByAccessToken(ctx context.Context, rawToken string) (*domain.User, error) {
	var user domain.User
	err := r.pool.QueryRow(ctx, `
		SELECT u.id, u.organization_id, u.oidc_subject, u.email, u.display_name,
			u.password_hash, u.role, u.created_at, u.updated_at
		FROM oauth_sessions s
		JOIN users u ON u.id = s.user_id
		WHERE s.access_token_hash = $1
			AND s.access_expires_at > NOW()
			AND s.revoked_at IS NULL
			AND u.deleted_at IS NULL
	`, hashToken(rawToken)).Scan(
		&user.ID, &user.OrganizationID, &user.OIDCSubject, &user.Email, &user.DisplayName,
		&user.PasswordHash, &user.Role, &user.CreatedAt, &user.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *OAuthRepository) RotateRefreshToken(ctx context.Context, rawRefreshToken, newAccessToken, newRefreshToken string, accessExpiresAt time.Time) (*OAuthSession, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	oldHash := hashToken(rawRefreshToken)
	var session OAuthSession
	err = tx.QueryRow(ctx, `
		SELECT id, user_id, client_id, scope, refresh_expires_at
		FROM oauth_sessions
		WHERE refresh_token_hash = $1
			AND refresh_expires_at > NOW()
			AND revoked_at IS NULL
		FOR UPDATE
	`, oldHash).Scan(&session.ID, &session.UserID, &session.ClientID, &session.Scope, &session.ExpiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		var replayedSessionID uuid.UUID
		replayErr := tx.QueryRow(ctx, `
			SELECT session_id FROM oauth_refresh_token_history WHERE token_hash = $1
		`, oldHash).Scan(&replayedSessionID)
		if replayErr == nil {
			if _, revokeErr := tx.Exec(ctx, `
				UPDATE oauth_sessions SET revoked_at = NOW(), updated_at = NOW()
				WHERE id = $1 AND revoked_at IS NULL
			`, replayedSessionID); revokeErr != nil {
				return nil, revokeErr
			}
			if err := tx.Commit(ctx); err != nil {
				return nil, err
			}
			return nil, fmt.Errorf("refresh token replay detected")
		}
		if !errors.Is(replayErr, pgx.ErrNoRows) {
			return nil, replayErr
		}
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO oauth_refresh_token_history (token_hash, session_id) VALUES ($1, $2)
	`, oldHash, session.ID); err != nil {
		return nil, err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE oauth_sessions
		SET access_token_hash = $2, access_expires_at = $3,
			refresh_token_hash = $4, updated_at = NOW()
		WHERE id = $1
	`, session.ID, hashToken(newAccessToken), accessExpiresAt, hashToken(newRefreshToken)); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &session, nil
}

func (r *OAuthRepository) RevokeRefreshToken(ctx context.Context, rawRefreshToken string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE oauth_sessions
		SET revoked_at = NOW(), updated_at = NOW()
		WHERE refresh_token_hash = $1 AND revoked_at IS NULL
	`, hashToken(rawRefreshToken))
	return err
}

func hashToken(rawToken string) string {
	sum := sha256.Sum256([]byte(rawToken))
	return hex.EncodeToString(sum[:])
}
