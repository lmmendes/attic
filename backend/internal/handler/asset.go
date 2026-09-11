package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lmmendes/attic/internal/domain"
	"github.com/lmmendes/attic/internal/repository"
)

const maxAssetQuantity = 1000000
const uncategorizedCategoryFilter = "uncategorized"

type CreateAssetRequest struct {
	CollectionIDs []string        `json:"collection_ids,omitempty"`
	CategoryID    *string         `json:"category_id,omitempty"`
	LocationID    *string         `json:"location_id,omitempty"`
	ConditionID   *string         `json:"condition_id,omitempty"`
	Name          string          `json:"name"`
	Description   *string         `json:"description,omitempty"`
	Quantity      int             `json:"quantity"`
	Attributes    json.RawMessage `json:"attributes,omitempty"`
	PurchaseAt    *string         `json:"purchase_at,omitempty"`
	PurchasePrice *float64        `json:"purchase_price,omitempty"`
	PurchaseNote  *string         `json:"purchase_note,omitempty"`
	Notes         *string         `json:"notes,omitempty"`
}

type UpdateAssetRequest struct {
	CollectionIDs []string        `json:"collection_ids,omitempty"`
	CategoryID    *string         `json:"category_id,omitempty"`
	LocationID    *string         `json:"location_id,omitempty"`
	ConditionID   *string         `json:"condition_id,omitempty"`
	Name          string          `json:"name"`
	Description   *string         `json:"description,omitempty"`
	Quantity      int             `json:"quantity"`
	Attributes    json.RawMessage `json:"attributes,omitempty"`
	PurchaseAt    *string         `json:"purchase_at,omitempty"`
	PurchasePrice *float64        `json:"purchase_price,omitempty"`
	PurchaseNote  *string         `json:"purchase_note,omitempty"`
	Notes         *string         `json:"notes,omitempty"`
}

type AssetListResponse struct {
	Assets []AssetWithImageURL `json:"assets"`
	Total  int                 `json:"total"`
	Limit  int                 `json:"limit"`
	Offset int                 `json:"offset"`
}

type AssetWithImageURL struct {
	domain.Asset
	MainAttachmentURL string `json:"main_attachment_url,omitempty"`
}

type AssetDetailResponse struct {
	domain.Asset
	MainAttachmentURL string `json:"main_attachment_url,omitempty"`
}

func (h *Handler) ListAssets(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	if enabled, err := h.featureEnabled(r, "locations"); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to read organization features")
		return
	} else if !enabled && q.Get("location_id") != "" {
		writeError(w, http.StatusForbidden, "locations feature is not enabled")
		return
	}
	if enabled, err := h.featureEnabled(r, "collections"); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to read organization features")
		return
	} else if !enabled && q.Get("collection_id") != "" {
		writeError(w, http.StatusForbidden, "collections feature is not enabled")
		return
	}
	if enabled, err := h.featureEnabled(r, "categories"); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to read organization features")
		return
	} else if !enabled && q.Get("category_id") != "" {
		writeError(w, http.StatusForbidden, "categories feature is not enabled")
		return
	}
	if enabled, err := h.featureEnabled(r, "conditions"); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to read organization features")
		return
	} else if !enabled && q.Get("condition_id") != "" {
		writeError(w, http.StatusForbidden, "conditions feature is not enabled")
		return
	}

	limit, _ := strconv.Atoi(q.Get("limit"))
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	offset, _ := strconv.Atoi(q.Get("offset"))
	if offset < 0 {
		offset = 0
	}

	filter := domain.AssetFilter{
		Query: q.Get("q"),
	}

	if catID := q.Get("category_id"); catID != "" {
		if catID == uncategorizedCategoryFilter {
			filter.Uncategorized = true
		} else if id, err := uuid.Parse(catID); err == nil {
			filter.CategoryID = &id
		}
	}
	if locID := q.Get("location_id"); locID != "" {
		if id, err := uuid.Parse(locID); err == nil {
			filter.LocationID = &id
		}
	}
	if condID := q.Get("condition_id"); condID != "" {
		if id, err := uuid.Parse(condID); err == nil {
			filter.ConditionID = &id
		}
	}

	page := domain.Pagination{Limit: limit, Offset: offset}
	if value := q.Get("collection_id"); value != "" {
		id, err := uuid.Parse(value)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid collection ID")
			return
		}
		filter.CollectionID = &id
	}
	assets, total, err := h.repos.Assets.List(r.Context(), h.orgID, filter, page)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list assets")
		return
	}

	if assets == nil {
		assets = []domain.Asset{}
	}

	// Generate presigned URLs for main attachments
	assetsWithURLs := make([]AssetWithImageURL, len(assets))
	for i, asset := range assets {
		if err := h.sanitizeAsset(r, &asset); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to apply organization features")
			return
		}
		assetsWithURLs[i] = AssetWithImageURL{Asset: asset}
		if asset.MainAttachment != nil && h.storage != nil {
			url, err := h.storage.GetPresignedURL(r.Context(), asset.MainAttachment.FileKey, 15*time.Minute)
			if err == nil {
				assetsWithURLs[i].MainAttachmentURL = url
			}
		}
	}

	writeJSON(w, http.StatusOK, AssetListResponse{
		Assets: assetsWithURLs,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	})
}

