package handler

import (
	"net/http"

	"github.com/mendelui/attic/internal/domain"
)

type CreateConditionRequest struct {
	Code        string  `json:"code"`
	Label       string  `json:"label"`
	Description *string `json:"description,omitempty"`
	SortOrder   int     `json:"sort_order"`
}

type UpdateConditionRequest struct {
	Code        string  `json:"code"`
	Label       string  `json:"label"`
	Description *string `json:"description,omitempty"`
	SortOrder   int     `json:"sort_order"`
}

func (h *Handler) ListConditions(w http.ResponseWriter, r *http.Request) {
	conditions, err := h.repos.Conditions.List(r.Context(), h.orgID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list conditions")
		return
	}

	if conditions == nil {
		conditions = []domain.Condition{}
	}

	writeJSON(w, http.StatusOK, conditions)
}

func (h *Handler) GetCondition(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid condition ID")
		return
	}

	cond, err := h.repos.Conditions.GetByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get condition")
		return
	}
	if cond == nil {
		writeError(w, http.StatusNotFound, "condition not found")
		return
	}

	writeJSON(w, http.StatusOK, cond)
}

func (h *Handler) CreateCondition(w http.ResponseWriter, r *http.Request) {
	var req CreateConditionRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Code == "" || req.Label == "" {
		writeError(w, http.StatusBadRequest, "code and label are required")
		return
	}

	cond := &domain.Condition{
		OrganizationID: h.orgID,
		Code:           req.Code,
		Label:          req.Label,
		Description:    req.Description,
		SortOrder:      req.SortOrder,
	}

	if err := h.repos.Conditions.Create(r.Context(), cond); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create condition")
		return
	}

	writeJSON(w, http.StatusCreated, cond)
}

func (h *Handler) UpdateCondition(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid condition ID")
		return
	}

	var req UpdateConditionRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	cond, err := h.repos.Conditions.GetByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get condition")
		return
	}
	if cond == nil {
		writeError(w, http.StatusNotFound, "condition not found")
		return
	}

	cond.Code = req.Code
	cond.Label = req.Label
	cond.Description = req.Description
	cond.SortOrder = req.SortOrder

	if err := h.repos.Conditions.Update(r.Context(), cond); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update condition")
		return
	}

	writeJSON(w, http.StatusOK, cond)
}

func (h *Handler) DeleteCondition(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid condition ID")
		return
	}

	if err := h.repos.Conditions.Delete(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete condition")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
