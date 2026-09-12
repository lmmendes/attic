package handler

import (
	"encoding/json"
	"net/http"

	"github.com/lmmendes/attic/internal/domain"
)

// GetOrganizationFeatures returns the effective feature switches for the
// current organization. Reading them is intentionally available to all users.
func (h *Handler) GetOrganizationFeatures(w http.ResponseWriter, r *http.Request) {
	features, err := h.features(r)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get organization features")
		return
	}
	writeJSON(w, http.StatusOK, features)
}

// UpdateOrganizationFeatures replaces the complete feature map. Rejecting
// unknown fields is handled by decoding into a map before decoding the model.
func (h *Handler) UpdateOrganizationFeatures(w http.ResponseWriter, r *http.Request) {
	var raw map[string]any
	if err := decodeJSON(r, &raw); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	known := map[string]bool{"locations": true, "collections": true, "categories": true, "attributes": true, "conditions": true, "warranties": true, "plugins": true}
	for key, value := range raw {
		if !known[key] {
			writeError(w, http.StatusBadRequest, "unknown feature: "+key)
			return
		}
		if value == nil {
			writeError(w, http.StatusBadRequest, "feature values must be booleans")
			return
		}
	}
	if len(raw) != len(known) {
		writeError(w, http.StatusBadRequest, "all features are required")
		return
	}
	features := &domain.OrganizationFeatures{}
	if err := decodeMap(raw, features); err != nil {
		writeError(w, http.StatusBadRequest, "feature values must be booleans")
		return
	}
	// Attributes are managed by the Categories switch. The legacy field remains
	// accepted in the complete map for compatibility but cannot diverge.
	features.Attributes = features.Categories
	if err := h.repos.Organizations.UpdateFeatures(r.Context(), h.orgID, features); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update organization features")
		return
	}
	writeJSON(w, http.StatusOK, features)
}

func decodeMap(raw map[string]any, target any) error {
	// Reuse the same JSON semantics as request decoding without accepting
	// unknown fields a second time.
	data, err := json.Marshal(raw)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, target)
}
