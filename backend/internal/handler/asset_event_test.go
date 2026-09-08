package handler

import (
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
	req := httptest.NewRequest(http.MethodPost, "/api/assets/id/events", strings.NewReader(`{
		"title":"  Repaired  ","description":"  New belt  ",
		"icon":"i-lucide-wrench","event_date":"2030-02-14"
	}`))
	rec := httptest.NewRecorder()
	decoded, eventDate, ok := decodeAssetEventRequest(rec, req)
	if !ok || decoded.Title != "Repaired" || decoded.Description == nil || *decoded.Description != "New belt" {
		t.Fatalf("unexpected decoded request: %#v", decoded)
	}
	if eventDate.Format("2006-01-02") != "2030-02-14" {
		t.Fatalf("unexpected event date: %v", eventDate)
	}
}

func Test_decodeAssetEventRequest_RejectsInvalidFields(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{"blank title", `{"title":" ","icon":"i-lucide-wrench","event_date":"2026-09-08"}`, "title must contain"},
		{"bad icon", `{"title":"Repair","icon":"<script>","event_date":"2026-09-08"}`, "icon must be"},
		{"bad date", `{"title":"Repair","icon":"i-lucide-wrench","event_date":"2026-02-30"}`, "event_date must be"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/assets/id/events", strings.NewReader(test.body))
			rec := httptest.NewRecorder()
			_, _, ok := decodeAssetEventRequest(rec, req)
			if ok || rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), test.want) {
				t.Fatalf("expected validation error containing %q, status=%d body=%s", test.want, rec.Code, rec.Body.String())
			}
		})
	}
}

func Test_assetEventResponse_FormatsDateWithoutTimestamp(t *testing.T) {
	event := &domain.AssetEvent{
		ID: uuid.New(), AssetID: uuid.New(), Title: "Repair", Icon: "i-lucide-wrench",
		EventDate: time.Date(2026, 9, 8, 0, 0, 0, 0, time.UTC), CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	rec := httptest.NewRecorder()
	writeJSON(rec, http.StatusOK, assetEventResponse(event))
	var response map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response["event_date"] != "2026-09-08" {
		t.Fatalf("expected date-only response, got %v", response["event_date"])
	}
}
