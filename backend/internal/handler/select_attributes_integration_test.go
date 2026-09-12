package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lmmendes/attic/internal/auth"
	"github.com/lmmendes/attic/internal/domain"
	"github.com/lmmendes/attic/internal/repository"
	"github.com/lmmendes/attic/internal/testutil"
)

func TestSelectHTTPPermissionsAndRequiredInheritance(t *testing.T) {
	ctx := context.Background()
	db, err := testutil.NewTestDB(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close(ctx)
	f := testutil.NewFixtures(db.Pool)
	org, err := f.CreateOrganization(ctx, "Select HTTP")
	if err != nil {
		t.Fatal(err)
	}
	parent, err := f.CreateCategory(ctx, org.ID, "Parent", nil)
	if err != nil {
		t.Fatal(err)
	}
	child, err := f.CreateCategory(ctx, org.ID, "Child", &parent.ID)
	if err != nil {
		t.Fatal(err)
	}
	repos := &Repositories{Attributes: repository.NewAttributeRepository(db.Pool), Categories: repository.NewCategoryRepository(db.Pool), Assets: repository.NewAssetRepository(db.Pool), Organizations: repository.NewOrganizationRepository(db.Pool)}
	h := New(nil, repos, nil, org.ID)
	userRepo := repository.NewUserRepository(db.Pool)
	sessions := auth.NewSessionManager("test-secret", 24)
	authentication, err := auth.NewMiddleware(ctx, auth.Config{})
	if err != nil {
		t.Fatal(err)
	}
	authentication.SetSessionManager(sessions)
	router := chi.NewRouter()
	router.Use(authentication.Authenticate)
	router.Use(auth.NewUserProvisioner(userRepo, org.ID).LoadLocalUser)
	router.Post("/api/attributes", h.CreateAttribute)
	router.Get("/api/attributes/{id}", h.GetAttribute)
	router.Post("/api/attributes/{id}/impact-preview", h.PreviewAttributeImpact)
	router.Delete("/api/attributes/{id}/options/{option_id}", h.DeleteAttributeOption)
	router.Put("/api/attributes/{id}", h.UpdateAttribute)
	admin, err := f.CreateUser(ctx, org.ID, "admin@example.test")
	if err != nil {
		t.Fatal(err)
	}
	admin.Role = domain.UserRoleAdmin
	if err = userRepo.Update(ctx, admin); err != nil {
		t.Fatal(err)
	}
	member, err := f.CreateUser(ctx, org.ID, "member@example.test")
	if err != nil {
		t.Fatal(err)
	}
	request := func(method, path, body, token string, user *domain.User) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		if user != nil {
			login := httptest.NewRecorder()
			if err := sessions.CreateSession(login, req, user); err != nil {
				t.Fatal(err)
			}
			for _, cookie := range login.Result().Cookies() {
				req.AddCookie(cookie)
			}
		}
		req.Header.Set("X-Impact-Token", token)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec
	}
	create := `{"name":"Vendor","key":"vendor","data_type":"select","selection_mode":"single","options":[{"label":"Acme","value":"acme"}]}`
	if response := request("POST", "/api/attributes", create, "", nil); response.Code != 401 {
		t.Fatalf("missing session: %d %s", response.Code, response.Body)
	}
	// A stale session role must not override the current database role.
	member.Role = domain.UserRoleAdmin
	if response := request("POST", "/api/attributes", create, "", member); response.Code != 403 {
		t.Fatalf("member write: %d %s", response.Code, response.Body)
	}
	response := request("POST", "/api/attributes", create, "", admin)
	if response.Code != 201 {
		t.Fatalf("create: %d %s", response.Code, response.Body)
	}
	var field domain.Attribute
	if err = json.Unmarshal(response.Body.Bytes(), &field); err != nil {
		t.Fatal(err)
	}
	if len(field.Options) != 1 {
		t.Fatalf("options: %+v", field)
	}
	if err = repos.Categories.SetAttributes(ctx, parent.ID, []domain.CategoryAttributeAssignment{{AttributeID: field.ID, Required: true}}); err != nil {
		t.Fatal(err)
	}
	inherited, err := repos.Categories.GetByIDWithInheritedAttributes(ctx, org.ID, child.ID)
	if err != nil || len(inherited.Attributes) != 1 || !inherited.Attributes[0].Inherited || len(inherited.Attributes[0].Attribute.Options) != 1 {
		t.Fatalf("inherited select: %+v %v", inherited, err)
	}
	asset := &domain.Asset{OrganizationID: org.ID, CategoryID: &child.ID, Name: "Inherited", Quantity: 1}
	if err = repos.Assets.Create(ctx, asset); err == nil {
		t.Fatal("missing required inherited select accepted")
	}
	asset.Attributes = json.RawMessage(`{"vendor":"acme"}`)
	if err = repos.Assets.Create(ctx, asset); err != nil {
		t.Fatal(err)
	}
	optionID := field.Options[0].ID.String()
	path := "/api/attributes/" + field.ID.String()
	previewBody := `{"action":"delete_option","option_id":"` + optionID + `","changes":{}}`
	if response = request("POST", path+"/impact-preview", previewBody, "", member); response.Code != 403 {
		t.Fatalf("member preview: %d", response.Code)
	}
	response = request("POST", path+"/impact-preview", previewBody, "", admin)
	if response.Code != 200 {
		t.Fatalf("preview: %s", response.Body)
	}
	var impact repository.AttributeImpact
	json.Unmarshal(response.Body.Bytes(), &impact)
	if impact.AffectedAssets != 1 || impact.RequiredFieldsLeftEmpty != 1 {
		t.Fatalf("required impact: %+v", impact)
	}
	if response = request("DELETE", path+"/options/"+optionID, "", "", admin); response.Code != 409 {
		t.Fatalf("missing confirmation: %d", response.Code)
	}
	if response = request("DELETE", path+"/options/"+optionID, "", impact.ConfirmationToken, admin); response.Code != 204 {
		t.Fatalf("confirm: %d %s", response.Code, response.Body)
	}
	current, err := repos.Assets.GetByID(ctx, asset.ID)
	if err != nil {
		t.Fatal(err)
	}
	if string(current.Attributes) != "{}" {
		t.Fatalf("delete did not clear selection: %s", current.Attributes)
	}
	if err = repos.Assets.Update(ctx, current); err == nil {
		t.Fatal("empty required select accepted on normal save")
	}
	foreignOrg, err := f.CreateOrganization(ctx, "Other organization")
	if err != nil {
		t.Fatal(err)
	}
	foreign, err := f.CreateUser(ctx, foreignOrg.ID, "foreign@example.test")
	if err != nil {
		t.Fatal(err)
	}
	foreign.Role = domain.UserRoleAdmin
	if err = userRepo.Update(ctx, foreign); err != nil {
		t.Fatal(err)
	}
	if response = request("POST", path+"/impact-preview", previewBody, "", foreign); response.Code != http.StatusUnauthorized {
		t.Fatalf("foreign admin: %d", response.Code)
	}
	multiple := `{"name":"adsd","key":"adsd","data_type":"select","selection_mode":"multiple","options":[{"id":"5123c56b-3273-41b3-9ce8-e01b9a1a95b3","label":"das","value":"das","sort_order":0}]}`
	if response = request("POST", "/api/attributes", multiple, "", admin); response.Code != http.StatusCreated {
		t.Fatalf("multiple select: %d %s", response.Code, response.Body)
	}
	var multipleField domain.Attribute
	if err = json.Unmarshal(response.Body.Bytes(), &multipleField); err != nil {
		t.Fatal(err)
	}
	multipleField.Options[0].Label = "DAS"
	changes, err := json.Marshal(map[string]any{
		"name": multipleField.Name, "key": multipleField.Key, "data_type": multipleField.DataType,
		"selection_mode": multipleField.SelectionMode, "options": multipleField.Options,
	})
	if err != nil {
		t.Fatal(err)
	}
	previewRequest, err := json.Marshal(map[string]any{"action": "update_attribute", "changes": json.RawMessage(changes)})
	if err != nil {
		t.Fatal(err)
	}
	response = request("POST", "/api/attributes/"+multipleField.ID.String()+"/impact-preview", string(previewRequest), "", admin)
	if response.Code != http.StatusOK {
		t.Fatalf("batch preview: %d %s", response.Code, response.Body)
	}
	if err = json.Unmarshal(response.Body.Bytes(), &impact); err != nil {
		t.Fatal(err)
	}
	response = request("PUT", "/api/attributes/"+multipleField.ID.String(), string(changes), impact.ConfirmationToken, admin)
	if response.Code != http.StatusOK {
		t.Fatalf("batch save: %d %s", response.Code, response.Body)
	}
	if err = userRepo.Delete(ctx, admin.ID); err != nil {
		t.Fatal(err)
	}
	if response = request("POST", "/api/attributes", create, "", admin); response.Code != http.StatusUnauthorized {
		t.Fatalf("deleted account: %d", response.Code)
	}
	admin.ID = uuid.New()
	if response = request("POST", "/api/attributes", create, "", admin); response.Code != http.StatusUnauthorized {
		t.Fatalf("missing account: %d", response.Code)
	}
}