func (h *Handler) GetAsset(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid asset ID")
		return
	}

	asset, err := h.repos.Assets.GetByIDFull(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get asset")
		return
	}
	if asset == nil || asset.OrganizationID != h.orgID {
		writeError(w, http.StatusNotFound, "asset not found")
		return
	}

	// Generate presigned URL for main attachment
	if err := h.sanitizeAsset(r, asset); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to apply organization features")
		return
	}
	response := AssetDetailResponse{Asset: *asset}
	if asset.MainAttachment != nil && h.storage != nil {
		url, err := h.storage.GetPresignedURL(r.Context(), asset.MainAttachment.FileKey, 15*time.Minute)
		if err == nil {
			response.MainAttachmentURL = url
		}
	}

	writeJSON(w, http.StatusOK, response)
}

func (h *Handler) CreateAsset(w http.ResponseWriter, r *http.Request) {
	var req CreateAssetRequest
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
	collectionIDs, err := parseCollectionIDs(req.CollectionIDs)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.rejectDisabledAssetFields(r, req.CategoryID, req.LocationID, req.ConditionID, collectionIDs, req.Attributes); err != nil {
		if _, ok := err.(featureDisabledError); ok {
			writeError(w, http.StatusForbidden, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to read organization features")
		return
	}

	var categoryID *uuid.UUID
	if req.CategoryID != nil && *req.CategoryID != "" {
		id, err := uuid.Parse(*req.CategoryID)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid category_id")
			return
		}
		categoryID = &id
	}
	if message, err := h.validateAssetCategory(r.Context(), categoryID, req.Attributes, features.Attributes, features.Plugins); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to validate category attributes")
		return
	} else if message != "" {
		writeError(w, http.StatusBadRequest, message)
		return
	}

	asset := &domain.Asset{
		OrganizationID: h.orgID,
		CategoryID:     categoryID,
		Name:           req.Name,
		Description:    req.Description,
		Quantity:       req.Quantity,
		Attributes:     req.Attributes,
	}
	if categoryID == nil {
		asset.Attributes = nil
	}

	if asset.Quantity <= 0 {
		asset.Quantity = 1
	}
	if asset.Quantity > maxAssetQuantity {
		writeError(w, http.StatusBadRequest, "quantity exceeds maximum allowed value")
		return
	}

	if req.LocationID != nil {
		if id, err := parseUUIDString(*req.LocationID); err == nil {
			asset.LocationID = &id
		}
	}
	if req.ConditionID != nil {
		if id, err := parseUUIDString(*req.ConditionID); err == nil {
			asset.ConditionID = &id
		}
	}
	if req.PurchaseAt != nil && *req.PurchaseAt != "" {
		if t, err := time.Parse("2006-01-02", *req.PurchaseAt); err == nil {
			asset.PurchaseAt = &t
		}
	}
	asset.PurchasePrice = req.PurchasePrice
	asset.PurchaseNote = req.PurchaseNote
	asset.Notes = req.Notes

	asset.CollectionIDs = collectionIDs
	if err := h.repos.Assets.Create(r.Context(), asset); err != nil {
		if errors.Is(err, repository.ErrInvalidCollections) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to create asset")
		return
	}

	writeJSON(w, http.StatusCreated, asset)
}

