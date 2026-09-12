package handler

import (
	"errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/lmmendes/attic/internal/auth"
	"github.com/lmmendes/attic/internal/repository"
	"net/http"
	"strings"

	"github.com/lmmendes/attic/internal/domain"
)

// CreateAttributeRequest represents the request body for creating an attribute
type CreateAttributeRequest struct {
	Name          string                   `json:"name"`
	Key           string                   `json:"key"`
	DataType      domain.AttributeDataType `json:"data_type"`
	SelectionMode string                   `json:"selection_mode,omitempty"`
	Options       []domain.AttributeOption `json:"options,omitempty"`
}

// UpdateAttributeRequest represents the request body for updating an attribute
type UpdateAttributeRequest struct {
	Name          string                   `json:"name"`
	DataType      domain.AttributeDataType `json:"data_type"`
	SelectionMode string                   `json:"selection_mode,omitempty"`
	Options       []domain.AttributeOption `json:"options,omitempty"`
}

func hasReservedPluginAttributeKey(attribute *domain.Attribute) bool {
	return strings.HasPrefix(attribute.Key, "plugin.") && attribute.PluginID == nil
}

// ListAttributes returns all attributes for the organization
func (h *Handler) ListAttributes(w http.ResponseWriter, r *http.Request) {
	attributes, err := h.repos.Attributes.List(r.Context(), h.orgID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list attributes")
		return
	}
	enabled, featureErr := h.featureEnabled(r, "plugins")
	if featureErr != nil {
		writeError(w, http.StatusInternalServerError, "failed to read organization features")
		return
	}
	if !enabled {
		filtered := attributes[:0]
		for _, attribute := range attributes {
			if attribute.PluginID == nil {
				filtered = append(filtered, attribute)
			}
		}
		attributes = filtered
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
	if attr == nil || attr.OrganizationID != h.orgID {
		writeError(w, http.StatusNotFound, "attribute not found")
		return
	}
	enabled, featureErr := h.featureEnabled(r, "plugins")
	if featureErr != nil {
		writeError(w, http.StatusInternalServerError, "failed to read organization features")
		return
	}
	if !enabled && attr.PluginID != nil {
		writeError(w, http.StatusNotFound, "attribute not found")
		return
	}

	writeJSON(w, http.StatusOK, attr)
}

// CreateAttribute creates a new attribute
func (h *Handler) CreateAttribute(w http.ResponseWriter, r *http.Request) {
	if !h.attributeAdmin(w, r) {
		return
	}
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
	case domain.AttributeTypeString, domain.AttributeTypeNumber, domain.AttributeTypeBoolean, domain.AttributeTypeText, domain.AttributeTypeDate, domain.AttributeTypeSelect:
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
		SelectionMode:  req.SelectionMode,
		Options:        req.Options,
	}
	if hasReservedPluginAttributeKey(attr) {
		writeError(w, http.StatusBadRequest, "attribute keys beginning with plugin. are reserved for plugins")
		return
	}

	if err := h.repos.Attributes.Create(r.Context(), attr); err != nil {
		writeAttributeError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, attr)
}

