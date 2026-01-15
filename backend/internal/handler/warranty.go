package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/mendelui/attic/internal/domain"
)

type CreateWarrantyRequest struct {
	Provider  *string `json:"provider,omitempty"`
	StartDate *string `json:"start_date,omitempty"`
	EndDate   *string `json:"end_date,omitempty"`
	Notes     *string `json:"notes,omitempty"`
}

type UpdateWarrantyRequest struct {
	Provider  *string `json:"provider,omitempty"`
	StartDate *string `json:"start_date,omitempty"`
	EndDate   *string `json:"end_date,omitempty"`
	Notes     *string `json:"notes,omitempty"`
}

func (h *Handler) GetWarranty(w http.ResponseWriter, r *http.Request) {
	assetID, err := parseUUID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid asset ID")
		return
	}

	warranty, err := h.repos.Warranties.GetByAssetID(r.Context(), assetID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get warranty")
		return
	}
	if warranty == nil {
		writeError(w, http.StatusNotFound, "warranty not found")
		return
	}

	writeJSON(w, http.StatusOK, warranty)
}

func (h *Handler) ListWarranties(w http.ResponseWriter, r *http.Request) {
	warranties, err := h.repos.Warranties.List(r.Context(), h.orgID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list warranties")
		return
	}

	if warranties == nil {
		warranties = []domain.WarrantyWithAsset{}
	}

	writeJSON(w, http.StatusOK, warranties)
}

func (h *Handler) ListExpiringWarranties(w http.ResponseWriter, r *http.Request) {
	days, _ := strconv.Atoi(r.URL.Query().Get("days"))
	if days <= 0 {
		days = 30 // Default to 30 days
	}

	warranties, err := h.repos.Warranties.ListExpiring(r.Context(), h.orgID, days)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list warranties")
		return
	}

	if warranties == nil {
		warranties = []domain.Warranty{}
	}

	writeJSON(w, http.StatusOK, warranties)
}

func (h *Handler) CreateWarranty(w http.ResponseWriter, r *http.Request) {
	assetID, err := parseUUID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid asset ID")
		return
	}

	// Check if asset exists
	asset, err := h.repos.Assets.GetByID(r.Context(), assetID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to check asset")
		return
	}
	if asset == nil {
		writeError(w, http.StatusNotFound, "asset not found")
		return
	}

	// Check if warranty already exists
	existing, err := h.repos.Warranties.GetByAssetID(r.Context(), assetID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to check existing warranty")
		return
	}
	if existing != nil {
		writeError(w, http.StatusConflict, "warranty already exists for this asset")
		return
	}

	var req CreateWarrantyRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	warranty := &domain.Warranty{
		AssetID:  assetID,
		Provider: req.Provider,
		Notes:    req.Notes,
	}

	if req.StartDate != nil {
		if t, err := time.Parse("2006-01-02", *req.StartDate); err == nil {
			warranty.StartDate = &t
		}
	}
	if req.EndDate != nil {
		if t, err := time.Parse("2006-01-02", *req.EndDate); err == nil {
			warranty.EndDate = &t
		}
	}

	if err := h.repos.Warranties.Create(r.Context(), warranty); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create warranty")
		return
	}

	writeJSON(w, http.StatusCreated, warranty)
}

func (h *Handler) UpdateWarranty(w http.ResponseWriter, r *http.Request) {
	assetID, err := parseUUID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid asset ID")
		return
	}

	warranty, err := h.repos.Warranties.GetByAssetID(r.Context(), assetID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get warranty")
		return
	}
	if warranty == nil {
		writeError(w, http.StatusNotFound, "warranty not found")
		return
	}

	var req UpdateWarrantyRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	warranty.Provider = req.Provider
	warranty.Notes = req.Notes

	if req.StartDate != nil {
		if t, err := time.Parse("2006-01-02", *req.StartDate); err == nil {
			warranty.StartDate = &t
		}
	} else {
		warranty.StartDate = nil
	}
	if req.EndDate != nil {
		if t, err := time.Parse("2006-01-02", *req.EndDate); err == nil {
			warranty.EndDate = &t
		}
	} else {
		warranty.EndDate = nil
	}

	if err := h.repos.Warranties.Update(r.Context(), warranty); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update warranty")
		return
	}

	writeJSON(w, http.StatusOK, warranty)
}

func (h *Handler) DeleteWarranty(w http.ResponseWriter, r *http.Request) {
	assetID, err := parseUUID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid asset ID")
		return
	}

	if err := h.repos.Warranties.Delete(r.Context(), assetID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete warranty")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