func (h *Handler) UpdateAsset(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid asset ID")
		return
	}

	var req UpdateAssetRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	asset, err := h.repos.Assets.GetByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get asset")
		return
	}
	if asset == nil || asset.OrganizationID != h.orgID {
		writeError(w, http.StatusNotFound, "asset not found")
		return
	}
	features, err := h.features(r)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to read organization features")
		return
	}
	collectionIDs, err := parseCollectionIDs(req.CollectionIDs)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.rejectDisabledAssetFields(r, req.CategoryID, req.LocationID, req.ConditionID, collectionIDs, req.Attributes); err != nil {
		if _, ok := err.(featureDisabledError); ok {
			writeError(w, http.StatusForbidden, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to read organization features")
		return
	}

	preserveHiddenPluginCategory := false
	if features.Categories && !features.Plugins && req.CategoryID == nil && asset.CategoryID != nil {
		existingCategory, categoryErr := h.repos.Categories.GetByID(r.Context(), *asset.CategoryID)
		if categoryErr != nil {
			writeError(w, http.StatusInternalServerError, "failed to validate existing category")
			return
		}
		preserveHiddenPluginCategory = shouldPreserveHiddenPluginCategory(features, req.CategoryID, existingCategory)
	}

	categoryID := asset.CategoryID
	if features.Categories {
		if !preserveHiddenPluginCategory {
			categoryID = nil
		}
		if req.CategoryID != nil && *req.CategoryID != "" {
			id, parseErr := uuid.Parse(*req.CategoryID)
			if parseErr != nil {
				writeError(w, http.StatusBadRequest, "invalid category_id")
				return
			}
			categoryID = &id
		}
	}
	attributesForValidation := asset.Attributes
	if features.Attributes {
		attributesForValidation = req.Attributes
	}
	if message, err := h.validateAssetCategory(r.Context(), categoryID, attributesForValidation, features.Attributes, features.Plugins || !features.Categories || preserveHiddenPluginCategory); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to validate category attributes")
		return
	} else if message != "" {
		writeError(w, http.StatusBadRequest, message)
		return
	}

	asset.CategoryID = categoryID
	asset.Name = req.Name
	asset.Description = req.Description
	asset.Quantity = req.Quantity
	if asset.Quantity <= 0 {
		asset.Quantity = 1
	}
	if asset.Quantity > maxAssetQuantity {
		writeError(w, http.StatusBadRequest, "quantity exceeds maximum allowed value")
		return
	}
	if features.Attributes {
		if features.Plugins {
			asset.Attributes = req.Attributes
		} else {
			asset.Attributes, err = mergePreservedPluginAttributes(asset.Attributes, req.Attributes)
			if err != nil {
				writeError(w, http.StatusBadRequest, "attributes must be a JSON object")
				return
			}
		}
		if categoryID == nil {
			asset.Attributes = nil
		}
	}

	if features.Locations && req.LocationID != nil {
		if id, err := parseUUIDString(*req.LocationID); err == nil {
			asset.LocationID = &id
		}
	} else if features.Locations {
		asset.LocationID = nil
	}
	if features.Conditions && req.ConditionID != nil {
		if id, err := parseUUIDString(*req.ConditionID); err == nil {
			asset.ConditionID = &id
		}
	} else if features.Conditions {
		asset.ConditionID = nil
	}
	if req.PurchaseAt != nil && *req.PurchaseAt != "" {
		if t, err := time.Parse("2006-01-02", *req.PurchaseAt); err == nil {
			asset.PurchaseAt = &t
		}
	} else {
		asset.PurchaseAt = nil
	}
	asset.PurchasePrice = req.PurchasePrice
	asset.PurchaseNote = req.PurchaseNote
	asset.Notes = req.Notes
	if features.Collections {
		asset.CollectionIDs = collectionIDs
	}

	if err := h.repos.Assets.Update(r.Context(), asset); err != nil {
		if errors.Is(err, repository.ErrInvalidCollections) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to update asset")
		return
	}

	writeJSON(w, http.StatusOK, asset)
}

