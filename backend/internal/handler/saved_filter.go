package handler

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/lmmendes/attic/internal/auth"
	"github.com/lmmendes/attic/internal/domain"
	"github.com/lmmendes/attic/internal/repository"
)

type assetSearchRequest struct {
	Criteria domain.FilterCriteria `json:"criteria"`
	Limit    int                   `json:"limit"`
	Offset   int                   `json:"offset"`
}

type savedFilterRequest struct {
	Name     string                 `json:"name"`
	Pinned   *bool                  `json:"pinned,omitempty"`
	Criteria *domain.FilterCriteria `json:"criteria,omitempty"`
}

func (h *Handler) SearchAssets(w http.ResponseWriter, r *http.Request) {
	var req assetSearchRequest
	if !decodeFilterRequest(w, r, &req) {
		return
	}
	if req.Limit <= 0 || req.Limit > 100 {
		req.Limit = 20
	}
	if req.Offset < 0 {
		req.Offset = 0
	}
	features, err := h.features(r)
	if err != nil {
		writeFilterError(w, err)
		return
	}
	page := domain.Pagination{Limit: req.Limit, Offset: req.Offset}
	assets, total, err := h.repos.Assets.List(r.Context(), h.orgID, domain.AssetFilter{Criteria: &req.Criteria, Features: features}, page)
	if err != nil {
		writeFilterError(w, err)
		return
	}
	h.writeAssetList(w, r, assets, total, page)
}

func (h *Handler) ListSavedFilters(w http.ResponseWriter, r *http.Request) {
	u := auth.GetUser(r.Context())
	if u == nil {
		writeError(w, 401, "not authenticated")
		return
	}
	items, err := h.repos.SavedFilters.List(r.Context(), h.orgID, u.ID)
	if err != nil {
		writeFilterError(w, err)
		return
	}
	features, err := h.features(r)
	if err != nil {
		writeFilterError(w, err)
		return
	}
	validator, err := h.repos.Assets.NewFilterValidator(r.Context(), h.orgID, *features)
	if err != nil {
		writeFilterError(w, err)
		return
	}
	for i := range items {
		_, _, err := validator.Compile(items[i].Criteria, 1)
		var invalid *repository.FilterError
		if errors.As(err, &invalid) {
			items[i].Issues = invalid.Issues
		} else if err != nil {
			writeFilterError(w, err)
			return
		}
	}
	writeJSON(w, 200, items)
}

func (h *Handler) GetSavedFilter(w http.ResponseWriter, r *http.Request) {
	item := h.loadSavedFilter(w, r)
	if item == nil {
		return
	}
	if err := h.savedFilterIssues(r, item); err != nil {
		writeFilterError(w, err)
		return
	}
	writeJSON(w, 200, item)
}

func (h *Handler) CreateSavedFilter(w http.ResponseWriter, r *http.Request) {
	u := auth.GetUser(r.Context())
	if u == nil {
		writeError(w, 401, "not authenticated")
		return
	}
	var req savedFilterRequest
	if !decodeFilterRequest(w, r, &req) || !validFilterName(w, &req.Name) {
		return
	}
	if req.Criteria == nil {
		writeError(w, 400, "criteria is required")
		return
	}
	item := domain.SavedFilter{OrganizationID: h.orgID, UserID: u.ID, Name: req.Name, Criteria: *req.Criteria}
	if req.Pinned != nil {
		item.Pinned = *req.Pinned
	}
	if err := h.validateCriteria(r, item.Criteria); err != nil {
		writeFilterError(w, err)
		return
	}
	if err := h.repos.SavedFilters.Create(r.Context(), &item); err != nil {
		writeFilterError(w, err)
		return
	}
	writeJSON(w, 201, item)
}

func (h *Handler) UpdateSavedFilter(w http.ResponseWriter, r *http.Request) {
	item := h.loadSavedFilter(w, r)
	if item == nil {
		return
	}
	var req savedFilterRequest
	if !decodeFilterRequest(w, r, &req) || !validFilterName(w, &req.Name) {
		return
	}
	item.Name = req.Name
	if req.Pinned != nil && *req.Pinned != item.Pinned {
		item.Pinned = *req.Pinned
	}
	if req.Criteria != nil {
		if err := h.validateCriteria(r, *req.Criteria); err != nil {
			writeFilterError(w, err)
			return
		}
		item.Criteria = *req.Criteria
	}
	if err := h.repos.SavedFilters.Update(r.Context(), item); err != nil {
		writeFilterError(w, err)
		return
	}
	if err := h.savedFilterIssues(r, item); err != nil {
		writeFilterError(w, err)
		return
	}
	writeJSON(w, 200, item)
}

func (h *Handler) DeleteSavedFilter(w http.ResponseWriter, r *http.Request) {
	item := h.loadSavedFilter(w, r)
	if item == nil {
		return
	}
	if err := h.repos.SavedFilters.Delete(r.Context(), h.orgID, item.UserID, item.ID); err != nil {
		writeFilterError(w, err)
		return
	}
	w.WriteHeader(204)
}

func (h *Handler) loadSavedFilter(w http.ResponseWriter, r *http.Request) *domain.SavedFilter {
	u := auth.GetUser(r.Context())
	if u == nil {
		writeError(w, 401, "not authenticated")
		return nil
	}
	id, err := parseUUID(r, "id")
	if err != nil {
		writeError(w, 400, "invalid saved filter ID")
		return nil
	}
	item, err := h.repos.SavedFilters.GetByID(r.Context(), h.orgID, u.ID, id)
	if err != nil {
		writeFilterError(w, err)
		return nil
	}
	if item == nil {
		writeError(w, 404, "saved filter not found")
	}
	return item
}

func (h *Handler) validateCriteria(r *http.Request, criteria domain.FilterCriteria) error {
	f, err := h.features(r)
	if err != nil {
		return err
	}
	_, _, err = h.repos.Assets.CompileCriteria(r.Context(), h.orgID, criteria, *f, 1)
	return err
}

func (h *Handler) savedFilterIssues(r *http.Request, item *domain.SavedFilter) error {
	err := h.validateCriteria(r, item.Criteria)
	var invalid *repository.FilterError
	if errors.As(err, &invalid) {
		item.Issues = invalid.Issues
		return nil
	}
	return err
}

func decodeFilterRequest(w http.ResponseWriter, r *http.Request, target any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 64*1024)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	decoder.UseNumber()
	if err := decoder.Decode(target); err != nil {
		writeError(w, 400, "invalid filter request (maximum 64 KiB)")
		return false
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		writeError(w, 400, "request must contain one JSON object")
		return false
	}
	return true
}

func validFilterName(w http.ResponseWriter, name *string) bool {
	*name = strings.TrimSpace(*name)
	if *name == "" || utf8.RuneCountInString(*name) > 100 {
		writeError(w, 400, "name must contain between 1 and 100 characters")
		return false
	}
	return true
}

func writeFilterError(w http.ResponseWriter, err error) {
	if errors.Is(err, repository.ErrPinnedFilterLimit) {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	var invalid *repository.FilterError
	if errors.As(err, &invalid) {
		writeJSON(w, 400, map[string]any{"error": invalid.Error(), "issues": invalid.Issues})
		return
	}
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, 404, "saved filter not found")
		return
	}
	slog.Error("asset filter request failed", "error", err)
	writeError(w, 500, "failed to process asset filter")
}
