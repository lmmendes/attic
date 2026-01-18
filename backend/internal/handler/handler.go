package handler

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/mendelui/attic/internal/database"
	"github.com/mendelui/attic/internal/repository"
)

// FileStorage defines the interface for file storage backends
type FileStorage interface {
	Upload(ctx context.Context, filename string, contentType string, body io.Reader) (string, error)
	GetPresignedURL(ctx context.Context, key string, expiry time.Duration) (string, error)
	Delete(ctx context.Context, key string) error
}

// Repositories holds all repository implementations
type Repositories struct {
	Organizations *repository.OrganizationRepository
	Users         *repository.UserRepository
	Categories    *repository.CategoryRepository
	Locations     *repository.LocationRepository
	Conditions    *repository.ConditionRepository
	Assets        *repository.AssetRepository
	Warranties    *repository.WarrantyRepository
	Attachments   *repository.AttachmentRepository
	Attributes    *repository.AttributeRepository
}

// Handler holds dependencies for HTTP handlers
type Handler struct {
	db      *database.DB
	repos   *Repositories
	storage FileStorage
	orgID   uuid.UUID // Default organization ID
}

// New creates a new Handler
func New(db *database.DB, repos *Repositories, storage FileStorage) *Handler {
	return &Handler{
		db:      db,
		repos:   repos,
		storage: storage,
		orgID:   uuid.MustParse("00000000-0000-0000-0000-000000000001"), // Default org from seed
	}
}

// Health returns server health status
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// Ready returns database readiness status
func (h *Handler) Ready(w http.ResponseWriter, r *http.Request) {
	if err := h.db.Health(r.Context()); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "not ready", "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

// Helper functions

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		slog.Error("failed to encode response", "error", err)
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func parseUUID(r *http.Request, param string) (uuid.UUID, error) {
	return uuid.Parse(chi.URLParam(r, param))
}

func decodeJSON(r *http.Request, v any) error {
	return json.NewDecoder(r.Body).Decode(v)
}
