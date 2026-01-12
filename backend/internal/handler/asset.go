package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/mendelui/attic/internal/domain"
)

type CreateAssetRequest struct {
	CategoryID    string          `json:"category_id"`
	LocationID    *string         `json:"location_id,omitempty"`
	ConditionID   *string         `json:"condition_id,omitempty"`
	CollectionID  *string         `json:"collection_id,omitempty"`
	Name          string          `json:"name"`
	Description   *string         `json:"description,omitempty"`
	Quantity      int             `json:"quantity"`
	Attributes    json.RawMessage `json:"attributes,omitempty"`
	PurchaseAt    *string         `json:"purchase_at,omitempty"`
	PurchasePrice *float64        `json:"purchase_price,omitempty"`
	PurchaseNote  *string         `json:"purchase_note,omitempty"`
}

type UpdateAssetRequest struct {
	CategoryID    string          `json:"category_id"`
	LocationID    *string         `json:"location_id,omitempty"`
	ConditionID   *string         `json:"condition_id,omitempty"`
	CollectionID  *string         `json:"collection_id,omitempty"`
	Name          string          `json:"name"`
	Description   *string         `json:"description,omitempty"`
	Quantity      int             `json:"quantity"`
	Attributes    json.RawMessage `json:"attributes,omitempty"`
	PurchaseAt    *string         `json:"purchase_at,omitempty"`
	PurchasePrice *float64        `json:"purchase_price,omitempty"`
	PurchaseNote  *string         `json:"purchase_note,omitempty"`
}

type AssetListResponse struct {
	Assets []domain.Asset `json:"assets"`
	Total  int            `json:"total"`
	Limit  int            `json:"limit"`
	Offset int            `json:"offset"`
}

func (h *Handler) ListAssets(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

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
		if id, err := uuid.Parse(catID); err == nil {
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
	assets, total, err := h.repos.Assets.List(r.Context(), h.orgID, filter, page)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list assets")
		return
	}

	if assets == nil {
		assets = []domain.Asset{}
	}

	writeJSON(w, http.StatusOK, AssetListResponse{
		Assets: assets,
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
	if asset == nil {
		writeError(w, http.StatusNotFound, "asset not found")
		return
	}

	writeJSON(w, http.StatusOK, asset)
}

func (h *Handler) CreateAsset(w http.ResponseWriter, r *http.Request) {
	var req CreateAssetRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" || req.CategoryID == "" {
		writeError(w, http.StatusBadRequest, "name and category_id are required")
		return
	}

	categoryID, err := uuid.Parse(req.CategoryID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid category_id")
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

	if asset.Quantity <= 0 {
		asset.Quantity = 1
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
	if req.CollectionID != nil {
		if id, err := parseUUIDString(*req.CollectionID); err == nil {
			asset.CollectionID = &id
		}
	}
	if req.PurchaseAt != nil && *req.PurchaseAt != "" {
		if t, err := time.Parse("2006-01-02", *req.PurchaseAt); err == nil {
			asset.PurchaseAt = &t
		}
	}
	asset.PurchasePrice = req.PurchasePrice
	asset.PurchaseNote = req.PurchaseNote

	if err := h.repos.Assets.Create(r.Context(), asset); err != nil {
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
	if asset == nil {
		writeError(w, http.StatusNotFound, "asset not found")
		return
	}

	categoryID, err := uuid.Parse(req.CategoryID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid category_id")
		return
	}

	asset.CategoryID = categoryID
	asset.Name = req.Name
	asset.Description = req.Description
	asset.Quantity = req.Quantity
	if asset.Quantity <= 0 {
		asset.Quantity = 1
	}
	asset.Attributes = req.Attributes

	if req.LocationID != nil {
		if id, err := parseUUIDString(*req.LocationID); err == nil {
			asset.LocationID = &id
		}
	} else {
		asset.LocationID = nil
	}
	if req.ConditionID != nil {
		if id, err := parseUUIDString(*req.ConditionID); err == nil {
			asset.ConditionID = &id
		}
	} else {
		asset.ConditionID = nil
	}
	if req.CollectionID != nil {
		if id, err := parseUUIDString(*req.CollectionID); err == nil {
			asset.CollectionID = &id
		}
	} else {
		asset.CollectionID = nil
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

	if err := h.repos.Assets.Update(r.Context(), asset); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update asset")
		return
	}

	writeJSON(w, http.StatusOK, asset)
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
	totalValue, err := h.repos.Assets.GetTotalValue(r.Context(), h.orgID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get asset stats")
		return
	}

	writeJSON(w, http.StatusOK, AssetStatsResponse{
		TotalValue: totalValue,
	})
}
