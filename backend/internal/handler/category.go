package handler

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/mendelui/attic/internal/domain"
)

// AttributeAssignment represents an attribute assignment in request body
type AttributeAssignment struct {
	AttributeID string `json:"attribute_id"`
	Required    bool   `json:"required"`
	SortOrder   int    `json:"sort_order"`
}

type CreateCategoryRequest struct {
	ParentID    *string               `json:"parent_id,omitempty"`
	Name        string                `json:"name"`
	Description *string               `json:"description,omitempty"`
	Icon        *string               `json:"icon,omitempty"`
	Attributes  []AttributeAssignment `json:"attributes,omitempty"`
}

type UpdateCategoryRequest struct {
	ParentID    *string               `json:"parent_id,omitempty"`
	Name        string                `json:"name"`
	Description *string               `json:"description,omitempty"`
	Icon        *string               `json:"icon,omitempty"`
	Attributes  []AttributeAssignment `json:"attributes,omitempty"`
}

func (h *Handler) ListCategories(w http.ResponseWriter, r *http.Request) {
	tree := r.URL.Query().Get("tree") == "true"

	var categories []domain.Category
	var err error

	if tree {
		categories, err = h.repos.Categories.ListTree(r.Context(), h.orgID)
	} else {
		categories, err = h.repos.Categories.List(r.Context(), h.orgID)
	}

	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list categories")
		return
	}

	if categories == nil {
		categories = []domain.Category{}
	}

	writeJSON(w, http.StatusOK, categories)
}

func (h *Handler) GetCategoryAssetCounts(w http.ResponseWriter, r *http.Request) {
	counts, err := h.repos.Categories.GetAssetCounts(r.Context(), h.orgID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get asset counts")
		return
	}
	writeJSON(w, http.StatusOK, counts)
}

func (h *Handler) GetCategory(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid category ID")
		return
	}

	cat, err := h.repos.Categories.GetByIDWithAttributes(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get category")
		return
	}
	if cat == nil {
		writeError(w, http.StatusNotFound, "category not found")
		return
	}

	writeJSON(w, http.StatusOK, cat)
}

func (h *Handler) CreateCategory(w http.ResponseWriter, r *http.Request) {
	var req CreateCategoryRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	cat := &domain.Category{
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
		cat.ParentID = &parentID
	}

	if err := h.repos.Categories.Create(r.Context(), cat); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create category")
		return
	}

	// Set attributes if provided
	if len(req.Attributes) > 0 {
		assignments, err := parseAttributeAssignments(req.Attributes)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid attribute_id in attributes")
			return
		}
		if err := h.repos.Categories.SetAttributes(r.Context(), cat.ID, assignments); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to set category attributes")
			return
		}
	}

	// Fetch the category with attributes to return
	cat, err := h.repos.Categories.GetByIDWithAttributes(r.Context(), cat.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get category")
		return
	}

	writeJSON(w, http.StatusCreated, cat)
}

func (h *Handler) UpdateCategory(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid category ID")
		return
	}

	var req UpdateCategoryRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	cat, err := h.repos.Categories.GetByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get category")
		return
	}
	if cat == nil {
		writeError(w, http.StatusNotFound, "category not found")
		return
	}

	cat.Name = req.Name
	cat.Description = req.Description
	cat.Icon = req.Icon

	if req.ParentID != nil {
		parentID, err := parseUUIDString(*req.ParentID)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid parent_id")
			return
		}
		cat.ParentID = &parentID
	} else {
		cat.ParentID = nil
	}

	if err := h.repos.Categories.Update(r.Context(), cat); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update category")
		return
	}

	// Update attributes - always set (even if empty to clear existing)
	assignments, err := parseAttributeAssignments(req.Attributes)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid attribute_id in attributes")
		return
	}
	if err := h.repos.Categories.SetAttributes(r.Context(), cat.ID, assignments); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to set category attributes")
		return
	}

	// Fetch the category with attributes to return
	cat, err = h.repos.Categories.GetByIDWithAttributes(r.Context(), cat.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get category")
		return
	}

	writeJSON(w, http.StatusOK, cat)
}

func (h *Handler) DeleteCategory(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid category ID")
		return
	}

	// Check if category exists and is not plugin-managed
	cat, err := h.repos.Categories.GetByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get category")
		return
	}
	if cat == nil {
		writeError(w, http.StatusNotFound, "category not found")
		return
	}
	if cat.PluginID != nil {
		writeError(w, http.StatusForbidden, "cannot delete plugin-managed category")
		return
	}

	if err := h.repos.Categories.Delete(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete category")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// parseAttributeAssignments converts request attribute assignments to domain assignments
func parseAttributeAssignments(attrs []AttributeAssignment) ([]domain.CategoryAttributeAssignment, error) {
	assignments := make([]domain.CategoryAttributeAssignment, 0, len(attrs))
	for _, a := range attrs {
		attrID, err := uuid.Parse(a.AttributeID)
		if err != nil {
			return nil, err
		}
		assignments = append(assignments, domain.CategoryAttributeAssignment{
			AttributeID: attrID,
			Required:    a.Required,
			SortOrder:   a.SortOrder,
		})
	}
	return assignments, nil
}
