package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/lmmendes/attic/internal/domain"
)

type organizationFeaturesContextKey struct{}

func featuresFromContext(ctx context.Context) (*domain.OrganizationFeatures, bool) {
	features, ok := ctx.Value(organizationFeaturesContextKey{}).(*domain.OrganizationFeatures)
	return features, ok
}

func normalizeFeatures(features *domain.OrganizationFeatures) *domain.OrganizationFeatures {
	if features == nil {
		return nil
	}
	normalized := *features
	normalized.Attributes = normalized.Categories
	return &normalized
}

// LoadFeatures resolves feature settings once for each API request. Handlers
// then share the same snapshot, avoiding repeated database reads and ensuring
// enforcement fails closed if settings cannot be loaded.
func (h *Handler) LoadFeatures(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		features, err := h.repos.Organizations.GetFeatures(r.Context(), h.orgID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to read organization features")
			return
		}
		ctx := context.WithValue(r.Context(), organizationFeaturesContextKey{}, normalizeFeatures(features))
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (h *Handler) features(r *http.Request) (*domain.OrganizationFeatures, error) {
	if features, ok := featuresFromContext(r.Context()); ok {
		return normalizeFeatures(features), nil
	}
	features, err := h.repos.Organizations.GetFeatures(r.Context(), h.orgID)
	return normalizeFeatures(features), err
}

func (h *Handler) featureEnabled(r *http.Request, name string) (bool, error) {
	f, err := h.features(r)
	if err != nil {
		return false, err
	}
	switch name {
	case "locations":
		return f.Locations, nil
	case "collections":
		return f.Collections, nil
	case "categories":
		return f.Categories, nil
	case "attributes":
		return f.Attributes, nil
	case "conditions":
		return f.Conditions, nil
	case "warranties":
		return f.Warranties, nil
	case "plugins":
		return f.Plugins, nil
	default:
		return false, nil
	}
}

// RequireFeature protects feature-owned endpoints while retaining data in the
// database for later re-enabling.
func (h *Handler) RequireFeature(name string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			enabled, err := h.featureEnabled(r, name)
			if err != nil {
				writeError(w, http.StatusInternalServerError, "failed to read organization features")
				return
			}
			if !enabled {
				writeError(w, http.StatusNotFound, "feature is not enabled")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func (h *Handler) rejectDisabledAssetFields(r *http.Request, categoryID *string, locationID *string, conditionID *string, collectionIDs []uuid.UUID, attributes []byte) error {
	f, err := h.features(r)
	if err != nil {
		return err
	}
	if !f.Categories && (categoryID != nil && *categoryID != "") {
		return errFeatureDisabled("categories")
	}
	if !f.Locations && (locationID != nil && *locationID != "") {
		return errFeatureDisabled("locations")
	}
	if !f.Conditions && (conditionID != nil && *conditionID != "") {
		return errFeatureDisabled("conditions")
	}
	if !f.Collections && len(collectionIDs) > 0 {
		return errFeatureDisabled("collections")
	}
	if !f.Attributes && len(attributes) > 0 && string(attributes) != "null" && string(attributes) != "{}" {
		return errFeatureDisabled("attributes")
	}
	if !f.Plugins && containsPluginAttributes(attributes) {
		return errFeatureDisabled("plugins")
	}
	return nil
}

func containsPluginAttributes(raw []byte) bool {
	if len(raw) == 0 || string(raw) == "null" {
		return false
	}
	values := make(map[string]any)
	if json.Unmarshal(raw, &values) != nil {
		return false
	}
	for key := range values {
		if strings.HasPrefix(key, "plugin.") {
			return true
		}
	}
	return false
}

func mergePreservedPluginAttributes(existing, incoming []byte) ([]byte, error) {
	merged := make(map[string]any)
	if len(incoming) > 0 && string(incoming) != "null" {
		if err := json.Unmarshal(incoming, &merged); err != nil {
			return nil, err
		}
	}
	if len(existing) > 0 && string(existing) != "null" {
		oldValues := make(map[string]any)
		if err := json.Unmarshal(existing, &oldValues); err != nil {
			return nil, err
		}
		for key, value := range oldValues {
			if strings.HasPrefix(key, "plugin.") {
				merged[key] = value
			}
		}
	}
	return json.Marshal(merged)
}

type featureDisabledError string

func (e featureDisabledError) Error() string { return string(e) + " feature is not enabled" }
func errFeatureDisabled(name string) error   { return featureDisabledError(name) }

func (h *Handler) sanitizeAsset(r *http.Request, asset *domain.Asset) error {
	f, err := h.features(r)
	if err != nil {
		return err
	}
	if !f.Locations {
		asset.LocationID, asset.Location = nil, nil
	}
	if !f.Collections {
		asset.CollectionIDs, asset.Collections = nil, nil
	}
	if !f.Categories {
		asset.CategoryID, asset.Category = nil, nil
	}
	if !f.Conditions {
		asset.ConditionID, asset.Condition = nil, nil
	}
	if !f.Attributes {
		asset.Attributes = nil
	}
	if !f.Warranties {
		asset.Warranty = nil
	}
	if !f.Plugins {
		asset.ImportPluginID, asset.ImportExternalID = nil, nil
		if asset.Category != nil && asset.Category.PluginID != nil {
			asset.CategoryID, asset.Category = nil, nil
		}
		if len(asset.Attributes) > 0 {
			values := make(map[string]any)
			if err := json.Unmarshal(asset.Attributes, &values); err != nil {
				return err
			}
			for key := range values {
				if strings.HasPrefix(key, "plugin.") {
					delete(values, key)
				}
			}
			asset.Attributes, err = json.Marshal(values)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
