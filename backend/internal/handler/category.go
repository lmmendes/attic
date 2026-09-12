package handler

import (
	"context"
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/lmmendes/attic/internal/domain"
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

var errInvalidCategoryAttribute = errors.New("invalid category attribute")

func validateCategoryAttributeOwnership(attribute *domain.Attribute, orgID uuid.UUID, pluginsEnabled bool) error {
	if attribute == nil || attribute.OrganizationID != orgID {
		return errInvalidCategoryAttribute
	}
	if !pluginsEnabled && attribute.PluginID != nil {
		return errFeatureDisabled("plugins")
	}
	return nil
}

func (h *Handler) validateCategoryAttributeAssignments(ctx context.Context, attrs []AttributeAssignment, pluginsEnabled bool) ([]domain.CategoryAttributeAssignment, error) {
	assignments, err := parseAttributeAssignments(attrs)
	if err != nil {
		return nil, errInvalidCategoryAttribute
	}
	if pluginsEnabled {
		return assignments, nil
	}

	for _, assignment := range assignments {
		attribute, err := h.repos.Attributes.GetByID(ctx, assignment.AttributeID)
		if err != nil {
			return nil, err
		}
		if err := validateCategoryAttributeOwnership(attribute, h.orgID, pluginsEnabled); err != nil {
			return nil, err
		}
	}

	return assignments, nil
}

func writeCategoryAttributeValidationError(w http.ResponseWriter, err error) {
	var disabled featureDisabledError
	switch {
	case errors.As(err, &disabled):
		writeError(w, http.StatusForbidden, err.Error())
	case errors.Is(err, errInvalidCategoryAttribute):
		writeError(w, http.StatusBadRequest, "invalid attribute_id in attributes")
	default:
		writeError(w, http.StatusInternalServerError, "failed to validate category attributes")
	}
}

func preservePluginCategoryAssignments(assignments []domain.CategoryAttributeAssignment, existing []domain.CategoryAttribute) []domain.CategoryAttributeAssignment {
	for _, current := range existing {
		if current.Attribute == nil || current.Attribute.PluginID == nil {
			continue
		}
		assignments = append(assignments, domain.CategoryAttributeAssignment{
			AttributeID: current.AttributeID,
			Required:    current.Required,
			SortOrder:   current.SortOrder,
		})
	}
	return assignments
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
	features, featureErr := h.features(r)
	if featureErr != nil {
		writeError(w, http.StatusInternalServerError, "failed to read organization features")
		return
	}
	if !features.Plugins {
		categories = filterPluginCategories(categories)
	}
	if !features.Attributes {
		for i := range categories {
			clearCategoryAttributes(&categories[i])
		}
	}

	writeJSON(w, http.StatusOK, categories)
}

func (h *Handler) GetCategoryAssetCounts(w http.ResponseWriter, r *http.Request) {
	counts, err := h.repos.Categories.GetAssetCounts(r.Context(), h.orgID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get asset counts")
		return
	}
	pluginsEnabled, featureErr := h.featureEnabled(r, "plugins")
	if featureErr != nil {
		writeError(w, http.StatusInternalServerError, "failed to read organization features")
		return
	}
	if !pluginsEnabled {
		categories, err := h.repos.Categories.List(r.Context(), h.orgID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to filter asset counts")
			return
		}
		for _, category := range categories {
			if category.PluginID != nil {
				delete(counts, category.ID.String())
			}
		}
	}
	writeJSON(w, http.StatusOK, counts)
}

func (h *Handler) GetCategory(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid category ID")
		return
	}

	var cat *domain.Category
	if r.URL.Query().Get("inherited") == "true" {
		cat, err = h.repos.Categories.GetByIDWithInheritedAttributes(r.Context(), h.orgID, id)
	} else {
		cat, err = h.repos.Categories.GetByIDWithAttributes(r.Context(), id)
		if cat != nil && cat.OrganizationID != h.orgID {
			cat = nil
		}
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get category")
		return
	}
	if cat == nil {
		writeError(w, http.StatusNotFound, "category not found")
		return
	}
	features, featureErr := h.features(r)
	if featureErr != nil {
		writeError(w, http.StatusInternalServerError, "failed to read organization features")
		return
	}
	if !features.Plugins {
		if cat.PluginID != nil {
			writeError(w, http.StatusNotFound, "category not found")
			return
		}
		cat.Attributes = filterPluginCategoryAttributes(cat.Attributes)
	}
	if !features.Attributes {
		cat.Attributes = nil
	}

	writeJSON(w, http.StatusOK, cat)
}

