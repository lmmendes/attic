package handler

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lmmendes/attic/internal/auth"
	"github.com/lmmendes/attic/internal/database"
	"github.com/lmmendes/attic/internal/domain"
	"github.com/lmmendes/attic/internal/plugin"
	"github.com/lmmendes/attic/internal/repository"
	"github.com/lmmendes/attic/internal/testutil"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type keyTransport struct{ key string }

func (t keyTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	r = r.Clone(r.Context())
	r.Header.Set("Authorization", "Bearer "+t.key)
	return http.DefaultTransport.RoundTrip(r)
}

func TestAPIKeysRESTAndMCPIntegration(t *testing.T) {
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
	org, err := fixtures.CreateOrganization(ctx, "Keys")
	must(err)
	foreignOrg, err := fixtures.CreateOrganization(ctx, "Other")
	must(err)
	users := repository.NewUserRepository(db.Pool)
	admin, err := fixtures.CreateUser(ctx, org.ID, "admin@keys.test")
	must(err)
	admin.Role = domain.UserRoleAdmin
	must(users.Update(ctx, admin))
	member, err := fixtures.CreateUser(ctx, org.ID, "member@keys.test")
	must(err)
	other, err := fixtures.CreateUser(ctx, foreignOrg.ID, "foreign@keys.test")
	must(err)
	foreignAsset := &domain.Asset{OrganizationID: foreignOrg.ID, Name: "Foreign"}
	must(fixtures.CreateAssetFull(ctx, foreignAsset))
	repos := &Repositories{Organizations: repository.NewOrganizationRepository(db.Pool), Users: users, Assets: repository.NewAssetRepository(db.Pool), Categories: repository.NewCategoryRepository(db.Pool), Locations: repository.NewLocationRepository(db.Pool), Collections: repository.NewCollectionRepository(db.Pool), Conditions: repository.NewConditionRepository(db.Pool), Attributes: repository.NewAttributeRepository(db.Pool), Tags: repository.NewTagRepository(db.Pool), Attachments: repository.NewAttachmentRepository(db.Pool), AssetEvents: repository.NewAssetEventRepository(db.Pool), Warranties: repository.NewWarrantyRepository(db.Pool), SavedFilters: repository.NewSavedFilterRepository(db.Pool)}
	h := New(&database.DB{Pool: db.Pool}, repos, &keyFileStorage{}, org.ID)
	keys := repository.NewAPIKeyRepository(db.Pool)
	kh := &APIKeyHandler{keys, org.ID}
	sessions := auth.NewSessionManager("key-tests", 24)
	browser, err := auth.NewMiddleware(ctx, auth.Config{})
	must(err)
	browser.SetSessionManager(sessions)
	keyAuth := &auth.APIKeyAuthenticator{Keys: keys, Users: users, OrganizationID: org.ID}
	registry := plugin.NewRegistry()
	must(registry.Register(&mockPlugin{id: "fixture", name: "Fixture", categoryName: "Fixture imports", searchFields: []domain.SearchField{{Key: "title", Label: "Title"}}, searchResults: []domain.SearchResult{{ExternalID: "fixture-1", Title: "Imported"}}, fetchData: &domain.ImportData{Name: "Imported", ExternalID: "fixture-1", Attributes: map[string]any{}}}))
	ops := Operations(h, NewPluginHandler(registry, repos, nil, org.ID), NewUserManagementHandler(users, sessions, 8, org.ID), NewAuthHandler(users, sessions, 8, false, false, false, nil), kh, "test")
	router := chi.NewRouter()
	router.Group(func(r chi.Router) {
		r.Use(keyAuth.Authenticate(browser.Authenticate))
		r.Use(auth.NewUserProvisioner(users, org.ID).LoadLocalUser)
		r.Use(h.LoadFeatures)
		h.RegisterOperations(r, ops)
	})
	router.Handle("/mcp", keyAuth.Authenticate(func(next http.Handler) http.Handler { return auth.RequireAPIKey(next) })(h.MCPHandler(ops, keys)))
	request := func(method, path, body, secret string, user *domain.User, want int) []byte {
		t.Helper()
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		if secret != "" {
			req.Header.Set("Authorization", "Bearer "+secret)
		}
		if user != nil {
			login := httptest.NewRecorder()
			must(sessions.CreateSession(login, req, user))
			for _, cookie := range login.Result().Cookies() {
				req.AddCookie(cookie)
			}
		}
		response := httptest.NewRecorder()
		router.ServeHTTP(response, req)
		if response.Code != want {
			t.Fatalf("%s %s: got %d want %d: %s", method, path, response.Code, want, response.Body)
		}
		return response.Body.Bytes()
	}
	create := func(user *domain.User, permissions map[string]string) (domain.APIKey, string) {
		t.Helper()
		body, _ := json.Marshal(map[string]any{"description": "  Automation  ", "permissions": permissions})
		raw := request("POST", "/api/me/api-keys", string(body), "", user, 201)
		var result struct {
			domain.APIKey
			Secret string `json:"secret"`
		}
		must(json.Unmarshal(raw, &result))
		if result.Description != "Automation" || len(result.Secret) != 49 {
			t.Fatalf("bad creation: %s", raw)
		}
		return result.APIKey, result.Secret
	}
	all := map[string]string{}
	for _, resource := range domain.PermissionCatalog(domain.UserRoleAdmin) {
		all[resource.Resource] = "write"
	}
	adminKey, secret := create(admin, all)
	memberKey, readSecret := create(member, map[string]string{"assets": "read"})
	listed := request("GET", "/api/me/api-keys", "", "", admin, 200)
	if strings.Contains(string(listed), secret) || strings.Contains(string(listed), "secret_hash") {
		t.Fatal("list disclosed secret")
	}
	var hash []byte
	must(db.Pool.QueryRow(ctx, "SELECT secret_hash FROM api_keys WHERE id=$1", adminKey.ID).Scan(&hash))
	if string(hash) != string(domain.HashAPIKey(secret)) {
		t.Fatal("secret not hashed")
	}
	request("PATCH", "/api/me/api-keys/"+adminKey.ID.String(), `{"description":"Renamed"}`, "", admin, 204)
	request("PATCH", "/api/me/api-keys/"+adminKey.ID.String(), `{"permissions":{"assets":"write"}}`, "", admin, 400)
	request("DELETE", "/api/me/api-keys/"+adminKey.ID.String(), "", "", member, 404)

	request("GET", "/api/organization/api-keys", "", "", member, 404)
	request("GET", "/api/organization/api-keys?owner_id="+member.ID.String(), "", "", admin, 404)
	request("DELETE", "/api/organization/api-keys/"+memberKey.ID.String(), "", "", admin, 404)
	request("DELETE", "/api/me/api-keys/"+memberKey.ID.String(), "", "", admin, 404)

	for _, body := range []string{`{"description":" ","permissions":{"assets":"read"}}`, `{"description":"key","permissions":{}}`, `{"description":"key","permissions":{"attributes":"write"}}`, `{"description":"key","permissions":{"assets":"read"},"expires_at":"2000-01-01T00:00:00Z"}`} {
		request("POST", "/api/me/api-keys", body, "", member, 400)
	}
	request("GET", "/api/assets", "", readSecret, nil, 200)
	request("GET", "/api/assets/", "", readSecret, nil, 200)
	request("POST", "/api/assets/search", `{"criteria":{"version":1},"limit":10}`, readSecret, nil, 200)
	request("POST", "/api/assets", `{"name":"Forbidden"}`, readSecret, nil, 403)
	for _, entry := range []struct{ method, path, body string }{{"GET", "/api/me/api-keys", ""}, {"POST", "/api/users", `{}`}, {"PUT", "/api/auth/password", `{}`}, {"POST", "/api/users/" + member.ID.String() + "/reset-password", `{}`}} {
		request(entry.method, entry.path, entry.body, secret, nil, 403)
	}
	request("GET", "/api/assets", "", "invalid", admin, 401)
	foreignKey := &domain.APIKey{ID: uuid.New(), OwnerID: other.ID, OrganizationID: foreignOrg.ID, Description: "foreign", Permissions: all, Prefix: "foreign"}
	foreignSecret, foreignHash, err := domain.NewAPIKeySecret()
	must(err)
	foreignKey.SecretHash = foreignHash
	must(keys.Create(ctx, foreignKey))
	request("GET", "/api/assets", "", foreignSecret, nil, 401)
	request("GET", "/api/assets/"+foreignAsset.ID.String(), "", secret, nil, 404)
	foreignLocation, err := fixtures.CreateLocation(ctx, foreignOrg.ID, "Foreign location", nil)
	must(err)
	request("POST", "/api/assets", `{"name":"Foreign reference","location_id":"`+foreignLocation.ID.String()+`"}`, secret, nil, 404)
	for _, suffix := range []string{" x", ` {"ignored":true}`} {
		request("POST", "/api/assets", `{"name":"Rejected reference","location_id":"`+foreignLocation.ID.String()+`"}`+suffix, secret, nil, 400)
	}
	var rejectedAssets int
	must(db.Pool.QueryRow(ctx, "SELECT count(*) FROM assets WHERE organization_id=$1 AND name='Rejected reference'", org.ID).Scan(&rejectedAssets))
	if rejectedAssets != 0 {
		t.Fatal("invalid write body persisted an asset with a foreign location")
	}
	forged := httptest.NewRequest("POST", "/api/assets", strings.NewReader(`{"name":"Foreign reference","LOCATION_ID":"`+foreignLocation.ID.String()+`"}`))
	forged.Header.Set("Content-Type", "multipart/form-data")
	forged.Header.Set("Authorization", "Bearer "+secret)
	forgedResult := httptest.NewRecorder()
	router.ServeHTTP(forgedResult, forged)
	if forgedResult.Code != 404 {
		t.Fatalf("content type or field case bypassed organization check: %d %s", forgedResult.Code, forgedResult.Body)
	}

	request("POST", "/mcp", `{}`, secret, nil, 404)
	request("POST", "/mcp", `{}`, "", admin, 401)
	request("PUT", "/api/organization/mcp", `{"enabled":true}`, "", admin, 200)
	disabledBrowser, err := auth.NewMiddleware(ctx, auth.Config{Disabled: true, DisabledUser: admin})
	must(err)
	disabledRouter := keyAuth.Authenticate(disabledBrowser.Authenticate)(auth.RequireAPIKey(h.MCPHandler(ops, keys)))
	disabledResponse := httptest.NewRecorder()
	disabledRouter.ServeHTTP(disabledResponse, httptest.NewRequest("POST", "/mcp", strings.NewReader(`{}`)))
	if disabledResponse.Code != 401 {
		t.Fatal("disabled auth bypassed MCP")
	}

	server := httptest.NewServer(router)
	defer server.Close()
	connect := func(key string) *mcp.ClientSession {
		t.Helper()
		client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "1"}, nil)
		session, err := client.Connect(ctx, &mcp.StreamableClientTransport{Endpoint: server.URL + "/mcp", HTTPClient: &http.Client{Transport: keyTransport{key}}, MaxRetries: -1}, nil)
		must(err)
		t.Cleanup(func() { session.Close() })
		return session
	}
	reader := connect(readSecret)
	noArgs, err := reader.CallTool(ctx, &mcp.CallToolParams{Name: "list_assets"})
	must(err)
	if noArgs.IsError {
		t.Fatal("omitted empty arguments rejected", noArgs)
	}
	tools, err := reader.ListTools(ctx, nil)
	must(err)
	found := false
	for _, tool := range tools.Tools {
		if tool.Name == "list_assets" {
			found = true
		}
		if tool.Name == "create_asset" {
			t.Fatal("write tool discovered")
		}
	}
	if !found {
		t.Fatal("read tool missing")
	}
	result, err := reader.CallTool(ctx, &mcp.CallToolParams{Name: "search_assets", Arguments: map[string]any{"body": map[string]any{"criteria": map[string]any{"version": 1}}}})
	must(err)
	if result.IsError {
		t.Fatalf("search failed: %+v", result)
	}
	result, err = reader.CallTool(ctx, &mcp.CallToolParams{Name: "create_asset", Arguments: map[string]any{"body": map[string]any{"name": "Forged"}}})
	if err == nil && !result.IsError {
		t.Fatal("hidden tool callable")
	}
	_, pluginSecret := create(member, map[string]string{"plugins": "write"})
	request("POST", "/api/plugins/fixture/import", `{"external_id":"fixture-1"}`, pluginSecret, nil, 403)
	pluginReader := connect(pluginSecret)
	pluginTools, err := pluginReader.ListTools(ctx, nil)
	must(err)
	for _, tool := range pluginTools.Tools {
		if tool.Name == "import" {
			t.Fatal("import available without asset write")
		}
	}
	writer := connect(secret)
	call := func(name string, args map[string]any, want int) map[string]any {
		t.Helper()
		result, err := writer.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: args})
		must(err)
		raw, _ := json.Marshal(result.StructuredContent)
		var envelope map[string]any
		must(json.Unmarshal(raw, &envelope))
		if envelope["status"] != float64(want) {
			t.Fatalf("%s got %s", name, raw)
		}
		body, _ := envelope["body"].(map[string]any)
		return body
	}
	call("search", map[string]any{"pluginId": "fixture", "query": map[string]any{"field": "title", "q": "Imported"}}, 200)
	imported := call("import", map[string]any{"pluginId": "fixture", "body": map[string]any{"external_id": "fixture-1"}}, 201)
	if imported["asset"].(map[string]any)["name"] != "Imported" {
		t.Fatal("MCP import failed")
	}
	asset := call("create_asset", map[string]any{"body": map[string]any{"name": "MCP asset", "quantity": 1}}, 201)
	id := asset["id"].(string)
	call("get_asset", map[string]any{"id": id}, 200)
	call("update_asset", map[string]any{"id": id, "body": map[string]any{"name": "Updated", "quantity": 2}}, 200)
	attachment := call("upload_attachment", map[string]any{"id": id, "body": map[string]any{"filename": "readme.txt", "content_type": "text/plain", "content": base64.StdEncoding.EncodeToString([]byte("hello"))}}, 201)
	call("get_attachment", map[string]any{"attachmentId": attachment["id"]}, 200)
	call("delete_attachment", map[string]any{"attachmentId": attachment["id"]}, 204)
	// Preserve attribute impact confirmations across the MCP adapter.
	attribute := call("create_attribute", map[string]any{"body": map[string]any{"name": "Vendor", "key": "vendor", "data_type": "select", "selection_mode": "single", "options": []any{map[string]any{"label": "Acme", "value": "acme"}}}}, 201)
	attributeID := attribute["id"].(string)
	var optionID string
	options := attribute["options"].([]any)
	optionID = options[0].(map[string]any)["id"].(string)
	categorized := call("create_category", map[string]any{"body": map[string]any{"name": "Typed", "attributes": []any{map[string]any{"attribute_id": attributeID, "required": false}}}}, 201)
	linked := call("create_asset", map[string]any{"body": map[string]any{"name": "Linked", "category_id": categorized["id"], "attributes": map[string]any{"vendor": "acme"}}}, 201)
	_ = linked
	call("delete_attribute_option", map[string]any{"id": attributeID, "option_id": optionID}, 409)
	impact := call("preview_attribute_impact", map[string]any{"id": attributeID, "body": map[string]any{"action": "delete_option", "option_id": optionID, "changes": map[string]any{}}}, 200)
	call("delete_attribute_option", map[string]any{"id": attributeID, "option_id": optionID, "impact_token": impact["confirmation_token"]}, 204)
	call("delete_asset", map[string]any{"id": id}, 204)
	call("get_asset", map[string]any{"id": uuid.New().String()}, 404)
	features, err := repos.Organizations.GetFeatures(ctx, org.ID)
	must(err)
	features.Locations = false
	must(repos.Organizations.UpdateFeatures(ctx, org.ID, features))
	request("GET", "/api/locations", "", secret, nil, 404)
	tools, err = writer.ListTools(ctx, nil)
	must(err)
	for _, tool := range tools.Tools {
		if tool.Name == "list_locations" {
			t.Fatal("disabled feature discovered")
		}
	}
	admin.Role = domain.UserRoleUser
	must(users.Update(ctx, admin))
	request("GET", "/api/users", "", secret, nil, 403)
	tools, err = writer.ListTools(ctx, nil)
	must(err)
	for _, tool := range tools.Tools {
		if tool.Name == "list_users" || tool.Name == "create_attribute" {
			t.Fatal("role demotion ignored")
		}
	}
	admin.Role = domain.UserRoleAdmin
	must(users.Update(ctx, admin))
	request("PUT", "/api/organization/mcp", `{"enabled":false}`, "", admin, 200)
	if _, err = writer.ListTools(ctx, nil); err == nil {
		t.Fatal("MCP disable ignored")
	}
	request("GET", "/api/assets", "", secret, nil, 200)
	// Admins receive aggregate usage only, scoped to active keys in their workspace.
	firstUsed := time.Now().UTC().Add(-2 * time.Hour).Truncate(time.Second)
	latestUsed := firstUsed.Add(time.Hour)
	_, err = db.Pool.Exec(ctx, "UPDATE api_keys SET last_used_at=$2 WHERE secret_hash=$1", domain.HashAPIKey(readSecret), firstUsed)
	must(err)
	_, err = db.Pool.Exec(ctx, "UPDATE api_keys SET last_used_at=$2 WHERE secret_hash=$1", domain.HashAPIKey(pluginSecret), latestUsed)
	must(err)
	checkUsage := func(count int) {
		t.Helper()
		raw := request("GET", "/api/users", "", "", admin, 200)
		if strings.Contains(string(raw), "permissions") || strings.Contains(string(raw), "prefix") {
			t.Fatal("user list exposes key metadata")
		}
		var entries []struct {
			ID       string     `json:"id"`
			Count    int        `json:"api_key_count"`
			LastUsed *time.Time `json:"api_key_last_used_at"`
		}
		must(json.Unmarshal(raw, &entries))
		for _, entry := range entries {
			if entry.ID == member.ID.String() {
				if entry.Count != count || entry.LastUsed == nil || !entry.LastUsed.Equal(latestUsed) {
					t.Fatalf("incorrect active key summary: %+v", entry)
				}
				return
			}
		}
		t.Fatal("member missing from user list")
	}
	checkUsage(2)
	request("DELETE", "/api/me/api-keys/"+memberKey.ID.String(), "", "", member, 204)
	// Revoked key activity must not supersede the most recently used active key.
	_, err = db.Pool.Exec(ctx, "UPDATE api_keys SET last_used_at=now() WHERE id=$1", memberKey.ID)
	must(err)
	checkUsage(1)
	var active []domain.APIKey
	must(json.Unmarshal(request("GET", "/api/me/api-keys", "", "", member, 200), &active))
	if len(active) != 1 {
		t.Fatalf("inactive key still listed: %d", len(active))
	}

	request("GET", "/api/assets", "", readSecret, nil, 401)
	_, expired := create(member, map[string]string{"assets": "read"})
	_, err = db.Pool.Exec(ctx, "UPDATE api_keys SET created_at=now()-interval '2 days',expires_at=now()-interval '1 day' WHERE secret_hash=$1", domain.HashAPIKey(expired))
	must(err)
	request("GET", "/api/assets", "", expired, nil, 401)
	checkUsage(1)
	must(json.Unmarshal(request("GET", "/api/me/api-keys", "", "", member, 200), &active))
	if len(active) != 1 {
		t.Fatal("expired key still listed")
	}
	must(users.Delete(ctx, admin.ID))
	request("GET", "/api/assets", "", secret, nil, 401)
	// Exercise rollback and reapply independently of the migration history table.
	down, err := os.ReadFile("../../migrations/000031_api_keys_mcp.down.sql")
	must(err)
	_, err = db.Pool.Exec(ctx, string(down))
	must(err)
	up, err := os.ReadFile("../../migrations/000031_api_keys_mcp.up.sql")
	must(err)
	_, err = db.Pool.Exec(ctx, string(up))
	must(err)
}

type keyFileStorage struct{}

func (*keyFileStorage) Upload(_ context.Context, _ string, _ string, body io.Reader) (string, error) {
	_, err := io.Copy(io.Discard, body)
	return "test-file", err
}
func (*keyFileStorage) GetPresignedURL(context.Context, string, time.Duration) (string, error) {
	return "https://storage.test/download", nil
}
func (*keyFileStorage) Delete(context.Context, string) error { return nil }
