package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/lmmendes/attic/internal/domain"
)

type UpdateConfigurationRequest struct {
	CollectionsEnabled *bool `json:"collections_enabled"`
	PluginsEnabled     *bool `json:"plugins_enabled"`
	ConditionsEnabled  *bool `json:"conditions_enabled"`
	LocationsEnabled   *bool `json:"locations_enabled"`
	WarrantiesEnabled  *bool `json:"warranties_enabled"`
}

func (h *Handler) GetConfiguration(w http.ResponseWriter, r *http.Request) {
	configuration, err := h.repos.Configurations.Get(r.Context(), h.orgID)
	if err != nil {
		slog.Error("failed to get feature configuration", "organization_id", h.orgID, "error", err)
		writeError(w, http.StatusInternalServerError, "failed to get configuration")
		return
	}
	writeJSON(w, http.StatusOK, configuration)
}

func (h *Handler) UpdateConfiguration(w http.ResponseWriter, r *http.Request) {
	var request UpdateConfigurationRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if request.CollectionsEnabled == nil && request.PluginsEnabled == nil && request.ConditionsEnabled == nil && request.LocationsEnabled == nil && request.WarrantiesEnabled == nil {
		writeError(w, http.StatusBadRequest, "at least one feature setting is required")
		return
	}
	configuration, err := h.repos.Configurations.Get(r.Context(), h.orgID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get configuration")
		return
	}
	applyConfigurationUpdate(configuration, request)
	if err := h.repos.Configurations.Update(r.Context(), configuration); err != nil {
		slog.Error("failed to update feature configuration", "organization_id", h.orgID, "error", err)
		writeError(w, http.StatusInternalServerError, "failed to update configuration")
		return
	}
	writeJSON(w, http.StatusOK, configuration)
}

func (h *Handler) RequireFeature(feature string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			configuration, err := h.repos.Configurations.Get(r.Context(), h.orgID)
			if err != nil {
				writeError(w, http.StatusInternalServerError, "failed to get configuration")
				return
			}
			if !featureEnabled(configuration, feature) {
				writeError(w, http.StatusForbidden, feature+" feature is disabled")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func applyConfigurationUpdate(configuration *domain.FeatureConfiguration, request UpdateConfigurationRequest) {
	if request.CollectionsEnabled != nil {
		configuration.CollectionsEnabled = *request.CollectionsEnabled
	}
	if request.PluginsEnabled != nil {
		configuration.PluginsEnabled = *request.PluginsEnabled
	}
	if request.ConditionsEnabled != nil {
		configuration.ConditionsEnabled = *request.ConditionsEnabled
	}
	if request.LocationsEnabled != nil {
		configuration.LocationsEnabled = *request.LocationsEnabled
	}
	if request.WarrantiesEnabled != nil {
		configuration.WarrantiesEnabled = *request.WarrantiesEnabled
	}
}

func featureEnabled(configuration *domain.FeatureConfiguration, feature string) bool {
	switch feature {
	case "collections":
		return configuration.CollectionsEnabled
	case "plugins":
		return configuration.PluginsEnabled
	case "conditions":
		return configuration.ConditionsEnabled
	case "locations":
		return configuration.LocationsEnabled
	case "warranties":
		return configuration.WarrantiesEnabled
	default:
		return false
	}
}
