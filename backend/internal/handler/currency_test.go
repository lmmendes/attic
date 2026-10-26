package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lmmendes/attic/internal/auth"
	"github.com/lmmendes/attic/internal/domain"
)

func TestGetOrganizationCurrenciesUsesCurrentRegistry(t *testing.T) {
	h := handlerWithFeatureRepository(nil)
	rec := httptest.NewRecorder()
	h.GetOrganizationCurrencies(rec, httptest.NewRequest(http.MethodGet, "/api/organization/settings/currencies", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status %d: %s", rec.Code, rec.Body.String())
	}
	var response CurrencyOptionsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(response.Currencies, domain.AvailableCurrencies()) {
		t.Fatal("endpoint diverges from validation registry")
	}
	for _, code := range response.Currencies {
		normalized, err := domain.NormalizeCurrency(code)
		if err != nil || normalized != code {
			t.Fatalf("offered unsupported code %q: %v", code, err)
		}
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(rec.Body.Bytes(), &fields); err != nil {
		t.Fatal(err)
	}
	if len(fields) != 1 || fields["currencies"] == nil {
		t.Fatal("endpoint should expose only the code list")
	}
}

func TestCurrencyOptionsRequireAuthenticationButNotAdmin(t *testing.T) {
	middleware, err := auth.NewMiddleware(t.Context(), auth.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sessions := auth.NewSessionManager("currency-options-test-session", 24)
	middleware.SetSessionManager(sessions)
	h := handlerWithFeatureRepository(nil)
	router := chi.NewRouter()
	router.Use(middleware.Authenticate)
	router.Get("/api/organization/settings/currencies", h.GetOrganizationCurrencies)
	req := httptest.NewRequest(http.MethodGet, "/api/organization/settings/currencies", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("anonymous access: %d", rec.Code)
	}
	login := httptest.NewRecorder()
	user := &domain.User{ID: uuid.New(), Email: "member@example.test", Role: domain.UserRoleUser}
	if err := sessions.CreateSession(login, req, user); err != nil {
		t.Fatal(err)
	}
	for _, cookie := range login.Result().Cookies() {
		req.AddCookie(cookie)
	}
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("signed-in member access: %d %s", rec.Code, rec.Body.String())
	}
}
