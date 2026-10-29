package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lmmendes/attic/internal/domain"
	"github.com/lmmendes/attic/internal/repository"
	"github.com/lmmendes/attic/internal/testutil"
)

func TestAssetFirstImageDefaultHTTPIntegration(t *testing.T) {
	ctx := context.WithValue(t.Context(), organizationFeaturesContextKey{}, allFeaturesEnabled())
	db, err := testutil.NewTestDB(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close(context.Background())
	org, err := testutil.NewFixtures(db.Pool).CreateOrganization(ctx, "Asset default image")
	if err != nil {
		t.Fatal(err)
	}
	repos := &Repositories{
		Organizations: repository.NewOrganizationRepository(db.Pool),
		Assets:        repository.NewAssetRepository(db.Pool),
		Attachments:   repository.NewAttachmentRepository(db.Pool),
	}
	h := New(nil, repos, newMockStorage(), org.ID)
	router := chi.NewRouter()
	router.Post("/api/assets", h.CreateAsset)
	router.Get("/api/assets/{id}", h.GetAsset)
	router.Post("/api/assets/{id}/attachments", h.UploadAttachment)
	router.Put("/api/assets/{id}/main-image/{attachmentId}", h.SetMainAttachment)
	router.Delete("/api/assets/{id}/main-image", h.ClearMainAttachment)
	router.Delete("/api/attachments/{attachmentId}", h.DeleteAttachment)
	request := func(method, path string, body io.Reader, contentType string, status int) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(method, path, body).WithContext(ctx)
		req.Header.Set("Content-Type", contentType)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != status {
			t.Fatalf("%s %s: got %d, want %d: %s", method, path, rec.Code, status, rec.Body)
		}
		return rec
	}
	rec := request(http.MethodPost, "/api/assets", strings.NewReader(`{"name":"New asset","quantity":1}`), "application/json", http.StatusCreated)
	var asset domain.Asset
	if err := json.Unmarshal(rec.Body.Bytes(), &asset); err != nil {
		t.Fatal(err)
	}
	assetPath := "/api/assets/" + asset.ID.String()
	assertMainImage := func(want *uuid.UUID) {
		t.Helper()
		rec := request(http.MethodGet, assetPath, nil, "", http.StatusOK)
		var detail AssetDetailResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &detail); err != nil {
			t.Fatal(err)
		}
		if want == nil {
			if detail.MainAttachmentID != nil || detail.MainAttachmentURL != "" {
				t.Fatalf("expected no default image: %s", rec.Body)
			}
		} else if detail.MainAttachmentID == nil || *detail.MainAttachmentID != *want || detail.MainAttachment == nil || detail.MainAttachment.ID != *want || detail.MainAttachmentURL == "" {
			t.Fatalf("expected default image %s with a display URL: %s", *want, rec.Body)
		}
	}
	upload := func(name, contentType string) domain.Attachment {
		t.Helper()
		var body bytes.Buffer
		writer := multipart.NewWriter(&body)
		part, err := writer.CreatePart(textproto.MIMEHeader{
			"Content-Disposition": {`form-data; name="file"; filename="` + name + `"`},
			"Content-Type":        {contentType},
		})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := part.Write([]byte("file contents")); err != nil {
			t.Fatal(err)
		}
		if err := writer.Close(); err != nil {
			t.Fatal(err)
		}
		rec := request(http.MethodPost, assetPath+"/attachments", &body, writer.FormDataContentType(), http.StatusCreated)
		var attachment domain.Attachment
		if err := json.Unmarshal(rec.Body.Bytes(), &attachment); err != nil {
			t.Fatal(err)
		}
		return attachment
	}

	assertMainImage(nil)
	upload("manual.pdf", "application/pdf")
	assertMainImage(nil)
	first := upload("first.png", "image/png")
	assertMainImage(&first.ID)
	second := upload("second.jpg", "image/jpeg")
	assertMainImage(&first.ID)

	request(http.MethodDelete, assetPath+"/main-image", nil, "", http.StatusNoContent)
	assertMainImage(nil)
	upload("third.webp", "image/webp")
	assertMainImage(nil)

	request(http.MethodPut, assetPath+"/main-image/"+second.ID.String(), nil, "", http.StatusNoContent)
	upload("fourth.gif", "image/gif")
	assertMainImage(&second.ID)
	request(http.MethodDelete, "/api/attachments/"+second.ID.String(), nil, "", http.StatusNoContent)
	assertMainImage(nil)
}
