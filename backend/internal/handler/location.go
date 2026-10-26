package handler

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/lmmendes/attic/internal/domain"
)

type CreateLocationRequest struct {
	ParentID    *string `json:"parent_id,omitempty"`
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
	Icon        *string `json:"icon,omitempty"`
}

type UpdateLocationRequest struct {
	ParentID    *string `json:"parent_id,omitempty"`
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
	Icon        *string `json:"icon,omitempty"`
}

func (h *Handler) ListLocations(w http.ResponseWriter, r *http.Request) {
	tree := r.URL.Query().Get("tree") == "true"

	var locations []domain.Location
	var err error

	if tree {
		locations, err = h.repos.Locations.ListTree(r.Context(), h.orgID)
	} else {
		locations, err = h.repos.Locations.List(r.Context(), h.orgID)
	}

	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list locations")
		return
	}

	if locations == nil {
		locations = []domain.Location{}
	}

	for i := range locations {
		if err := h.populateLocationPhoto(r, &locations[i]); err != nil {
			slog.Warn("failed to read location photo", "location_id", locations[i].ID, "error", err)
		}
	}
	writeJSON(w, http.StatusOK, locations)
}

func (h *Handler) GetLocation(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid location ID")
		return
	}

	loc, err := h.repos.Locations.GetByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get location")
		return
	}
	if loc == nil || loc.OrganizationID != h.orgID {
		writeError(w, http.StatusNotFound, "location not found")
		return
	}

	if err := h.populateLocationPhoto(r, loc); err != nil {
		slog.Warn("failed to read location photo", "location_id", loc.ID, "error", err)
	}
	writeJSON(w, http.StatusOK, loc)
}

func (h *Handler) CreateLocation(w http.ResponseWriter, r *http.Request) {
	var req CreateLocationRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	loc := &domain.Location{
		OrganizationID: h.orgID,
		Name:           req.Name,
		Description:    req.Description,
		Icon:           req.Icon,
	}

	if req.ParentID != nil {
		parentID, err := parseUUIDString(*req.ParentID)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid parent_id")
			return
		}
		loc.ParentID = &parentID
	}

	if err := h.repos.Locations.Create(r.Context(), loc); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create location")
		return
	}

	writeJSON(w, http.StatusCreated, loc)
}

func (h *Handler) UpdateLocation(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid location ID")
		return
	}

	var req UpdateLocationRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	loc, err := h.repos.Locations.GetByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get location")
		return
	}
	if loc == nil || loc.OrganizationID != h.orgID {
		writeError(w, http.StatusNotFound, "location not found")
		return
	}

	loc.Name = req.Name
	loc.Description = req.Description
	loc.Icon = req.Icon

	if req.ParentID != nil {
		parentID, err := parseUUIDString(*req.ParentID)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid parent_id")
			return
		}
		loc.ParentID = &parentID
	} else {
		loc.ParentID = nil
	}

	if err := h.repos.Locations.Update(r.Context(), loc); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update location")
		return
	}

	if err := h.populateLocationPhoto(r, loc); err != nil {
		slog.Warn("failed to read location photo", "location_id", loc.ID, "error", err)
	}
	writeJSON(w, http.StatusOK, loc)
}

func (h *Handler) DeleteLocation(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid location ID")
		return
	}

	old, err := h.repos.Locations.DeleteWithPhotos(r.Context(), h.orgID, id)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, 404, "location not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete location")
		return
	}

	h.cleanupLocationPhotos(r.Context(), old)
	w.WriteHeader(http.StatusNoContent)
}