func filterPluginCategories(categories []domain.Category) []domain.Category {
	filtered := make([]domain.Category, 0, len(categories))
	for _, category := range categories {
		if category.PluginID != nil {
			continue
		}
		category.Attributes = filterPluginCategoryAttributes(category.Attributes)
		category.Children = filterPluginCategories(category.Children)
		filtered = append(filtered, category)
	}
	return filtered
}

func filterPluginCategoryAttributes(attributes []domain.CategoryAttribute) []domain.CategoryAttribute {
	filtered := make([]domain.CategoryAttribute, 0, len(attributes))
	for _, assignment := range attributes {
		if assignment.Attribute != nil && assignment.Attribute.PluginID != nil {
			continue
		}
		filtered = append(filtered, assignment)
	}
	return filtered
}

func clearCategoryAttributes(category *domain.Category) {
	category.Attributes = nil
	for i := range category.Children {
		clearCategoryAttributes(&category.Children[i])
	}
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
	features, err := h.features(r)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to read organization features")
		return
	}
	if !features.Attributes && len(req.Attributes) > 0 {
		writeError(w, http.StatusForbidden, "attributes feature is not enabled")
		return
	}
	assignments, err := h.validateCategoryAttributeAssignments(r.Context(), req.Attributes, features.Plugins)
	if err != nil {
		writeCategoryAttributeValidationError(w, err)
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
		valid, err := h.repos.Categories.ValidateParent(r.Context(), h.orgID, uuid.Nil, parentID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to validate parent category")
			return
		}
		if !valid {
			writeError(w, http.StatusBadRequest, "parent category must belong to this workspace")
			return
		}
		cat.ParentID = &parentID
	}

	if err := h.repos.Categories.Create(r.Context(), cat); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create category")
		return
	}

	// Set attributes if provided
	if len(assignments) > 0 {
		if err := h.repos.Categories.SetAttributes(r.Context(), cat.ID, assignments); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to set category attributes")
			return
		}
	}

	// Fetch the category with attributes to return
	cat, err = h.repos.Categories.GetByIDWithAttributes(r.Context(), cat.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get category")
		return
	}
	if !features.Attributes {
		cat.Attributes = nil
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

	cat, err := h.repos.Categories.GetByIDWithAttributes(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get category")
		return
	}
	if cat == nil || cat.OrganizationID != h.orgID {
		writeError(w, http.StatusNotFound, "category not found")
		return
	}
	features, featureErr := h.features(r)
	if featureErr != nil {
		writeError(w, http.StatusInternalServerError, "failed to read organization features")
		return
	} else if !features.Plugins && cat.PluginID != nil {
		writeError(w, http.StatusNotFound, "category not found")
		return
	}

	var assignments []domain.CategoryAttributeAssignment
	if features.Attributes {
		assignments, err = h.validateCategoryAttributeAssignments(r.Context(), req.Attributes, features.Plugins)
		if err != nil {
			writeCategoryAttributeValidationError(w, err)
			return
		}
		if !features.Plugins {
			assignments = preservePluginCategoryAssignments(assignments, cat.Attributes)
		}
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
		valid, err := h.repos.Categories.ValidateParent(r.Context(), h.orgID, cat.ID, parentID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to validate parent category")
			return
		}
		if !valid {
			writeError(w, http.StatusBadRequest, "parent category cannot be itself, a descendant, or outside this workspace")
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

	if features.Attributes {
		if err := h.repos.Categories.SetAttributes(r.Context(), cat.ID, assignments); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to set category attributes")
			return
		}
	}

	// Fetch the category with attributes to return
	cat, err = h.repos.Categories.GetByIDWithAttributes(r.Context(), cat.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get category")
		return
	}
	if !features.Plugins {
		cat.Attributes = filterPluginCategoryAttributes(cat.Attributes)
	}
	if !features.Attributes {
		cat.Attributes = nil
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
	if cat == nil || cat.OrganizationID != h.orgID {
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
