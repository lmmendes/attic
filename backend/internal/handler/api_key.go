package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/lmmendes/attic/internal/auth"
	"github.com/lmmendes/attic/internal/domain"
	"github.com/lmmendes/attic/internal/repository"
)

type APIKeyHandler struct {
	Keys           *repository.APIKeyRepository
	OrganizationID uuid.UUID
}

func (h *APIKeyHandler) Catalog(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, domain.PermissionCatalog(auth.GetUser(r.Context()).Role))
}
func (h *APIKeyHandler) List(w http.ResponseWriter, r *http.Request) {
	keys, err := h.Keys.List(r.Context(), h.OrganizationID, auth.GetUser(r.Context()).ID)
	if err != nil {
		writeError(w, 500, "failed to list keys")
		return
	}
	writeJSON(w, 200, keys)
}
func (h *APIKeyHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Description string            `json:"description"`
		Permissions map[string]string `json:"permissions"`
		ExpiresAt   *time.Time        `json:"expires_at"`
	}
	if err := decodeKeyJSON(r, &req); err != nil {
		writeError(w, 400, "invalid request body")
		return
	}
	description, err := domain.ValidateKeyDescription(req.Description)
	if err != nil {
		writeError(w, 400, err.Error())
		return
	}
	user := auth.GetUser(r.Context())
	if err := domain.ValidateKeyPermissions(req.Permissions, user.Role); err != nil {
		writeError(w, 400, err.Error())
		return
	}
	if req.ExpiresAt != nil && !req.ExpiresAt.After(time.Now()) {
		writeError(w, 400, "expiry must be in the future")
		return
	}
	secret, hash, err := domain.NewAPIKeySecret()
	if err != nil {
		writeError(w, 500, "failed to generate key")
		return
	}
	key := domain.APIKey{ID: uuid.New(), OwnerID: user.ID, OrganizationID: h.OrganizationID, Description: description, Permissions: req.Permissions, ExpiresAt: req.ExpiresAt, SecretHash: hash, Prefix: secret[:14]}
	if err := h.Keys.Create(r.Context(), &key); err != nil {
		writeError(w, 500, "failed to create key")
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, 201, struct {
		domain.APIKey
		Secret string `json:"secret"`
	}{key, secret})
}
func decodeKeyJSON(r *http.Request, target any) error {
	decoder := json.NewDecoder(http.MaxBytesReader(nil, r.Body, 64<<10))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return fmt.Errorf("expected a single JSON object")
	}
	return nil
}
func (h *APIKeyHandler) Rename(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		writeError(w, 400, "invalid key ID")
		return
	}
	var req struct {
		Description string `json:"description"`
	}
	if err := decodeKeyJSON(r, &req); err != nil {
		writeError(w, 400, "only description can be updated")
		return
	}
	description, err := domain.ValidateKeyDescription(req.Description)
	if err != nil {
		writeError(w, 400, err.Error())
		return
	}
	ok, err := h.Keys.Rename(r.Context(), h.OrganizationID, auth.GetUser(r.Context()).ID, id, description)
	keyMutationResult(w, ok, err)
}
func (h *APIKeyHandler) Revoke(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		writeError(w, 400, "invalid key ID")
		return
	}
	ok, err := h.Keys.Revoke(r.Context(), h.OrganizationID, auth.GetUser(r.Context()).ID, id)
	keyMutationResult(w, ok, err)
}
func keyMutationResult(w http.ResponseWriter, ok bool, err error) {
	if err != nil {
		writeError(w, 500, "failed to update key")
		return
	}
	if !ok {
		writeError(w, 404, "key not found")
		return
	}
	w.WriteHeader(204)
}
func (h *APIKeyHandler) GetMCP(w http.ResponseWriter, r *http.Request) {
	enabled, err := h.Keys.MCPEnabled(r.Context(), h.OrganizationID)
	if err != nil {
		writeError(w, 500, "failed to read MCP setting")
		return
	}
	writeJSON(w, 200, map[string]bool{"enabled": enabled})
}
func (h *APIKeyHandler) UpdateMCP(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Enabled *bool `json:"enabled"`
	}
	if err := decodeKeyJSON(r, &req); err != nil || req.Enabled == nil {
		writeError(w, 400, "enabled must be a boolean")
		return
	}
	if err := h.Keys.SetMCPEnabled(r.Context(), h.OrganizationID, *req.Enabled); err != nil {
		writeError(w, 500, "failed to update MCP setting")
		return
	}
	writeJSON(w, 200, map[string]bool{"enabled": *req.Enabled})
}
