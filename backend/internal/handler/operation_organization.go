package handler

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// checkOperationOrganization protects ID-based handlers and foreign-key assignments.
// The inventory is scoped to the configured workspace; never dispatch a key into
// an ID belonging to a different workspace, even for nested resources.
func (h *Handler) checkOperationOrganization(w http.ResponseWriter, r *http.Request, op Operation) bool {
	check := func(table, raw string) bool {
		id, err := uuid.Parse(raw)
		if err != nil {
			return true
		} // Existing handler reports invalid IDs.
		var query string
		switch table {
		case "attachments", "warranties", "asset_events":
			query = "SELECT EXISTS(SELECT 1 FROM " + table + " t JOIN assets a ON a.id=t.asset_id WHERE t.id=$1 AND a.organization_id=$2)"
		case "assets", "categories", "locations", "conditions", "collections", "attributes", "tags", "users", "saved_filters":
			query = "SELECT EXISTS(SELECT 1 FROM " + table + " WHERE id=$1 AND organization_id=$2)"
		default:
			return true
		}
		var exists bool
		if err := h.db.Pool.QueryRow(r.Context(), query, id, h.orgID).Scan(&exists); err != nil {
			writeError(w, 500, "failed to check resource organization")
			return false
		}
		if !exists {
			writeError(w, 404, "resource not found")
			return false
		}
		return true
	}
	root := strings.ReplaceAll(strings.Split(strings.TrimPrefix(op.Path, "/api/"), "/")[0], "-", "_")
	for _, entry := range []struct{ param, table string }{{"id", root}, {"attachmentId", "attachments"}, {"eventId", "asset_events"}} {
		if raw := chi.URLParam(r, entry.param); raw != "" && !check(entry.table, raw) {
			return false
		}
	}
	// Validate references in write bodies before the handler can persist them.
	if r.Body == nil || op.ToolName == "upload_attachment" || op.Action != "write" {
		return true
	}
	data, err := io.ReadAll(io.LimitReader(r.Body, 4<<20+1))
	if err != nil || len(data) > 4<<20 {
		writeError(w, 413, "request body too large")
		return false
	}
	r.Body = io.NopCloser(bytes.NewReader(data))
	var value any
	if json.Unmarshal(data, &value) != nil {
		return true
	}
	tables := map[string]string{"category_id": "categories", "location_id": "locations", "condition_id": "conditions", "collection_id": "collections", "collection_ids": "collections", "tag_ids": "tags", "attribute_id": "attributes", "attribute_ids": "attributes", "main_attachment_id": "attachments", "parent_id": root}
	var walk func(any) bool
	walk = func(value any) bool {
		switch v := value.(type) {
		case map[string]any:
			for field, child := range v {
				field = strings.ToLower(field)
				// Asset attribute values are opaque user data, not resource references.
				if field == "attributes" {
					if _, ok := child.(map[string]any); ok {
						continue
					}
				}
				if table := tables[field]; table != "" {
					switch ids := child.(type) {
					case string:
						if !check(table, ids) {
							return false
						}
					case []any:
						for _, id := range ids {
							if raw, ok := id.(string); ok && !check(table, raw) {
								return false
							}
						}
					}
				}
				if !walk(child) {
					return false
				}
			}
		case []any:
			for _, child := range v {
				if !walk(child) {
					return false
				}
			}
		}
		return true
	}
	return walk(value)
}
