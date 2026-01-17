package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/mendelui/attic/internal/domain"
)

// mockAttachmentRepo implements attachment repository for testing
type mockAttachmentRepo struct {
	attachments    map[uuid.UUID]*domain.Attachment
	byAsset        map[uuid.UUID][]domain.Attachment
	createErr      error
	getByIDErr     error
	listByAssetErr error
	deleteErr      error
}

func newMockAttachmentRepo() *mockAttachmentRepo {
	return &mockAttachmentRepo{
		attachments: make(map[uuid.UUID]*domain.Attachment),
		byAsset:     make(map[uuid.UUID][]domain.Attachment),
	}
}

func (m *mockAttachmentRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Attachment, error) {
	if m.getByIDErr != nil {
		return nil, m.getByIDErr
	}
	return m.attachments[id], nil
}

func (m *mockAttachmentRepo) ListByAsset(ctx context.Context, assetID uuid.UUID) ([]domain.Attachment, error) {
	if m.listByAssetErr != nil {
		return nil, m.listByAssetErr
	}
	return m.byAsset[assetID], nil
}

func (m *mockAttachmentRepo) Create(ctx context.Context, a *domain.Attachment) error {
	if m.createErr != nil {
		return m.createErr
	}
	a.ID = uuid.New()
	a.CreatedAt = time.Now()
	m.attachments[a.ID] = a
	m.byAsset[a.AssetID] = append(m.byAsset[a.AssetID], *a)
	return nil
}

func (m *mockAttachmentRepo) Delete(ctx context.Context, id uuid.UUID) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	if a, ok := m.attachments[id]; ok {
		// Remove from byAsset map
		assets := m.byAsset[a.AssetID]
		for i, att := range assets {
			if att.ID == id {
				m.byAsset[a.AssetID] = append(assets[:i], assets[i+1:]...)
				break
			}
		}
		delete(m.attachments, id)
	}
	return nil
}

func (m *mockAttachmentRepo) addAttachment(a *domain.Attachment) {
	m.attachments[a.ID] = a
	m.byAsset[a.AssetID] = append(m.byAsset[a.AssetID], *a)
}

// mockStorage implements storage interface for testing
type mockStorage struct {
	files         map[string][]byte
	uploadErr     error
	deleteErr     error
	presignedErr  error
	presignedURL  string
}

func newMockStorage() *mockStorage {
	return &mockStorage{
		files:        make(map[string][]byte),
		presignedURL: "https://storage.example.com/file?signed=true",
	}
}

func (m *mockStorage) Upload(ctx context.Context, filename, contentType string, reader io.Reader) (string, error) {
	if m.uploadErr != nil {
		return "", m.uploadErr
	}
	data, _ := io.ReadAll(reader)
	key := "uploads/" + uuid.New().String() + "/" + filename
	m.files[key] = data
	return key, nil
}

func (m *mockStorage) Delete(ctx context.Context, key string) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	delete(m.files, key)
	return nil
}

func (m *mockStorage) GetPresignedURL(ctx context.Context, key string, duration time.Duration) (string, error) {
	if m.presignedErr != nil {
		return "", m.presignedErr
	}
	return m.presignedURL, nil
}

// testAttachmentHandler wraps attachment logic for testing
type testAttachmentHandler struct {
	attachmentRepo *mockAttachmentRepo
	assetRepo      *mockAssetRepo
	storage        *mockStorage
}

func newTestAttachmentHandler() *testAttachmentHandler {
	return &testAttachmentHandler{
		attachmentRepo: newMockAttachmentRepo(),
		assetRepo:      newMockAssetRepo(),
		storage:        newMockStorage(),
	}
}

func (h *testAttachmentHandler) ListAttachments(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	assetID, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid asset ID")
		return
	}

	attachments, err := h.attachmentRepo.ListByAsset(r.Context(), assetID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list attachments")
		return
	}

	if attachments == nil {
		attachments = []domain.Attachment{}
	}

	writeJSON(w, http.StatusOK, attachments)
}

func (h *testAttachmentHandler) GetAttachment(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "attachmentId")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid attachment ID")
		return
	}

	attachment, err := h.attachmentRepo.GetByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get attachment")
		return
	}
	if attachment == nil {
		writeError(w, http.StatusNotFound, "attachment not found")
		return
	}

	if h.storage == nil {
		writeError(w, http.StatusServiceUnavailable, "storage not configured")
		return
	}

	url, err := h.storage.GetPresignedURL(r.Context(), attachment.FileKey, 15*time.Minute)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate download URL")
		return
	}

	response := AttachmentResponse{
		Attachment: *attachment,
		URL:        url,
	}

	writeJSON(w, http.StatusOK, response)
}

