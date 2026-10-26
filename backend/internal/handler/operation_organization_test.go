package handler

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/lmmendes/attic/internal/database"
	"github.com/lmmendes/attic/internal/domain"
	"github.com/lmmendes/attic/internal/repository"
	"github.com/lmmendes/attic/internal/testutil"
)

func TestOperationOrganizationRejectsInvalidWriteBodies(t *testing.T) {
	for _, tt := range []struct {
		name    string
		body    string
		allowed bool
	}{
		{name: "empty body", body: "", allowed: true},
		{name: "whitespace body", body: " \t\r\n", allowed: true},
		{name: "valid object", body: `{"name":"Asset"}`, allowed: true},
		{name: "trailing whitespace", body: "{\"name\":\"Asset\"}\n\t ", allowed: true},
		{name: "malformed object", body: `{"name":`, allowed: false},
		{name: "trailing garbage", body: `{"name":"Asset","location_id":"11111111-1111-4111-8111-111111111111"} x`, allowed: false},
		{name: "second JSON value", body: `{"name":"Asset","location_id":"11111111-1111-4111-8111-111111111111"} {}`, allowed: false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			// No database is needed: invalid bodies must fail before reference lookup.
			h := &Handler{}
			request := httptest.NewRequest(http.MethodPost, "/api/assets", strings.NewReader(tt.body))
			response := httptest.NewRecorder()
			allowed := h.checkOperationOrganization(response, request, Operation{Path: "/api/assets", Action: "write"})
			if allowed != tt.allowed {
				t.Fatalf("allowed=%v, want %v (status=%d)", allowed, tt.allowed, response.Code)
			}
			if !allowed && response.Code != http.StatusBadRequest {
				t.Fatalf("status=%d, want 400", response.Code)
			}
			restored, err := io.ReadAll(request.Body)
			if err != nil || string(restored) != tt.body {
				t.Fatalf("handler request body changed: %q, %v", restored, err)
			}
		})
	}
}

func TestOperationOrganizationMatchesJSONFieldNames(t *testing.T) {
	ctx := t.Context()
	db, err := testutil.NewTestDB(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close(ctx)
	fixtures := testutil.NewFixtures(db.Pool)
	org, err := fixtures.CreateOrganization(ctx, "Current workspace")
	if err != nil {
		t.Fatal(err)
	}
	other, err := fixtures.CreateOrganization(ctx, "Other workspace")
	if err != nil {
		t.Fatal(err)
	}
	collection := &domain.Collection{OrganizationID: other.ID, Name: "Other collection"}
	if err := repository.NewCollectionRepository(db.Pool).Create(ctx, collection); err != nil {
		t.Fatal(err)
	}
	tag, err := fixtures.CreateTag(ctx, other.ID, "Other tag")
	if err != nil {
		t.Fatal(err)
	}
	attribute, err := fixtures.CreateAttribute(ctx, other.ID, "Other attribute", "other", domain.AttributeTypeString)
	if err != nil {
		t.Fatal(err)
	}
	h := New(&database.DB{Pool: db.Pool}, nil, nil, org.ID)
	for _, tt := range []struct {
		field   string
		value   any
		allowed bool
	}{
		{"collection_ids", []string{collection.ID.String()}, false},
		{"COLLECTION_IDS", []string{collection.ID.String()}, false},
		{"collection_idſ", []string{collection.ID.String()}, false},
		{"tag_ids", []string{tag.String()}, false},
		{"tag_idſ", []string{tag.String()}, false},
		{"attribute_ids", []string{attribute.ID.String()}, false},
		{"attribute_idſ", []string{attribute.ID.String()}, false},
		{"attributes", map[string]any{"location_id": other.ID.String()}, true},
		{"ATTRIBUTES", map[string]any{"location_id": other.ID.String()}, true},
		{"attributeſ", map[string]any{"location_id": other.ID.String()}, true},
	} {
		t.Run(tt.field, func(t *testing.T) {
			body, err := json.Marshal(map[string]any{tt.field: tt.value})
			if err != nil {
				t.Fatal(err)
			}
			// Verify that the spelling is accepted by the decoder used by handlers.
			var decoded struct {
				CollectionIDs []string       `json:"collection_ids"`
				TagIDs        []string       `json:"tag_ids"`
				AttributeIDs  []string       `json:"attribute_ids"`
				Attributes    map[string]any `json:"attributes"`
			}
			if err := json.Unmarshal(body, &decoded); err != nil {
				t.Fatal(err)
			}
			if len(decoded.CollectionIDs)+len(decoded.TagIDs)+len(decoded.AttributeIDs)+len(decoded.Attributes) == 0 {
				t.Fatal("test field was not decoded")
			}
			request := httptest.NewRequest(http.MethodPost, "/api/assets", bytes.NewReader(body))
			response := httptest.NewRecorder()
			allowed := h.checkOperationOrganization(response, request, Operation{Path: "/api/assets", Action: "write"})
			if allowed != tt.allowed {
				t.Fatalf("allowed=%v, want %v (status=%d)", allowed, tt.allowed, response.Code)
			}
			if !allowed && response.Code != http.StatusNotFound {
				t.Fatalf("status=%d, want 404", response.Code)
			}
			restored, err := io.ReadAll(request.Body)
			if err != nil || !bytes.Equal(restored, body) {
				t.Fatal("request body changed before handler decoding")
			}
		})
	}
}
