package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/lmmendes/attic/internal/domain"
	"github.com/lmmendes/attic/internal/repository"
	"github.com/lmmendes/attic/internal/testutil"
)

func TestWarrantyPolicyNumberHTTPIntegration(t *testing.T) {
	ctx := context.Background()
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	db, err := testutil.NewTestDB(ctx)
	must(err)
	defer db.Close(ctx)
	fixtures := testutil.NewFixtures(db.Pool)
	org, err := fixtures.CreateOrganization(ctx, "Warranty HTTP")
	must(err)
	category, err := fixtures.CreateCategory(ctx, org.ID, "Electronics", nil)
	must(err)
	asset, err := fixtures.CreateAsset(ctx, org.ID, category.ID, "Phone")
	must(err)
	h := New(nil, &Repositories{
		Assets:     repository.NewAssetRepository(db.Pool),
		Warranties: repository.NewWarrantyRepository(db.Pool),
	}, nil, org.ID)
	router := chi.NewRouter()
	router.Post("/api/assets/{id}/warranty", h.CreateWarranty)
	router.Put("/api/assets/{id}/warranty", h.UpdateWarranty)
	router.Get("/api/assets/{id}/warranty", h.GetWarranty)
	router.Get("/api/warranties", h.ListWarranties)
	router.Get("/api/warranties/expiring", h.ListExpiringWarranties)
	path := "/api/assets/" + asset.ID.String() + "/warranty"
	request := func(method, path, body string, status int) []byte {
		t.Helper()
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != status {
			t.Fatalf("%s %s: got %d, want %d: %s", method, path, rec.Code, status, rec.Body)
		}
		return rec.Body.Bytes()
	}
	checkPolicy := func(body []byte, want *string) {
		t.Helper()
		var warranty domain.Warranty
		must(json.Unmarshal(body, &warranty))
		if want == nil {
			if warranty.PolicyNumber != nil {
				t.Fatalf("expected no policy number: %s", body)
			}
		} else if warranty.PolicyNumber == nil || *warranty.PolicyNumber != *want {
			t.Fatalf("expected policy number %q: %s", *want, body)
		}
	}
	policy := "DESAMGEB6L2026" + strings.Repeat("x", 14)
	endDate := time.Now().AddDate(0, 0, 10).Format("2006-01-02")
	body, err := json.Marshal(map[string]string{"provider": "Samsung Care+", "policy_number": policy, "end_date": endDate})
	must(err)
	checkPolicy(request(http.MethodPost, path, string(body), http.StatusCreated), &policy)
	checkPolicy(request(http.MethodGet, path, "", http.StatusOK), &policy)
	for _, listPath := range []string{"/api/warranties", "/api/warranties/expiring"} {
		var warranties []json.RawMessage
		must(json.Unmarshal(request(http.MethodGet, listPath, "", http.StatusOK), &warranties))
		if len(warranties) != 1 {
			t.Fatalf("%s: expected one warranty, got %d", listPath, len(warranties))
		}
		checkPolicy(warranties[0], &policy)
	}
	policy = "123ABC"
	checkPolicy(request(http.MethodPut, path, `{"policy_number":"123ABC"}`, http.StatusOK), &policy)
	checkPolicy(request(http.MethodGet, path, "", http.StatusOK), &policy)
	for _, clearBody := range []string{`{}`, `{"policy_number":null}`, `{"policy_number":""}`} {
		request(http.MethodPut, path, `{"policy_number":"123ABC"}`, http.StatusOK)
		var want *string
		if clearBody == `{"policy_number":""}` {
			empty := ""
			want = &empty
		}
		checkPolicy(request(http.MethodPut, path, clearBody, http.StatusOK), want)
		checkPolicy(request(http.MethodGet, path, "", http.StatusOK), want)
	}
}
