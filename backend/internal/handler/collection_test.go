package handler

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestCollectionRequestValidation(t *testing.T) {
	for _, body := range []string{`{}`, `{"name":"  "}`, `{"name":"Games","icon":"https://example.com/icon"}`, `{"name":"` + strings.Repeat("a", 256) + `"}`, `{"name":"Games","description":"` + strings.Repeat("a", 2001) + `"}`} {
		t.Run(body[:min(len(body), 40)], func(t *testing.T) {
			w := httptest.NewRecorder()
			if _, ok := decodeCollectionRequest(w, httptest.NewRequest("POST", "/api/collections", strings.NewReader(body))); ok || w.Code != 400 {
				t.Fatalf("invalid request accepted: %d", w.Code)
			}
		})
	}
	w := httptest.NewRecorder()
	req, ok := decodeCollectionRequest(w, httptest.NewRequest("POST", "/api/collections", strings.NewReader(`{"name":"  Games  ","description":"  Our favorites  "}`)))
	if !ok || req.Name != "Games" || req.Icon != "i-lucide-library" || *req.Description != "Our favorites" {
		t.Fatalf("normalization: %+v", req)
	}
}

func TestParseCollectionIDs(t *testing.T) {
	id := uuid.New()
	values, err := parseCollectionIDs([]string{id.String(), id.String()})
	if err != nil || len(values) != 1 || values[0] != id {
		t.Fatal("expected deduplicated UUIDs")
	}
	for _, value := range []string{"bad", "", uuid.Nil.String()} {
		if _, err := parseCollectionIDs([]string{value}); err == nil {
			t.Fatalf("accepted %q", value)
		}
	}
	omitted, err := parseCollectionIDs(nil)
	if err != nil || omitted != nil {
		t.Fatal("omission should preserve assignments")
	}
	empty, err := parseCollectionIDs([]string{})
	if err != nil || empty == nil || len(empty) != 0 {
		t.Fatal("empty should clear assignments")
	}
}
