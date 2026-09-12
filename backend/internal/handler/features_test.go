package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/lmmendes/attic/internal/domain"
)

type mockOrganizationFeatureRepository struct {
	features    *domain.OrganizationFeatures
	getCalls    int
	updated     *domain.OrganizationFeatures
	featuresErr error
	updateErr   error
}

func (m *mockOrganizationFeatureRepository) GetByID(context.Context, uuid.UUID) (*domain.Organization, error) {
	return nil, nil
}

func (m *mockOrganizationFeatureRepository) GetDefault(context.Context) (*domain.Organization, error) {
	return nil, nil
}

func (m *mockOrganizationFeatureRepository) Create(context.Context, *domain.Organization) error {
	return nil
}

func (m *mockOrganizationFeatureRepository) Update(context.Context, *domain.Organization) error {
	return nil
}

func (m *mockOrganizationFeatureRepository) GetFeatures(context.Context, uuid.UUID) (*domain.OrganizationFeatures, error) {
	m.getCalls++
	return m.features, m.featuresErr
}

func (m *mockOrganizationFeatureRepository) UpdateFeatures(_ context.Context, _ uuid.UUID, features *domain.OrganizationFeatures) error {
	m.updated = features
	return m.updateErr
}

func handlerWithFeatureRepository(repo domain.OrganizationRepository) *Handler {
	return New(nil, &Repositories{Organizations: repo}, nil, uuid.New())
}

func requestWithFeatures(features *domain.OrganizationFeatures) *http.Request {
	req := httptest.NewRequest("GET", "/api/assets", nil)
	ctx := context.WithValue(req.Context(), organizationFeaturesContextKey{}, features)
	return req.WithContext(ctx)
}

func TestSanitizeAssetHidesDisabledRelationsAndPluginValues(t *testing.T) {
	pluginID := "google_books"
	categoryID, locationID, conditionID := uuid.New(), uuid.New(), uuid.New()
	asset := &domain.Asset{
		CategoryID:     &categoryID,
		Category:       &domain.Category{ID: categoryID, PluginID: &pluginID},
		LocationID:     &locationID,
		Location:       &domain.Location{ID: locationID},
		ConditionID:    &conditionID,
		Condition:      &domain.Condition{ID: conditionID},
		Attributes:     json.RawMessage(`{"serial":"user-value","plugin.google_books.books.isbn":"123"}`),
		ImportPluginID: &pluginID,
	}
	features := allFeaturesEnabled()
	features.Locations = false
	features.Conditions = false
	features.Plugins = false

	if err := (&Handler{}).sanitizeAsset(requestWithFeatures(features), asset); err != nil {
		t.Fatalf("sanitizeAsset returned an error: %v", err)
	}
	if asset.LocationID != nil || asset.Location != nil || asset.ConditionID != nil || asset.Condition != nil {
		t.Fatal("disabled relations were not hidden")
	}
	if asset.CategoryID != nil || asset.Category != nil || asset.ImportPluginID != nil {
		t.Fatal("plugin-owned metadata was not hidden")
	}
	var attributes map[string]any
	if err := json.Unmarshal(asset.Attributes, &attributes); err != nil {
		t.Fatal(err)
	}
	if attributes["serial"] != "user-value" {
		t.Fatalf("user attribute was not preserved: %#v", attributes)
	}
	if _, exists := attributes["plugin.google_books.books.isbn"]; exists {
		t.Fatal("plugin attribute was exposed")
	}
}

func TestMergePreservedPluginAttributes(t *testing.T) {
	merged, err := mergePreservedPluginAttributes(
		[]byte(`{"serial":"old","plugin.google_books.books.isbn":"123"}`),
		[]byte(`{"serial":"new"}`),
	)
	if err != nil {
		t.Fatal(err)
	}
	var attributes map[string]any
	if err := json.Unmarshal(merged, &attributes); err != nil {
		t.Fatal(err)
	}
	if attributes["serial"] != "new" || attributes["plugin.google_books.books.isbn"] != "123" {
		t.Fatalf("unexpected merged attributes: %#v", attributes)
	}
}

func TestPluginOwnedCategoryIsPreservedWhenHiddenFromEditPayload(t *testing.T) {
	pluginID := "google_books"
	category := &domain.Category{PluginID: &pluginID}
	features := allFeaturesEnabled()
	features.Plugins = false

	if !shouldPreserveHiddenPluginCategory(features, nil, category) {
		t.Fatal("expected hidden plugin category to be preserved")
	}
	requestedCategoryID := uuid.NewString()
	if shouldPreserveHiddenPluginCategory(features, &requestedCategoryID, category) {
		t.Fatal("explicit replacement category should not preserve the hidden category")
	}
}

