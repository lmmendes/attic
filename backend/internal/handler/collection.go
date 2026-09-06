package handler

import (
	"errors"
	"net/http"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/lmmendes/attic/internal/domain"
)

type CollectionRequest struct {
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
	Icon        string  `json:"icon"`
}

var collectionIconPattern = regexp.MustCompile(`^i-lucide-[a-z0-9]+(?:-[a-z0-9]+)*$`)

// ListCollections returns shared collections and active asset counts.
func (h *Handler) ListCollections(w http.ResponseWriter, r *http.Request) {
	collections, err := h.repos.Collections.List(r.Context(), h.orgID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list collections")
		return
	}
	writeJSON(w, http.StatusOK, collections)
}

// GetCollection returns a collection within the current workspace.
func (h *Handler) GetCollection(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid collection ID")
		return
	}
	c, err := h.repos.Collections.GetByID(r.Context(), h.orgID, id)
	if err != nil {
		writeCollectionError(w, err)
		return
	}
	if c == nil {
		writeCollectionError(w, pgx.ErrNoRows)
		return
	}
	writeJSON(w, http.StatusOK, c)
}

// CreateCollection creates a named group without changing any assets.
func (h *Handler) CreateCollection(w http.ResponseWriter, r *http.Request) {
	req, ok := decodeCollectionRequest(w, r)
	if !ok {
		return
	}
	c := &domain.Collection{OrganizationID: h.orgID, Name: req.Name, Description: req.Description, Icon: req.Icon}
	if err := h.repos.Collections.Create(r.Context(), c); err != nil {
		writeCollectionError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, c)
}

// UpdateCollection updates the shared name, description, and icon.
func (h *Handler) UpdateCollection(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid collection ID")
		return
	}
	req, ok := decodeCollectionRequest(w, r)
	if !ok {
		return
	}
	c, err := h.repos.Collections.GetByID(r.Context(), h.orgID, id)
	if err != nil {
		writeCollectionError(w, err)
		return
	}
	if c == nil {
		writeCollectionError(w, pgx.ErrNoRows)
		return
	}
	c.Name, c.Description, c.Icon = req.Name, req.Description, req.Icon
	if err := h.repos.Collections.Update(r.Context(), c); err != nil {
		writeCollectionError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, c)
}

// DeleteCollection removes the collection and its memberships, never its assets.
func (h *Handler) DeleteCollection(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid collection ID")
		return
	}
	if err := h.repos.Collections.Delete(r.Context(), h.orgID, id); err != nil {
		writeCollectionError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func decodeCollectionRequest(w http.ResponseWriter, r *http.Request) (CollectionRequest, bool) {
	var req CollectionRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return req, false
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" || utf8.RuneCountInString(req.Name) > 255 {
		writeError(w, http.StatusBadRequest, "name must contain 1 to 255 characters")
		return req, false
	}
	if req.Description != nil {
		description := strings.TrimSpace(*req.Description)
		if utf8.RuneCountInString(description) > 2000 {
			writeError(w, http.StatusBadRequest, "description must be at most 2000 characters")
			return req, false
		}
		req.Description = &description
	}
	if req.Icon == "" {
		req.Icon = "i-lucide-library"
	}
	if len(req.Icon) > 100 || !collectionIconPattern.MatchString(req.Icon) {
		writeError(w, http.StatusBadRequest, "icon must be a Lucide icon name")
		return req, false
	}
	return req, true
}

func writeCollectionError(w http.ResponseWriter, err error) {
	var pgErr *pgconn.PgError
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		writeError(w, http.StatusNotFound, "collection not found")
	case errors.As(err, &pgErr) && pgErr.Code == "23505":
		writeError(w, http.StatusConflict, "a collection with this name already exists")
	default:
		writeError(w, http.StatusInternalServerError, "failed to save collection")
	}
}

func parseCollectionIDs(values []string) ([]uuid.UUID, error) {
	if values == nil {
		return nil, nil
	}
	ids := make([]uuid.UUID, 0, len(values))
	seen := make(map[uuid.UUID]bool)
	for _, value := range values {
		id, err := uuid.Parse(value)
		if err != nil || id == uuid.Nil {
			return nil, errors.New("invalid collection ID")
		}
		if !seen[id] {
			ids = append(ids, id)
			seen[id] = true
		}
	}
	return ids, nil
}
