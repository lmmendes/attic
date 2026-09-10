package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lmmendes/attic/internal/domain"
)

type ConfigurationRepository struct {
	pool *pgxpool.Pool
}

func NewConfigurationRepository(pool *pgxpool.Pool) *ConfigurationRepository {
	return &ConfigurationRepository{pool: pool}
}

func (r *ConfigurationRepository) Get(ctx context.Context, orgID uuid.UUID) (*domain.FeatureConfiguration, error) {
	configuration := &domain.FeatureConfiguration{OrganizationID: orgID}
	if _, err := r.pool.Exec(ctx, `
		INSERT INTO feature_configurations (organization_id)
		VALUES ($1)
		ON CONFLICT (organization_id) DO NOTHING
	`, orgID); err != nil {
		return nil, err
	}
	err := r.pool.QueryRow(ctx, `
		SELECT organization_id, collections_enabled, plugins_enabled, conditions_enabled, locations_enabled, warranties_enabled, updated_at
		FROM feature_configurations
		WHERE organization_id = $1
	`, orgID).Scan(
		&configuration.OrganizationID,
		&configuration.CollectionsEnabled,
		&configuration.PluginsEnabled,
		&configuration.ConditionsEnabled,
		&configuration.LocationsEnabled,
		&configuration.WarrantiesEnabled,
		&configuration.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return configuration, nil
}

func (r *ConfigurationRepository) Update(ctx context.Context, configuration *domain.FeatureConfiguration) error {
	return r.pool.QueryRow(ctx, `
		INSERT INTO feature_configurations (
			organization_id, collections_enabled, plugins_enabled, conditions_enabled, locations_enabled, warranties_enabled
		) VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (organization_id) DO UPDATE SET
			collections_enabled = EXCLUDED.collections_enabled,
			plugins_enabled = EXCLUDED.plugins_enabled,
			conditions_enabled = EXCLUDED.conditions_enabled,
			locations_enabled = EXCLUDED.locations_enabled
			warranties_enabled = EXCLUDED.warranties_enabled
		RETURNING updated_at
	`,
		configuration.OrganizationID,
		configuration.CollectionsEnabled,
		configuration.PluginsEnabled,
		configuration.ConditionsEnabled,
		configuration.LocationsEnabled,
		configuration.WarrantiesEnabled,
	).Scan(&configuration.UpdatedAt)
}
