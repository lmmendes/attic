package handler

import (
	"net/http"

	"github.com/lmmendes/attic/internal/domain"
)

type CreateLocationRequest struct {
	ParentID    *string `json:"parent_id,omitempty"`
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
}

type UpdateLocationRequest struct {
	ParentID    *string `json:"parent_id,omitempty"`
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
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
	if loc == nil {
		writeError(w, http.StatusNotFound, "location not found")
		return
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
	if loc == nil {
		writeError(w, http.StatusNotFound, "location not found")
		return
	}

	loc.Name = req.Name
	loc.Description = req.Description

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

	writeJSON(w, http.StatusOK, loc)
}

func (h *Handler) DeleteLocation(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid location ID")
		return
	}

	if err := h.repos.Locations.Delete(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete location")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