func shouldPreserveHiddenPluginCategory(features *domain.OrganizationFeatures, requestedCategoryID *string, existingCategory *domain.Category) bool {
	return features.Categories &&
		!features.Plugins &&
		requestedCategoryID == nil &&
		existingCategory != nil &&
		existingCategory.PluginID != nil
}

func (h *Handler) validateAssetCategory(ctx context.Context, categoryID *uuid.UUID, rawAttributes json.RawMessage, validateAttributes, pluginsEnabled bool) (string, error) {
	if categoryID == nil {
		return "", nil
	}
	category, err := h.repos.Categories.GetByIDWithInheritedAttributes(ctx, h.orgID, *categoryID)
	if err != nil {
		return "", err
	}
	if category == nil {
		return "category does not exist in this workspace", nil
	}
	if !pluginsEnabled && category.PluginID != nil {
		return "category is not available while plugins are disabled", nil
	}
	if !validateAttributes {
		return "", nil
	}
	missing, err := missingRequiredCategoryAttributes(category, rawAttributes)
	if err != nil {
		return "attributes must be a JSON object", nil
	}
	if len(missing) > 0 {
		return fmt.Sprintf("required category attributes are missing: %s", strings.Join(missing, ", ")), nil
	}
	return "", nil
}

func missingRequiredCategoryAttributes(category *domain.Category, rawAttributes json.RawMessage) ([]string, error) {
	values := make(map[string]any)
	if len(rawAttributes) > 0 && string(rawAttributes) != "null" {
		if err := json.Unmarshal(rawAttributes, &values); err != nil {
			return nil, err
		}
	}
	missing := make([]string, 0)
	for _, assignment := range category.Attributes {
		if !assignment.Required || assignment.Attribute == nil {
			continue
		}
		value, ok := values[assignment.Attribute.Key]
		if !ok || value == nil {
			missing = append(missing, assignment.Attribute.Name)
			continue
		}
		if text, ok := value.(string); ok && strings.TrimSpace(text) == "" {
			missing = append(missing, assignment.Attribute.Name)
		}
	}
	return missing, nil
}

func (h *Handler) DeleteAsset(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid asset ID")
		return
	}

	if err := h.repos.Assets.Delete(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete asset")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func parseUUIDString(s string) (uuid.UUID, error) {
	return uuid.Parse(s)
}

type AssetStatsResponse struct {
	TotalValue float64 `json:"total_value"`
}

func (h *Handler) GetAssetStats(w http.ResponseWriter, r *http.Request) {
	filter := domain.AssetFilter{}
	if locID := r.URL.Query().Get("location_id"); locID != "" {
		enabled, featureErr := h.featureEnabled(r, "locations")
		if featureErr != nil {
			writeError(w, http.StatusInternalServerError, "failed to read organization features")
			return
		}
		if !enabled {
			writeError(w, http.StatusForbidden, "locations feature is not enabled")
			return
		}
		id, err := uuid.Parse(locID)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid location ID")
			return
		}
		filter.LocationID = &id
	}

	totalValue, err := h.repos.Assets.GetTotalValue(r.Context(), h.orgID, filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get asset stats")
		return
	}

	writeJSON(w, http.StatusOK, AssetStatsResponse{
		TotalValue: totalValue,
	})
}
