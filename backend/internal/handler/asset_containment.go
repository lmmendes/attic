package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/lmmendes/attic/internal/domain"
	"github.com/lmmendes/attic/internal/repository"
)

func (h *Handler) SetAssetParent(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid asset ID")
		return
	}
	var req struct {
		ParentID json.RawMessage `json:"parent_id"`
	}
	if err = decodeJSON(r, &req); err != nil || len(req.ParentID) == 0 {
		writeError(w, http.StatusBadRequest, "parent_id is required (use null to detach)")
		return
	}
	a := &domain.Asset{}
	if err = assignContainmentInput(a, req.ParentID, nil, nil); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	asset, err := h.repos.Assets.SetParent(r.Context(), h.orgID, id, a.ParentID)
	if err != nil {
		if !writeContainmentError(w, err) {
			writeError(w, http.StatusInternalServerError, "failed to change parent")
		}
		return
	}
	if err = h.sanitizeAsset(r, asset); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to apply organization features")
		return
	}
	writeJSON(w, http.StatusOK, asset)
}

func assignContainmentInput(a *domain.Asset, raw json.RawMessage, add, remove []string) error {
	a.ParentProvided = len(raw) > 0
	if a.ParentProvided {
		var value *string
		if err := json.Unmarshal(raw, &value); err != nil {
			return errors.New("parent_id must be a UUID or null")
		}
		a.ParentID = nil
		if value != nil {
			id, err := uuid.Parse(*value)
			if err != nil || id == uuid.Nil {
				return errors.New("invalid parent_id")
			}
			a.ParentID = &id
		}
	}
	var err error
	if a.AddChildIDs, err = parseContentIDs(add); err != nil {
		return err
	}
	if a.RemoveChildIDs, err = parseContentIDs(remove); err != nil {
		return err
	}
	return nil
}

func parseContentIDs(values []string) ([]uuid.UUID, error) {
	ids := []uuid.UUID{}
	seen := map[uuid.UUID]bool{}
	for _, value := range values {
		id, err := uuid.Parse(value)
		if err != nil || id == uuid.Nil {
			return nil, errors.New("invalid content asset ID")
		}
		if !seen[id] {
			ids = append(ids, id)
			seen[id] = true
		}
	}
	return ids, nil
}

func writeContainmentError(w http.ResponseWriter, err error) bool {
	var containmentErr *repository.ContainmentError
	if !errors.As(err, &containmentErr) {
		return false
	}
	writeError(w, containmentErr.Status, containmentErr.Message)
	return true
}