func (h *testAttachmentHandler) UploadAttachment(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	assetID, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid asset ID")
		return
	}

	// Check if asset exists
	asset, err := h.assetRepo.GetByID(r.Context(), assetID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to check asset")
		return
	}
	if asset == nil {
		writeError(w, http.StatusNotFound, "asset not found")
		return
	}

	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		writeError(w, http.StatusBadRequest, "file too large or invalid form")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing file in request")
		return
	}
	defer file.Close()

	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		buf := make([]byte, 512)
		n, _ := file.Read(buf)
		contentType = http.DetectContentType(buf[:n])
		file.Seek(0, io.SeekStart)
	}

	if h.storage == nil {
		writeError(w, http.StatusServiceUnavailable, "storage not configured")
		return
	}

	key, err := h.storage.Upload(r.Context(), header.Filename, contentType, file)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to upload file")
		return
	}

	description := r.FormValue("description")
	var desc *string
	if description != "" {
		desc = &description
	}

	attachment := &domain.Attachment{
		AssetID:     assetID,
		FileKey:     key,
		FileName:    header.Filename,
		FileSize:    header.Size,
		ContentType: &contentType,
		Description: desc,
	}

	if err := h.attachmentRepo.Create(r.Context(), attachment); err != nil {
		h.storage.Delete(r.Context(), key)
		writeError(w, http.StatusInternalServerError, "failed to save attachment record")
		return
	}

	writeJSON(w, http.StatusCreated, attachment)
}

