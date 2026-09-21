package handler

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestTagRequestValidation(t *testing.T) {
	for _, body := range []string{`{}`, `{"name":"  "}`, `{"name":"` + strings.Repeat("a", 101) + `"}`, `{"name":"Retro","description":"` + strings.Repeat("a", 2001) + `"}`} {
		w := httptest.NewRecorder()
		if _, ok := decodeTagRequest(w, httptest.NewRequest("POST", "/api/tags", strings.NewReader(body))); ok || w.Code != 400 {
			t.Fatalf("invalid request accepted: %s", body[:min(40, len(body))])
		}
	}
	w := httptest.NewRecorder()
	req, ok := decodeTagRequest(w, httptest.NewRequest("POST", "/api/tags", strings.NewReader(`{"name":" Retro ","description":" Favorites "}`)))
	if !ok || req.Name != "Retro" || req.Description == nil || *req.Description != "Favorites" {
		t.Fatalf("normalization failed: %+v", req)
	}
}

func TestParseTagAssignment(t *testing.T) {
	id := uuid.New().String()
	ids, names, provided, err := parseTagAssignment(&[]string{id, id}, &[]string{" Retro ", "retro", "Portable"})
	if err != nil || !provided || len(ids) != 1 || len(names) != 2 || names[0] != "Retro" {
		t.Fatalf("unexpected assignment: %v %v %v %v", ids, names, provided, err)
	}
	if _, _, provided, err := parseTagAssignment(nil, nil); err != nil || provided {
		t.Fatal("omission should preserve assignments")
	}
	empty := []string{}
	ids, names, provided, err = parseTagAssignment(&empty, &empty)
	if err != nil || !provided || ids == nil || names == nil {
		t.Fatal("explicit empty arrays should clear assignments")
	}
	bad := []string{"bad"}
	if _, _, _, err := parseTagAssignment(&bad, nil); err == nil {
		t.Fatal("invalid UUID accepted")
	}
}
