package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/lmmendes/attic/internal/domain"
)

func Test_decodeAssetEventRequest_ValidInput(t *testing.T) {
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/assets/id/events", strings.NewReader(`{
		"title":"  Repaired  ","description":"  New belt  ",
		"icon":"i-lucide-wrench","occurred_at":"2030-02-14T16:30:00+01:00"
	}`))
	rec := httptest.NewRecorder()
	decoded, occurredAt, ok := decodeAssetEventRequest(rec, req)
	if !ok || decoded.Title != "Repaired" || decoded.Description != "New belt" {
		t.Fatalf("unexpected decoded request: %#v", decoded)
	}
	if occurredAt.Format(time.RFC3339) != "2030-02-14T16:30:00+01:00" {
		t.Fatalf("unexpected occurrence timestamp: %v", occurredAt)
	}
}

func Test_decodeAssetEventRequest_RejectsInvalidFields(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{"blank title", `{"title":" ","description":"New belt","icon":"i-lucide-wrench","occurred_at":"2026-09-08T10:00:00Z"}`, "title must contain"},
		{"blank description", `{"title":"Repair","description":" ","icon":"i-lucide-wrench","occurred_at":"2026-09-08T10:00:00Z"}`, "description must contain"},
		{"bad icon", `{"title":"Repair","description":"New belt","icon":"<script>","occurred_at":"2026-09-08T10:00:00Z"}`, "icon must be"},
		{"bad timestamp", `{"title":"Repair","description":"New belt","icon":"i-lucide-wrench","occurred_at":"2026-02-30T10:00:00Z"}`, "occurred_at must be"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/assets/id/events", strings.NewReader(test.body))
			rec := httptest.NewRecorder()
			_, _, ok := decodeAssetEventRequest(rec, req)
			if ok || rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), test.want) {
				t.Fatalf("expected validation error containing %q, status=%d body=%s", test.want, rec.Code, rec.Body.String())
			}
		})
	}
}

func Test_assetEventResponse_IncludesOccurrenceTimestamp(t *testing.T) {
	event := &domain.AssetEvent{
		ID: uuid.New(), AssetID: uuid.New(), Title: "Repair", Description: "Replaced belt", Icon: "i-lucide-wrench",
		OccurredAt: time.Date(2026, 9, 8, 14, 30, 0, 0, time.UTC), CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	rec := httptest.NewRecorder()
	writeJSON(rec, http.StatusOK, assetEventResponse(event))
	var response map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response["occurred_at"] != "2026-09-08T14:30:00Z" {
		t.Fatalf("expected occurrence timestamp, got %v", response)
	}
}