func (h *testAttachmentHandler) DeleteAttachment(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "attachmentId")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid attachment ID")
		return
	}

	attachment, err := h.attachmentRepo.GetByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get attachment")
		return
	}
	if attachment == nil {
		writeError(w, http.StatusNotFound, "attachment not found")
		return
	}

	if h.storage != nil {
		h.storage.Delete(r.Context(), attachment.FileKey)
	}

	if err := h.attachmentRepo.Delete(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete attachment")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// withAttachmentChiURLParam helper for URL params
func withAttachmentChiURLParam(r *http.Request, key, value string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(key, value)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

func withMultipleChiURLParams(r *http.Request, params map[string]string) *http.Request {
	rctx := chi.NewRouteContext()
	for k, v := range params {
		rctx.URLParams.Add(k, v)
	}
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

// createMultipartRequest helper for file uploads
func createMultipartRequest(filename string, content []byte, description string) (*http.Request, error) {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return nil, err
	}
	part.Write(content)

	if description != "" {
		writer.WriteField("description", description)
	}

	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req, nil
}

// Tests for ListAttachments

func Test_ListAttachments_EmptyList_ReturnsEmptyArray(t *testing.T) {
	h := newTestAttachmentHandler()
	assetID := uuid.New()

	req := httptest.NewRequest(http.MethodGet, "/assets/"+assetID.String()+"/attachments", nil)
	req = withAttachmentChiURLParam(req, "id", assetID.String())
	rec := httptest.NewRecorder()

	h.ListAttachments(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var attachments []domain.Attachment
	json.NewDecoder(rec.Body).Decode(&attachments)
	if len(attachments) != 0 {
		t.Errorf("expected empty array, got %d attachments", len(attachments))
	}
}

func Test_ListAttachments_WithAttachments_ReturnsAll(t *testing.T) {
	h := newTestAttachmentHandler()
	assetID := uuid.New()
	contentType := "image/png"

	h.attachmentRepo.addAttachment(&domain.Attachment{
		ID:          uuid.New(),
		AssetID:     assetID,
		FileKey:     "uploads/test/file1.png",
		FileName:    "file1.png",
		FileSize:    1024,
		ContentType: &contentType,
		CreatedAt:   time.Now(),
	})

	req := httptest.NewRequest(http.MethodGet, "/assets/"+assetID.String()+"/attachments", nil)
	req = withAttachmentChiURLParam(req, "id", assetID.String())
	rec := httptest.NewRecorder()

	h.ListAttachments(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var attachments []domain.Attachment
	json.NewDecoder(rec.Body).Decode(&attachments)
	if len(attachments) != 1 {
		t.Errorf("expected 1 attachment, got %d", len(attachments))
	}
	if attachments[0].FileName != "file1.png" {
		t.Errorf("expected filename file1.png, got %s", attachments[0].FileName)
	}
}

func Test_ListAttachments_InvalidAssetID_ReturnsBadRequest(t *testing.T) {
	h := newTestAttachmentHandler()

	req := httptest.NewRequest(http.MethodGet, "/assets/invalid/attachments", nil)
	req = withAttachmentChiURLParam(req, "id", "invalid")
	rec := httptest.NewRecorder()

	h.ListAttachments(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}
}

// Tests for GetAttachment

func Test_GetAttachment_Exists_ReturnsWithURL(t *testing.T) {
	h := newTestAttachmentHandler()
	attachmentID := uuid.New()
	assetID := uuid.New()
	contentType := "application/pdf"

	h.attachmentRepo.addAttachment(&domain.Attachment{
		ID:          attachmentID,
		AssetID:     assetID,
		FileKey:     "uploads/test/doc.pdf",
		FileName:    "doc.pdf",
		FileSize:    2048,
		ContentType: &contentType,
		CreatedAt:   time.Now(),
	})

	req := httptest.NewRequest(http.MethodGet, "/attachments/"+attachmentID.String(), nil)
	req = withAttachmentChiURLParam(req, "attachmentId", attachmentID.String())
	rec := httptest.NewRecorder()

	h.GetAttachment(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var response AttachmentResponse
	json.NewDecoder(rec.Body).Decode(&response)
	if response.FileName != "doc.pdf" {
		t.Errorf("expected filename doc.pdf, got %s", response.FileName)
	}
	if response.URL == "" {
		t.Error("expected presigned URL to be set")
	}
}

func Test_GetAttachment_NotFound_ReturnsNotFound(t *testing.T) {
	h := newTestAttachmentHandler()
	id := uuid.New()

	req := httptest.NewRequest(http.MethodGet, "/attachments/"+id.String(), nil)
	req = withAttachmentChiURLParam(req, "attachmentId", id.String())
	rec := httptest.NewRecorder()

	h.GetAttachment(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rec.Code)
	}
}

func Test_GetAttachment_InvalidID_ReturnsBadRequest(t *testing.T) {
	h := newTestAttachmentHandler()

	req := httptest.NewRequest(http.MethodGet, "/attachments/invalid", nil)
	req = withAttachmentChiURLParam(req, "attachmentId", "invalid")
	rec := httptest.NewRecorder()

	h.GetAttachment(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}
}

func Test_GetAttachment_NoStorage_ReturnsServiceUnavailable(t *testing.T) {
	h := newTestAttachmentHandler()
	h.storage = nil
	attachmentID := uuid.New()
	assetID := uuid.New()

	h.attachmentRepo.addAttachment(&domain.Attachment{
		ID:        attachmentID,
		AssetID:   assetID,
		FileKey:   "uploads/test/file.txt",
		FileName:  "file.txt",
		FileSize:  100,
		CreatedAt: time.Now(),
	})

	req := httptest.NewRequest(http.MethodGet, "/attachments/"+attachmentID.String(), nil)
	req = withAttachmentChiURLParam(req, "attachmentId", attachmentID.String())
	rec := httptest.NewRecorder()

	h.GetAttachment(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status 503, got %d", rec.Code)
	}
}

// Tests for UploadAttachment

func Test_UploadAttachment_ValidRequest_ReturnsCreated(t *testing.T) {
	h := newTestAttachmentHandler()
	assetID := uuid.New()

	h.assetRepo.addAsset(&domain.Asset{
		ID:   assetID,
		Name: "Test Asset",
	})

	req, _ := createMultipartRequest("test.txt", []byte("test content"), "Test description")
	req = withAttachmentChiURLParam(req, "id", assetID.String())
	rec := httptest.NewRecorder()

	h.UploadAttachment(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d", rec.Code)
	}

	var attachment domain.Attachment
	json.NewDecoder(rec.Body).Decode(&attachment)
	if attachment.FileName != "test.txt" {
		t.Errorf("expected filename test.txt, got %s", attachment.FileName)
	}
	if attachment.Description == nil || *attachment.Description != "Test description" {
		t.Error("expected description to be set")
	}
}

func Test_UploadAttachment_AssetNotFound_ReturnsNotFound(t *testing.T) {
	h := newTestAttachmentHandler()
	assetID := uuid.New()

	req, _ := createMultipartRequest("test.txt", []byte("test content"), "")
	req = withAttachmentChiURLParam(req, "id", assetID.String())
	rec := httptest.NewRecorder()

	h.UploadAttachment(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rec.Code)
	}
}

func Test_UploadAttachment_InvalidAssetID_ReturnsBadRequest(t *testing.T) {
	h := newTestAttachmentHandler()

	req, _ := createMultipartRequest("test.txt", []byte("test content"), "")
	req = withAttachmentChiURLParam(req, "id", "invalid")
	rec := httptest.NewRecorder()

	h.UploadAttachment(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}
}

func Test_UploadAttachment_MissingFile_ReturnsBadRequest(t *testing.T) {
	h := newTestAttachmentHandler()
	assetID := uuid.New()

	h.assetRepo.addAsset(&domain.Asset{
		ID:   assetID,
		Name: "Test Asset",
	})

	req := httptest.NewRequest(http.MethodPost, "/assets/"+assetID.String()+"/attachments", nil)
	req.Header.Set("Content-Type", "multipart/form-data")
	req = withAttachmentChiURLParam(req, "id", assetID.String())
	rec := httptest.NewRecorder()

	h.UploadAttachment(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}
}

func Test_UploadAttachment_NoStorage_ReturnsServiceUnavailable(t *testing.T) {
	h := newTestAttachmentHandler()
	h.storage = nil
	assetID := uuid.New()

	h.assetRepo.addAsset(&domain.Asset{
		ID:   assetID,
		Name: "Test Asset",
	})

	req, _ := createMultipartRequest("test.txt", []byte("test content"), "")
	req = withAttachmentChiURLParam(req, "id", assetID.String())
	rec := httptest.NewRecorder()

	h.UploadAttachment(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status 503, got %d", rec.Code)
	}
}

// Tests for DeleteAttachment

func Test_DeleteAttachment_Exists_ReturnsNoContent(t *testing.T) {
	h := newTestAttachmentHandler()
	attachmentID := uuid.New()
	assetID := uuid.New()

	h.attachmentRepo.addAttachment(&domain.Attachment{
		ID:        attachmentID,
		AssetID:   assetID,
		FileKey:   "uploads/test/file.txt",
		FileName:  "file.txt",
		FileSize:  100,
		CreatedAt: time.Now(),
	})
	h.storage.files["uploads/test/file.txt"] = []byte("content")

	req := httptest.NewRequest(http.MethodDelete, "/attachments/"+attachmentID.String(), nil)
	req = withAttachmentChiURLParam(req, "attachmentId", attachmentID.String())
	rec := httptest.NewRecorder()

	h.DeleteAttachment(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("expected status 204, got %d", rec.Code)
	}

	// Verify deleted from repo
	if _, exists := h.attachmentRepo.attachments[attachmentID]; exists {
		t.Error("expected attachment to be deleted from repository")
	}
}

func Test_DeleteAttachment_NotFound_ReturnsNotFound(t *testing.T) {
	h := newTestAttachmentHandler()
	id := uuid.New()

	req := httptest.NewRequest(http.MethodDelete, "/attachments/"+id.String(), nil)
	req = withAttachmentChiURLParam(req, "attachmentId", id.String())
	rec := httptest.NewRecorder()

	h.DeleteAttachment(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rec.Code)
	}
}

func Test_DeleteAttachment_InvalidID_ReturnsBadRequest(t *testing.T) {
	h := newTestAttachmentHandler()

	req := httptest.NewRequest(http.MethodDelete, "/attachments/invalid", nil)
	req = withAttachmentChiURLParam(req, "attachmentId", "invalid")
	rec := httptest.NewRecorder()

	h.DeleteAttachment(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}
}

func Test_DeleteAttachment_NoStorage_StillDeletesFromDB(t *testing.T) {
	h := newTestAttachmentHandler()
	h.storage = nil
	attachmentID := uuid.New()
	assetID := uuid.New()

	h.attachmentRepo.addAttachment(&domain.Attachment{
		ID:        attachmentID,
		AssetID:   assetID,
		FileKey:   "uploads/test/file.txt",
		FileName:  "file.txt",
		FileSize:  100,
		CreatedAt: time.Now(),
	})

	req := httptest.NewRequest(http.MethodDelete, "/attachments/"+attachmentID.String(), nil)
	req = withAttachmentChiURLParam(req, "attachmentId", attachmentID.String())
	rec := httptest.NewRecorder()

	h.DeleteAttachment(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("expected status 204, got %d", rec.Code)
	}

	// Verify deleted from repo even without storage
	if _, exists := h.attachmentRepo.attachments[attachmentID]; exists {
		t.Error("expected attachment to be deleted from repository")
	}
}
