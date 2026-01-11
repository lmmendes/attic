package repository

import (
	"context"
	"errors"

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
		SELECT id, organization_id, oidc_subject, email, display_name, created_at, updated_at
		FROM users
		WHERE id = $1 AND deleted_at IS NULL
	`
	var u domain.User
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&u.ID, &u.OrganizationID, &u.OIDCSubject, &u.Email, &u.DisplayName,
		&u.CreatedAt, &u.UpdatedAt,
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
		SELECT id, organization_id, oidc_subject, email, display_name, created_at, updated_at
		FROM users
		WHERE oidc_subject = $1 AND deleted_at IS NULL
	`
	var u domain.User
	err := r.pool.QueryRow(ctx, query, subject).Scan(
		&u.ID, &u.OrganizationID, &u.OIDCSubject, &u.Email, &u.DisplayName,
		&u.CreatedAt, &u.UpdatedAt,
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
		SELECT id, organization_id, oidc_subject, email, display_name, created_at, updated_at
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
			&u.CreatedAt, &u.UpdatedAt,
		); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

func (r *UserRepository) Create(ctx context.Context, u *domain.User) error {
	query := `
		INSERT INTO users (id, organization_id, oidc_subject, email, display_name)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING created_at, updated_at
	`
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	return r.pool.QueryRow(ctx, query,
		u.ID, u.OrganizationID, u.OIDCSubject, u.Email, u.DisplayName,
	).Scan(&u.CreatedAt, &u.UpdatedAt)
}

func (r *UserRepository) Update(ctx context.Context, u *domain.User) error {
	query := `
		UPDATE users
		SET email = $2, display_name = $3
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING updated_at
	`
	return r.pool.QueryRow(ctx, query,
		u.ID, u.Email, u.DisplayName,
	).Scan(&u.UpdatedAt)
}

// GetOrCreate finds a user by OIDC subject or creates a new one
func (r *UserRepository) GetOrCreate(ctx context.Context, orgID uuid.UUID, subject, email, displayName string) (*domain.User, bool, error) {
	// Try to find existing user
	user, err := r.GetByOIDCSubject(ctx, subject)
	if err != nil {
		return nil, false, err
	}
	if user != nil {
		// Update email/name if changed
		if user.Email != email || (displayName != "" && (user.DisplayName == nil || *user.DisplayName != displayName)) {
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

	// Create new user
	user = &domain.User{
		OrganizationID: orgID,
		OIDCSubject:    subject,
		Email:          email,
	}
	if displayName != "" {
		user.DisplayName = &displayName
	}

	if err := r.Create(ctx, user); err != nil {
		return nil, false, err
	}

	return user, true, nil
}
