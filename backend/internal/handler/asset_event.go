package handler

import (
	"errors"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/lmmendes/attic/internal/domain"
)

// AssetEventRequest is the editable payload for a custom asset event.
type AssetEventRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
	OccurredAt  string `json:"occurred_at"`
}

// AssetEventResponse is the API representation of a custom asset event.
type AssetEventResponse struct {
	ID          uuid.UUID `json:"id"`
	AssetID     uuid.UUID `json:"asset_id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Icon        string    `json:"icon"`
	OccurredAt  time.Time `json:"occurred_at"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ListAssetEvents returns custom history events for an asset.
func (h *Handler) ListAssetEvents(w http.ResponseWriter, r *http.Request) {
	assetID, ok := h.activeAssetID(w, r)
	if !ok {
		return
	}
	events, err := h.repos.AssetEvents.ListByAsset(r.Context(), h.orgID, assetID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list asset events")
		return
	}
	responses := make([]AssetEventResponse, 0, len(events))
	for i := range events {
		responses = append(responses, assetEventResponse(&events[i]))
	}
	writeJSON(w, http.StatusOK, responses)
}

// CreateAssetEvent adds a custom history event to an asset.
func (h *Handler) CreateAssetEvent(w http.ResponseWriter, r *http.Request) {
	assetID, ok := h.activeAssetID(w, r)
	if !ok {
		return
	}
	req, occurredAt, ok := decodeAssetEventRequest(w, r)
	if !ok {
		return
	}
	event := &domain.AssetEvent{
		AssetID: assetID, Title: req.Title, Description: req.Description,
		Icon: req.Icon, OccurredAt: occurredAt,
	}
	if err := h.repos.AssetEvents.Create(r.Context(), h.orgID, event); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "asset not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to create asset event")
		return
	}
	writeJSON(w, http.StatusCreated, assetEventResponse(event))
}

// UpdateAssetEvent replaces an existing custom history event.
func (h *Handler) UpdateAssetEvent(w http.ResponseWriter, r *http.Request) {
	assetID, ok := h.activeAssetID(w, r)
	if !ok {
		return
	}
	eventID, err := parseUUID(r, "eventId")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid event ID")
		return
	}
	req, occurredAt, ok := decodeAssetEventRequest(w, r)
	if !ok {
		return
	}
	event, err := h.repos.AssetEvents.GetByID(r.Context(), h.orgID, assetID, eventID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get asset event")
		return
	}
	if event == nil {
		writeError(w, http.StatusNotFound, "asset event not found")
		return
	}
	event.Title, event.Description = req.Title, req.Description
	event.Icon, event.OccurredAt = req.Icon, occurredAt
	if err := h.repos.AssetEvents.Update(r.Context(), h.orgID, event); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "asset event not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to update asset event")
		return
	}
	writeJSON(w, http.StatusOK, assetEventResponse(event))
}

// DeleteAssetEvent removes a custom history event from an asset.
func (h *Handler) DeleteAssetEvent(w http.ResponseWriter, r *http.Request) {
	assetID, ok := h.activeAssetID(w, r)
	if !ok {
		return
	}
	eventID, err := parseUUID(r, "eventId")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid event ID")
		return
	}
	if err := h.repos.AssetEvents.Delete(r.Context(), h.orgID, assetID, eventID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "asset event not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to delete asset event")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// activeAssetID resolves an active asset in the current organization.
func (h *Handler) activeAssetID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	assetID, err := parseUUID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid asset ID")
		return uuid.Nil, false
	}
	asset, err := h.repos.Assets.GetByID(r.Context(), assetID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to check asset")
		return uuid.Nil, false
	}
	if asset == nil || asset.OrganizationID != h.orgID {
		writeError(w, http.StatusNotFound, "asset not found")
		return uuid.Nil, false
	}
	return assetID, true
}

// decodeAssetEventRequest validates and normalizes an event request.
func decodeAssetEventRequest(w http.ResponseWriter, r *http.Request) (AssetEventRequest, time.Time, bool) {
	var req AssetEventRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return req, time.Time{}, false
	}
	req.Title = strings.TrimSpace(req.Title)
	if req.Title == "" || utf8.RuneCountInString(req.Title) > 255 {
		writeError(w, http.StatusBadRequest, "title must contain 1 to 255 characters")
		return req, time.Time{}, false
	}
	req.Description = strings.TrimSpace(req.Description)
	if utf8.RuneCountInString(req.Description) > 2000 {
		writeError(w, http.StatusBadRequest, "description must contain at most 2000 characters")
		return req, time.Time{}, false
	}
	if len(req.Icon) > 100 || !collectionIconPattern.MatchString(req.Icon) {
		writeError(w, http.StatusBadRequest, "icon must be a Lucide icon name")
		return req, time.Time{}, false
	}
	occurredAt, err := time.Parse(time.RFC3339, req.OccurredAt)
	if err != nil {
		writeError(w, http.StatusBadRequest, "occurred_at must be a valid RFC3339 date-time")
		return req, time.Time{}, false
	}
	return req, occurredAt, true
}

// assetEventResponse maps a domain event to its API representation.
func assetEventResponse(event *domain.AssetEvent) AssetEventResponse {
	return AssetEventResponse{
		ID: event.ID, AssetID: event.AssetID, Title: event.Title,
		Description: event.Description, Icon: event.Icon,
		OccurredAt: event.OccurredAt,
		CreatedAt:  event.CreatedAt, UpdatedAt: event.UpdatedAt,
	}
}
