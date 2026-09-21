package handler

import (
	"errors"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/lmmendes/attic/internal/domain"
)

type tagRequest struct {
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
}

func (h *Handler) ListTags(w http.ResponseWriter, r *http.Request) {
	tags, err := h.repos.Tags.List(r.Context(), h.orgID)
	if err != nil {
		writeTagError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, tags)
}

func (h *Handler) GetTag(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid tag ID")
		return
	}
	tag, err := h.repos.Tags.GetByID(r.Context(), h.orgID, id)
	if err != nil {
		writeTagError(w, err)
		return
	}
	if tag == nil {
		writeTagError(w, pgx.ErrNoRows)
		return
	}
	writeJSON(w, http.StatusOK, tag)
}

func (h *Handler) CreateTag(w http.ResponseWriter, r *http.Request) {
	req, ok := decodeTagRequest(w, r)
	if !ok {
		return
	}
	tag := &domain.Tag{OrganizationID: h.orgID, Name: req.Name, Description: req.Description}
	if err := h.repos.Tags.Create(r.Context(), tag); err != nil {
		writeTagError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, tag)
}

func (h *Handler) UpdateTag(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid tag ID")
		return
	}
	req, ok := decodeTagRequest(w, r)
	if !ok {
		return
	}
	tag, err := h.repos.Tags.GetByID(r.Context(), h.orgID, id)
	if err != nil {
		writeTagError(w, err)
		return
	}
	if tag == nil {
		writeTagError(w, pgx.ErrNoRows)
		return
	}
	tag.Name, tag.Description = req.Name, req.Description
	if err := h.repos.Tags.Update(r.Context(), tag); err != nil {
		writeTagError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, tag)
}

func (h *Handler) DeleteTag(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid tag ID")
		return
	}
	if err := h.repos.Tags.Delete(r.Context(), h.orgID, id); err != nil {
		writeTagError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func decodeTagRequest(w http.ResponseWriter, r *http.Request) (tagRequest, bool) {
	var req tagRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return req, false
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" || utf8.RuneCountInString(req.Name) > 100 {
		writeError(w, http.StatusBadRequest, "name must contain 1 to 100 characters")
		return req, false
	}
	if req.Description != nil {
		description := strings.TrimSpace(*req.Description)
		if utf8.RuneCountInString(description) > 2000 {
			writeError(w, http.StatusBadRequest, "description must be at most 2000 characters")
			return req, false
		}
		if description == "" {
			req.Description = nil
		} else {
			req.Description = &description
		}
	}
	return req, true
}

func writeTagError(w http.ResponseWriter, err error) {
	var pgErr *pgconn.PgError
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		writeError(w, http.StatusNotFound, "tag not found")
	case errors.As(err, &pgErr) && pgErr.Code == "23505":
		writeError(w, http.StatusConflict, "a tag with this name already exists")
	default:
		writeError(w, http.StatusInternalServerError, "failed to save tag")
	}
}
