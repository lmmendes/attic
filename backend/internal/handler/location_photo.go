package handler

import (
	"bytes"
	"context"
	"errors"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"log/slog"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/lmmendes/attic/internal/domain"
)

const maxLocationPhotoSize = 10 << 20
const maxLocationPhotoPixels = 40_000_000

func (h *Handler) populateLocationPhoto(r *http.Request, loc *domain.Location) error {
	var photoErr error
	if loc.PhotoKey != nil && h.storage != nil {
		url, err := h.storage.GetPresignedURL(r.Context(), *loc.PhotoKey, 15*time.Minute)
		if err != nil {
			photoErr = err
		} else {
			loc.PhotoURL = &url
		}
	}
	for i := range loc.Children {
		if err := h.populateLocationPhoto(r, &loc.Children[i]); err != nil {
			photoErr = err
		}
	}
	return photoErr
}

func (h *Handler) photoLocation(w http.ResponseWriter, r *http.Request) *domain.Location {
	id, err := parseUUID(r, "id")
	if err != nil {
		writeError(w, 400, "invalid location ID")
		return nil
	}
	loc, err := h.repos.Locations.GetByID(r.Context(), id)
	if err != nil {
		writeError(w, 500, "failed to read location")
		return nil
	}
	if loc == nil || loc.OrganizationID != h.orgID {
		writeError(w, 404, "location not found")
		return nil
	}
	return loc
}

// UploadLocationPhoto appends a validated image without replacing existing photos.
func (h *Handler) UploadLocationPhoto(w http.ResponseWriter, r *http.Request) {
	loc := h.photoLocation(w, r)
	if loc == nil {
		return
	}
	if h.storage == nil {
		writeError(w, 503, "storage not configured")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxLocationPhotoSize+(1<<20))
	if err := r.ParseMultipartForm(1 << 20); err != nil {
		writeError(w, 400, "photo exceeds 10 MB or invalid form")
		return
	}
	defer r.MultipartForm.RemoveAll()
	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, 400, "a photo file is required")
		return
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, maxLocationPhotoSize+1))
	if err != nil {
		writeError(w, 400, "failed to read photo")
		return
	}
	if len(data) > maxLocationPhotoSize {
		writeError(w, 400, "photo exceeds 10 MB")
		return
	}
	config, format, err := image.DecodeConfig(bytes.NewReader(data))
	extensions := map[string]string{"jpeg": "jpg", "png": "png", "gif": "gif"}
	extension, ok := extensions[format]
	if err != nil || !ok || config.Width <= 0 || config.Height <= 0 || config.Width > maxLocationPhotoPixels/config.Height {
		writeError(w, 400, "choose a JPEG, PNG or GIF photo up to 40 megapixels")
		return
	}
	contentType := http.DetectContentType(data)
	// Use a controlled filename instead of trusting a multipart path or extension.
	key, err := h.storage.Upload(r.Context(), "locations", "location-photo."+extension, contentType, bytes.NewReader(data))
	if err != nil {
		writeError(w, 500, "failed to upload photo")
		return
	}
	photo := domain.LocationPhoto{LocationID: loc.ID, FileKey: key, FileName: filepath.Base(strings.ReplaceAll(header.Filename, `\`, "/"))}
	photo.URL, err = h.storage.GetPresignedURL(r.Context(), key, 15*time.Minute)
	if err != nil {
		h.cleanupLocationPhotos(r.Context(), []string{key})
		writeError(w, 500, "failed to read uploaded photo")
		return
	}
	if err = h.repos.Locations.AddPhoto(r.Context(), h.orgID, &photo); err != nil {
		h.cleanupLocationPhotos(r.Context(), []string{key})
		writeLocationPhotoError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, photo)
}

// ListLocationPhotos returns only photos belonging to the active workspace location.
func (h *Handler) ListLocationPhotos(w http.ResponseWriter, r *http.Request) {
	loc := h.photoLocation(w, r)
	if loc == nil {
		return
	}
	photos, err := h.repos.Locations.ListPhotos(r.Context(), h.orgID, loc.ID)
	if err != nil {
		writeError(w, 500, "failed to list photos")
		return
	}
	for i := range photos {
		if h.storage != nil {
			photos[i].URL, err = h.storage.GetPresignedURL(r.Context(), photos[i].FileKey, 15*time.Minute)
			if err != nil {
				slog.Warn("failed to read location photo", "error", err)
			}
		}
	}
	writeJSON(w, 200, photos)
}

// DeleteLocationPhoto removes one photo without changing other gallery images.
func (h *Handler) DeleteLocationPhoto(w http.ResponseWriter, r *http.Request) {
	loc := h.photoLocation(w, r)
	if loc == nil {
		return
	}
	photoID, err := parseUUID(r, "photoId")
	if err != nil {
		writeError(w, 400, "invalid photo ID")
		return
	}
	key, err := h.repos.Locations.DeletePhoto(r.Context(), h.orgID, loc.ID, photoID)
	if err != nil {
		writeLocationPhotoError(w, err)
		return
	}
	h.cleanupLocationPhotos(r.Context(), []string{key})
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) SetLocationCoverPhoto(w http.ResponseWriter, r *http.Request) {
	h.changeLocationCoverPhoto(w, r, true)
}

func (h *Handler) UnsetLocationCoverPhoto(w http.ResponseWriter, r *http.Request) {
	h.changeLocationCoverPhoto(w, r, false)
}

func (h *Handler) changeLocationCoverPhoto(w http.ResponseWriter, r *http.Request, selected bool) {
	loc := h.photoLocation(w, r)
	if loc == nil {
		return
	}
	photoID, err := parseUUID(r, "photoId")
	if err != nil {
		writeError(w, 400, "invalid photo ID")
		return
	}
	if err = h.repos.Locations.SetCoverPhoto(r.Context(), h.orgID, loc.ID, photoID, selected); err != nil {
		writeLocationPhotoError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func writeLocationPhotoError(w http.ResponseWriter, err error) {
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, 404, "location or photo not found")
	} else {
		writeError(w, 500, "failed to save location photos")
	}
}

func (h *Handler) cleanupLocationPhotos(ctx context.Context, keys []string) {
	if len(keys) == 0 || h.storage == nil {
		return
	}
	cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
	defer cancel()
	for _, key := range keys {
		if err := h.storage.Delete(cleanupCtx, key); err != nil {
			slog.Warn("failed to clean up location photo", "error", err)
		}
	}
}