func TestRejectDisabledAssetFields(t *testing.T) {
	features := allFeaturesEnabled()
	features.Categories = false
	request := requestWithFeatures(features)
	categoryID := uuid.NewString()

	err := (&Handler{}).rejectDisabledAssetFields(request, &categoryID, nil, nil, nil, nil)
	if want := errFeatureDisabled("categories"); err != want {
		t.Fatalf("expected %v, got %v", want, err)
	}
}

func TestRejectPluginAttributesWhenPluginsDisabled(t *testing.T) {
	features := allFeaturesEnabled()
	features.Plugins = false
	request := requestWithFeatures(features)

	err := (&Handler{}).rejectDisabledAssetFields(request, nil, nil, nil, nil, []byte(`{"plugin.google_books.books.isbn":"123"}`))
	if _, ok := err.(featureDisabledError); !ok {
		t.Fatalf("expected featureDisabledError, got %v", err)
	}
}

func TestLoadFeaturesSharesSnapshotAcrossHandlerChecks(t *testing.T) {
	repo := &mockOrganizationFeatureRepository{features: allFeaturesEnabled()}
	h := handlerWithFeatureRepository(repo)
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for _, name := range []string{"locations", "plugins"} {
			enabled, err := h.featureEnabled(r, name)
			if err != nil || !enabled {
				t.Fatalf("feature %q was not available: enabled=%v err=%v", name, enabled, err)
			}
		}
		w.WriteHeader(http.StatusNoContent)
	})

	recorder := httptest.NewRecorder()
	h.LoadFeatures(next).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/assets", nil))

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, recorder.Code)
	}
	if repo.getCalls != 1 {
		t.Fatalf("expected one feature query, got %d", repo.getCalls)
	}
}

func TestLoadFeaturesFailsClosed(t *testing.T) {
	repo := &mockOrganizationFeatureRepository{featuresErr: errors.New("database unavailable")}
	h := handlerWithFeatureRepository(repo)
	called := false
	next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true })

	recorder := httptest.NewRecorder()
	h.LoadFeatures(next).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/assets", nil))

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, recorder.Code)
	}
	if called {
		t.Fatal("downstream handler ran without feature settings")
	}
}

func TestUpdateOrganizationFeaturesRequiresCompleteMap(t *testing.T) {
	repo := &mockOrganizationFeatureRepository{features: allFeaturesEnabled()}
	h := handlerWithFeatureRepository(repo)
	request := httptest.NewRequest(http.MethodPut, "/api/organization/features", strings.NewReader(`{"locations":true}`))
	recorder := httptest.NewRecorder()

	h.UpdateOrganizationFeatures(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, recorder.Code)
	}
	if repo.updated != nil {
		t.Fatal("incomplete feature map was persisted")
	}
}

func TestUpdateOrganizationFeaturesRejectsNullValues(t *testing.T) {
	repo := &mockOrganizationFeatureRepository{features: allFeaturesEnabled()}
	h := handlerWithFeatureRepository(repo)
	body := `{"locations":null,"collections":true,"categories":true,"attributes":true,"conditions":true,"warranties":true,"plugins":true}`
	request := httptest.NewRequest(http.MethodPut, "/api/organization/features", strings.NewReader(body))
	recorder := httptest.NewRecorder()

	h.UpdateOrganizationFeatures(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, recorder.Code)
	}
	if recorder.Body.String() != "{\"error\":\"feature values must be booleans\"}\n" {
		t.Fatalf("unexpected response: %s", recorder.Body.String())
	}
	if repo.updated != nil {
		t.Fatal("feature map containing null was persisted")
	}
}

func TestUpdateOrganizationFeaturesPersistsCompleteMap(t *testing.T) {
	repo := &mockOrganizationFeatureRepository{features: allFeaturesEnabled()}
	h := handlerWithFeatureRepository(repo)
	body := `{"locations":false,"collections":true,"categories":true,"attributes":false,"conditions":true,"warranties":true,"plugins":false}`
	request := httptest.NewRequest(http.MethodPut, "/api/organization/features", strings.NewReader(body))
	recorder := httptest.NewRecorder()

	h.UpdateOrganizationFeatures(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, recorder.Code, recorder.Body.String())
	}
	if repo.updated == nil || repo.updated.Locations || !repo.updated.Attributes || repo.updated.Plugins {
		t.Fatalf("unexpected persisted settings: %#v", repo.updated)
	}
}
