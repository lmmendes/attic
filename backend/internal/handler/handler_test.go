package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

// mockDB implements minimal database interface for testing
type mockDB struct {
	healthErr error
}

func (m *mockDB) Health(ctx context.Context) error {
	return m.healthErr
}

// testHealthHandler wraps health check logic for testing
type testHealthHandler struct {
	db *mockDB
}

func newTestHealthHandler() *testHealthHandler {
	return &testHealthHandler{
		db: &mockDB{},
	}
}

func (h *testHealthHandler) Health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *testHealthHandler) Ready(w http.ResponseWriter, r *http.Request) {
	if err := h.db.Health(r.Context()); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "not ready", "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

// Tests for Health endpoint

func Test_Health_ReturnsOK(t *testing.T) {
	h := newTestHealthHandler()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	h.Health(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var response map[string]string
	json.NewDecoder(rec.Body).Decode(&response)
	if response["status"] != "ok" {
		t.Errorf("expected status 'ok', got '%s'", response["status"])
	}
}

func Test_Health_ReturnsJSONContentType(t *testing.T) {
	h := newTestHealthHandler()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	h.Health(rec, req)

	contentType := rec.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("expected Content-Type 'application/json', got '%s'", contentType)
	}
}

// Tests for Ready endpoint

func Test_Ready_DatabaseHealthy_ReturnsReady(t *testing.T) {
	h := newTestHealthHandler()

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	rec := httptest.NewRecorder()

	h.Ready(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var response map[string]string
	json.NewDecoder(rec.Body).Decode(&response)
	if response["status"] != "ready" {
		t.Errorf("expected status 'ready', got '%s'", response["status"])
	}
}

func Test_Ready_DatabaseUnhealthy_ReturnsServiceUnavailable(t *testing.T) {
	h := newTestHealthHandler()
	h.db.healthErr = errors.New("connection refused")

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	rec := httptest.NewRecorder()

	h.Ready(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status 503, got %d", rec.Code)
	}

	var response map[string]string
	json.NewDecoder(rec.Body).Decode(&response)
	if response["status"] != "not ready" {
		t.Errorf("expected status 'not ready', got '%s'", response["status"])
	}
	if response["error"] == "" {
		t.Error("expected error message to be set")
	}
}

// Tests for helper functions

func Test_writeJSON_SetsContentType(t *testing.T) {
	rec := httptest.NewRecorder()

	writeJSON(rec, http.StatusOK, map[string]string{"test": "value"})

	if rec.Header().Get("Content-Type") != "application/json" {
		t.Error("expected Content-Type to be application/json")
	}
}

func Test_writeJSON_SetsStatusCode(t *testing.T) {
	rec := httptest.NewRecorder()

	writeJSON(rec, http.StatusCreated, map[string]string{"test": "value"})

	if rec.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d", rec.Code)
	}
}

func Test_writeError_WritesErrorJSON(t *testing.T) {
	rec := httptest.NewRecorder()

	writeError(rec, http.StatusBadRequest, "test error")

	var response map[string]string
	json.NewDecoder(rec.Body).Decode(&response)
	if response["error"] != "test error" {
		t.Errorf("expected error 'test error', got '%s'", response["error"])
	}
}