// UpdateAttribute applies a previewed definition change.
func (h *Handler) UpdateAttribute(w http.ResponseWriter, r *http.Request) {
	h.applyAttributeAction(w, r, "update_attribute")
}
func (h *Handler) DeleteAttribute(w http.ResponseWriter, r *http.Request) {
	h.applyAttributeAction(w, r, "delete_attribute")
}
func (h *Handler) PreviewAttributeImpact(w http.ResponseWriter, r *http.Request) {
	if !h.attributeAdmin(w, r) {
		return
	}
	id, err := parseUUID(r, "id")
	if err != nil {
		writeError(w, 400, "invalid attribute ID")
		return
	}
	var action repository.AttributeAction
	if err = decodeJSON(r, &action); err != nil {
		writeError(w, 400, "invalid impact request")
		return
	}
	impact, err := h.repos.Attributes.ChangeWithImpact(r.Context(), h.orgID, auth.GetUser(r.Context()).ID, id, action, "", true)
	if err != nil {
		writeAttributeError(w, err)
		return
	}
	writeJSON(w, 200, impact)
}
func (h *Handler) ListAttributeOptions(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		writeError(w, 400, "invalid attribute ID")
		return
	}
	a, err := h.repos.Attributes.GetByID(r.Context(), id)
	if err != nil {
		writeAttributeError(w, err)
		return
	}
	if a == nil || a.OrganizationID != h.orgID {
		writeError(w, 404, "attribute not found")
		return
	}
	if a.Options == nil {
		a.Options = []domain.AttributeOption{}
	}
	writeJSON(w, 200, a.Options)
}
func (h *Handler) AddAttributeOption(w http.ResponseWriter, r *http.Request) {
	if !h.attributeAdmin(w, r) {
		return
	}
	id, err := parseUUID(r, "id")
	if err != nil {
		writeError(w, 400, "invalid attribute ID")
		return
	}
	var option domain.AttributeOption
	if err = decodeJSON(r, &option); err != nil {
		writeError(w, 400, "invalid option")
		return
	}
	created, err := h.repos.Attributes.AddOption(r.Context(), h.orgID, id, option)
	if err != nil {
		writeAttributeError(w, err)
		return
	}
	writeJSON(w, 201, created)
}
func (h *Handler) UpdateAttributeOption(w http.ResponseWriter, r *http.Request) {
	h.applyAttributeAction(w, r, "update_option")
}
func (h *Handler) DeleteAttributeOption(w http.ResponseWriter, r *http.Request) {
	h.applyAttributeAction(w, r, "delete_option")
}
func (h *Handler) OrderAttributeOptions(w http.ResponseWriter, r *http.Request) {
	if !h.attributeAdmin(w, r) {
		return
	}
	id, err := parseUUID(r, "id")
	if err != nil {
		writeError(w, 400, "invalid attribute ID")
		return
	}
	var body struct {
		IDs []uuid.UUID `json:"ids"`
	}
	if err = decodeJSON(r, &body); err != nil {
		writeError(w, 400, "invalid option order")
		return
	}
	if err = h.repos.Attributes.OrderOptions(r.Context(), h.orgID, id, body.IDs); err != nil {
		writeAttributeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
func (h *Handler) applyAttributeAction(w http.ResponseWriter, r *http.Request, kind string) {
	if !h.attributeAdmin(w, r) {
		return
	}
	id, err := parseUUID(r, "id")
	if err != nil {
		writeError(w, 400, "invalid attribute ID")
		return
	}
	action := repository.AttributeAction{Action: kind}
	if strings.HasSuffix(kind, "option") {
		oid, err := parseUUID(r, "option_id")
		if err != nil {
			writeError(w, 400, "invalid option ID")
			return
		}
		action.OptionID = &oid
	}
	if strings.HasPrefix(kind, "update") {
		if err = decodeJSON(r, &action.Changes); err != nil {
			writeError(w, 400, "invalid changes")
			return
		}
	}
	_, err = h.repos.Attributes.ChangeWithImpact(r.Context(), h.orgID, auth.GetUser(r.Context()).ID, id, action, r.Header.Get("X-Impact-Token"), false)
	if err != nil {
		writeAttributeError(w, err)
		return
	}
	if strings.HasPrefix(kind, "delete") {
		w.WriteHeader(204)
		return
	}
	a, err := h.repos.Attributes.GetByID(r.Context(), id)
	if err != nil {
		writeAttributeError(w, err)
		return
	}
	writeJSON(w, 200, a)
}
func (h *Handler) attributeAdmin(w http.ResponseWriter, r *http.Request) bool {
	user := auth.GetUser(r.Context())
	if user == nil {
		writeError(w, 401, "authentication required")
		return false
	}
	if user.OrganizationID != h.orgID || !user.IsAdmin() {
		writeError(w, 403, "administrator access required")
		return false
	}
	return true
}
func writeAttributeError(w http.ResponseWriter, err error) {
	var problem *repository.AttributeError
	if errors.As(err, &problem) {
		writeJSON(w, problem.Status, map[string]string{"error": problem.Message, "code": problem.Code})
		return
	}
	var pg *pgconn.PgError
	if errors.As(err, &pg) && pg.Code == "23505" {
		writeError(w, 409, "field keys and option labels/values must be unique")
		return
	}
	writeError(w, 500, "failed to update attribute data")
}
