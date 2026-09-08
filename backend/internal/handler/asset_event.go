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

type AssetEventRequest struct {
	Title       string  `json:"title"`
	Description *string `json:"description,omitempty"`
	Icon        string  `json:"icon"`
	EventDate   string  `json:"event_date"`
}

type AssetEventResponse struct {
	ID          uuid.UUID `json:"id"`
	AssetID     uuid.UUID `json:"asset_id"`
	Title       string    `json:"title"`
	Description *string   `json:"description,omitempty"`
	Icon        string    `json:"icon"`
	EventDate   string    `json:"event_date"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

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

func (h *Handler) CreateAssetEvent(w http.ResponseWriter, r *http.Request) {
	assetID, ok := h.activeAssetID(w, r)
	if !ok {
		return
	}
	req, eventDate, ok := decodeAssetEventRequest(w, r)
	if !ok {
		return
	}
	event := &domain.AssetEvent{
		AssetID: assetID, Title: req.Title, Description: req.Description,
		Icon: req.Icon, EventDate: eventDate,
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
	req, eventDate, ok := decodeAssetEventRequest(w, r)
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
	event.Title, event.Description, event.Icon, event.EventDate = req.Title, req.Description, req.Icon, eventDate
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
	if req.Description != nil {
		description := strings.TrimSpace(*req.Description)
		if utf8.RuneCountInString(description) > 2000 {
			writeError(w, http.StatusBadRequest, "description must be at most 2000 characters")
			return req, time.Time{}, false
		}
		if description == "" {
			req.Description = nil
		} else {
			req.Description = &description
		}
	}
	if len(req.Icon) > 100 || !collectionIconPattern.MatchString(req.Icon) {
		writeError(w, http.StatusBadRequest, "icon must be a Lucide icon name")
		return req, time.Time{}, false
	}
	eventDate, err := time.Parse("2006-01-02", req.EventDate)
	if err != nil {
		writeError(w, http.StatusBadRequest, "event_date must be a valid date in YYYY-MM-DD format")
		return req, time.Time{}, false
	}
	return req, eventDate, true
}

func assetEventResponse(event *domain.AssetEvent) AssetEventResponse {
	return AssetEventResponse{
		ID: event.ID, AssetID: event.AssetID, Title: event.Title,
		Description: event.Description, Icon: event.Icon,
		EventDate: event.EventDate.Format("2006-01-02"),
		CreatedAt: event.CreatedAt, UpdatedAt: event.UpdatedAt,
	}
}
