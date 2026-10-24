package handler

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
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
