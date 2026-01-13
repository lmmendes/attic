package repository

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mendelui/attic/internal/domain"
)

type UserRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	query := `
		SELECT id, organization_id, oidc_subject, email, display_name, password_hash, role, created_at, updated_at
		FROM users
		WHERE id = $1 AND deleted_at IS NULL
	`
	var u domain.User
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&u.ID, &u.OrganizationID, &u.OIDCSubject, &u.Email, &u.DisplayName,
		&u.PasswordHash, &u.Role, &u.CreatedAt, &u.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	query := `
		SELECT id, organization_id, oidc_subject, email, display_name, password_hash, role, created_at, updated_at
		FROM users
		WHERE LOWER(email) = LOWER($1) AND deleted_at IS NULL
	`
	var u domain.User
	err := r.pool.QueryRow(ctx, query, email).Scan(
		&u.ID, &u.OrganizationID, &u.OIDCSubject, &u.Email, &u.DisplayName,
		&u.PasswordHash, &u.Role, &u.CreatedAt, &u.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepository) GetByOIDCSubject(ctx context.Context, subject string) (*domain.User, error) {
	query := `
		SELECT id, organization_id, oidc_subject, email, display_name, password_hash, role, created_at, updated_at
		FROM users
		WHERE oidc_subject = $1 AND deleted_at IS NULL
	`
	var u domain.User
	err := r.pool.QueryRow(ctx, query, subject).Scan(
		&u.ID, &u.OrganizationID, &u.OIDCSubject, &u.Email, &u.DisplayName,
		&u.PasswordHash, &u.Role, &u.CreatedAt, &u.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepository) List(ctx context.Context, orgID uuid.UUID) ([]domain.User, error) {
	query := `
		SELECT id, organization_id, oidc_subject, email, display_name, password_hash, role, created_at, updated_at
		FROM users
		WHERE organization_id = $1 AND deleted_at IS NULL
		ORDER BY email
	`
	rows, err := r.pool.Query(ctx, query, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []domain.User
	for rows.Next() {
		var u domain.User
		if err := rows.Scan(
			&u.ID, &u.OrganizationID, &u.OIDCSubject, &u.Email, &u.DisplayName,
			&u.PasswordHash, &u.Role, &u.CreatedAt, &u.UpdatedAt,
		); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

func (r *UserRepository) Count(ctx context.Context) (int, error) {
	query := `SELECT COUNT(*) FROM users WHERE deleted_at IS NULL`
	var count int
	err := r.pool.QueryRow(ctx, query).Scan(&count)
	return count, err
}

func (r *UserRepository) Create(ctx context.Context, u *domain.User) error {
	query := `
		INSERT INTO users (id, organization_id, oidc_subject, email, display_name, password_hash, role)
		VALUES ($1, $2, $3, LOWER($4), $5, $6, $7)
		RETURNING created_at, updated_at
	`
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	// Normalize email to lowercase
	u.Email = strings.ToLower(u.Email)
	return r.pool.QueryRow(ctx, query,
		u.ID, u.OrganizationID, u.OIDCSubject, u.Email, u.DisplayName, u.PasswordHash, u.Role,
	).Scan(&u.CreatedAt, &u.UpdatedAt)
}

func (r *UserRepository) Update(ctx context.Context, u *domain.User) error {
	query := `
		UPDATE users
		SET email = LOWER($2), display_name = $3, role = $4, updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING updated_at
	`
	// Normalize email to lowercase
	u.Email = strings.ToLower(u.Email)
	return r.pool.QueryRow(ctx, query,
		u.ID, u.Email, u.DisplayName, u.Role,
	).Scan(&u.UpdatedAt)
}

func (r *UserRepository) UpdatePassword(ctx context.Context, id uuid.UUID, passwordHash string) error {
	query := `
		UPDATE users
		SET password_hash = $2, updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
	`
	_, err := r.pool.Exec(ctx, query, id, passwordHash)
	return err
}

func (r *UserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE users
		SET deleted_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
	`
	_, err := r.pool.Exec(ctx, query, id)
	return err
}

func (r *UserRepository) LinkOIDC(ctx context.Context, id uuid.UUID, oidcSubject string) error {
	query := `
		UPDATE users
		SET oidc_subject = $2, updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
	`
	_, err := r.pool.Exec(ctx, query, id, oidcSubject)
	return err
}

// GetOrCreate finds a user by OIDC subject or creates a new one
func (r *UserRepository) GetOrCreate(ctx context.Context, orgID uuid.UUID, subject, email, displayName string) (*domain.User, bool, error) {
	// Try to find existing user by OIDC subject
	user, err := r.GetByOIDCSubject(ctx, subject)
	if err != nil {
		return nil, false, err
	}
	if user != nil {
		// Update email/name if changed
		if user.Email != strings.ToLower(email) || (displayName != "" && (user.DisplayName == nil || *user.DisplayName != displayName)) {
			user.Email = email
			if displayName != "" {
				user.DisplayName = &displayName
			}
			if err := r.Update(ctx, user); err != nil {
				return nil, false, err
			}
		}
		return user, false, nil
	}

	// Try to find existing user by email (for account linking)
	user, err = r.GetByEmail(ctx, email)
	if err != nil {
		return nil, false, err
	}
	if user != nil {
		// Link OIDC subject to existing user
		if err := r.LinkOIDC(ctx, user.ID, subject); err != nil {
			return nil, false, err
		}
		user.OIDCSubject = &subject
		// Update display name if provided
		if displayName != "" && (user.DisplayName == nil || *user.DisplayName != displayName) {
			user.DisplayName = &displayName
			if err := r.Update(ctx, user); err != nil {
				return nil, false, err
			}
		}
		return user, false, nil
	}

	// Create new user
	oidcSubject := subject
	user = &domain.User{
		OrganizationID: orgID,
		OIDCSubject:    &oidcSubject,
		Email:          email,
		Role:           domain.UserRoleUser,
	}
	if displayName != "" {
		user.DisplayName = &displayName
	}

	if err := r.Create(ctx, user); err != nil {
		return nil, false, err
	}

	return user, true, nil
}
