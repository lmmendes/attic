package handler

import (
	"net/http"

	"github.com/lmmendes/attic/internal/domain"
)

// CreateAttributeRequest represents the request body for creating an attribute
type CreateAttributeRequest struct {
	Name     string                  `json:"name"`
	Key      string                  `json:"key"`
	DataType domain.AttributeDataType `json:"data_type"`
}

// UpdateAttributeRequest represents the request body for updating an attribute
type UpdateAttributeRequest struct {
	Name     string                  `json:"name"`
	DataType domain.AttributeDataType `json:"data_type"`
}

// ListAttributes returns all attributes for the organization
func (h *Handler) ListAttributes(w http.ResponseWriter, r *http.Request) {
	attributes, err := h.repos.Attributes.List(r.Context(), h.orgID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list attributes")
		return
	}
	writeJSON(w, http.StatusOK, attributes)
}

// GetAttribute returns a single attribute by ID
func (h *Handler) GetAttribute(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid attribute ID")
		return
	}

	attr, err := h.repos.Attributes.GetByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get attribute")
		return
	}
	if attr == nil {
		writeError(w, http.StatusNotFound, "attribute not found")
		return
	}

	writeJSON(w, http.StatusOK, attr)
}

// CreateAttribute creates a new attribute
func (h *Handler) CreateAttribute(w http.ResponseWriter, r *http.Request) {
	var req CreateAttributeRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	if req.Key == "" {
		writeError(w, http.StatusBadRequest, "key is required")
		return
	}
	if req.DataType == "" {
		writeError(w, http.StatusBadRequest, "data_type is required")
		return
	}

	// Validate data type
	switch req.DataType {
	case domain.AttributeTypeString, domain.AttributeTypeNumber, domain.AttributeTypeBoolean, domain.AttributeTypeText, domain.AttributeTypeDate:
		// Valid
	default:
		writeError(w, http.StatusBadRequest, "invalid data_type: must be one of string, number, boolean, text, date")
		return
	}

	attr := &domain.Attribute{
		OrganizationID: h.orgID,
		Name:           req.Name,
		Key:            req.Key,
		DataType:       req.DataType,
	}

	if err := h.repos.Attributes.Create(r.Context(), attr); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create attribute")
		return
	}

	writeJSON(w, http.StatusCreated, attr)
}

// UpdateAttribute updates an existing attribute
func (h *Handler) UpdateAttribute(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid attribute ID")
		return
	}

	// Check if attribute exists
	existing, err := h.repos.Attributes.GetByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get attribute")
		return
	}
	if existing == nil {
		writeError(w, http.StatusNotFound, "attribute not found")
		return
	}

	var req UpdateAttributeRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	if req.DataType == "" {
		writeError(w, http.StatusBadRequest, "data_type is required")
		return
	}

	// Validate data type
	switch req.DataType {
	case domain.AttributeTypeString, domain.AttributeTypeNumber, domain.AttributeTypeBoolean, domain.AttributeTypeText, domain.AttributeTypeDate:
		// Valid
	default:
		writeError(w, http.StatusBadRequest, "invalid data_type: must be one of string, number, boolean, text, date")
		return
	}

	existing.Name = req.Name
	existing.DataType = req.DataType

	if err := h.repos.Attributes.Update(r.Context(), existing); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update attribute")
		return
	}

	writeJSON(w, http.StatusOK, existing)
}

// DeleteAttribute soft-deletes an attribute
func (h *Handler) DeleteAttribute(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid attribute ID")
		return
	}

	// Check if attribute exists
	existing, err := h.repos.Attributes.GetByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get attribute")
		return
	}
	if existing == nil {
		writeError(w, http.StatusNotFound, "attribute not found")
		return
	}

	// Prevent deleting plugin-owned attributes
	if existing.PluginID != nil {
		writeError(w, http.StatusForbidden, "cannot delete plugin-owned attribute")
		return
	}

	if err := h.repos.Attributes.Delete(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete attribute")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
