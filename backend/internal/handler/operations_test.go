package handler

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/jsonschema-go/jsonschema"
	"github.com/google/uuid"
	"github.com/lmmendes/attic/internal/auth"
	"github.com/lmmendes/attic/internal/domain"
)

type policyKeys struct{ key *domain.APIKey }

func (s policyKeys) Authenticate(context.Context, string) (*domain.APIKey, error) { return s.key, nil }

type policyUsers struct{ user *domain.User }

func (s policyUsers) GetByID(context.Context, uuid.UUID) (*domain.User, error) { return s.user, nil }

func TestOperationRegistryCoverageAndScopeMatrix(t *testing.T) {
	h := New(nil, nil, nil, uuid.New())
	operations := Operations(h, &PluginHandler{}, &UserManagementHandler{}, &AuthHandler{}, &APIKeyHandler{}, "test")
	router := chi.NewRouter()
	h.RegisterOperations(router, operations)
	byRoute := map[string]Operation{}
	byTool := map[string]bool{}
	for _, op := range operations {
		route := op.Method + " " + op.Path
		if _, exists := byRoute[route]; exists {
			t.Fatal("duplicate route", route)
		}
		for _, path := range operationPaths(op) {
			byRoute[op.Method+" "+path] = op
		}
		if op.Handler == nil {
			t.Fatal("missing handler", route)
		}
		if op.Resource == "" && !op.Interactive && op.Path != "/api/" && op.Path != "/api/me" {
			t.Fatal("undeclared permission policy", route)
		}
		if op.ToolName == "" && !op.Interactive && op.Path != "/api/" && op.Path != "/api/me" {
			t.Fatal("undeclared MCP exclusion", route)
		}
		if op.Interactive && op.ToolName != "" {
			t.Fatal("credential operation exposed as MCP tool", route)
		}
		if op.ToolName != "" {
			if byTool[op.ToolName] {
				t.Fatal("duplicate tool", op.ToolName)
			}
			byTool[op.ToolName] = true
			raw, err := json.Marshal(operationSchema(op))
			if err != nil {
				t.Fatal(err)
			}
			var schema jsonschema.Schema
			if err = json.Unmarshal(raw, &schema); err != nil {
				t.Fatal(err)
			}
			if _, err = schema.Resolve(nil); err != nil {
				t.Fatal(op.ToolName, err)
			}
			if op.Action == "write" && (op.Method == "POST" || op.Method == "PUT" || op.Method == "PATCH") && op.Body == nil && !strings.Contains(op.Path, "main-image") {
				t.Fatal("missing write input schema", route)
			}
		}
	}
	count := 0
	if err := chi.Walk(router, func(method, path string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		count++
		if _, exists := byRoute[method+" "+path]; !exists {
			t.Fatal("unregistered route", method, path)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if count != len(byRoute) {
		t.Fatal("route registration mismatch")
	}
	for _, resource := range domain.PermissionCatalog(domain.UserRoleAdmin) {
		for _, role := range []domain.UserRole{domain.UserRoleUser, domain.UserRoleAdmin} {
			for _, level := range []string{"none", "read", "write"} {
				for _, action := range []string{"read", "write"} {
					user := &domain.User{ID: uuid.New(), OrganizationID: h.orgID, Role: role}
					key := &domain.APIKey{OwnerID: user.ID, OrganizationID: user.OrganizationID, Permissions: map[string]string{resource.Resource: level}}
					authentication := auth.APIKeyAuthenticator{Keys: policyKeys{key}, Users: policyUsers{user}, OrganizationID: h.orgID}
					op := Operation{Resource: resource.Resource, Action: action, Path: "/api/test"}
					next := h.AuthorizeOperation(op, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(204) }))
					request := httptest.NewRequest("GET", "/api/test", nil)
					request.Header.Set("Authorization", "Bearer test")
					response := httptest.NewRecorder()
					authentication.Authenticate(func(next http.Handler) http.Handler { return next })(next).ServeHTTP(response, request)
					want := 403
					if key.Allows(user, resource.Resource, action) {
						want = 204
					}
					if response.Code != want {
						t.Fatalf("%s %s %s %s: got %d want %d", resource.Resource, role, level, action, response.Code, want)
					}
				}
			}
		}
	}
}

func TestMCPAttachmentInputValidation(t *testing.T) {
	op := Operation{Method: "POST", Path: "/api/assets/{id}/attachments", ToolName: "upload_attachment", Body: struct{}{}}
	id := uuid.New().String()
	for _, body := range []string{`{"filename":"a.txt","content_type":"text/plain","content":"%%%"}`, `{"filename":"","content_type":"text/plain","content":""}`} {
		_, err := operationRequest(context.Background(), op, json.RawMessage(`{"id":"`+id+`","body":`+body+`}`))
		if err == nil {
			t.Fatal("accepted invalid attachment", body)
		}
	}
}

type enabledMCP struct{}

func (enabledMCP) MCPEnabled(context.Context, uuid.UUID) (bool, error) { return true, nil }

func TestMCPDiscoveryAndInvocationScopeMatrix(t *testing.T) {
	h := handlerWithFeatureRepository(&mockOrganizationFeatureRepository{features: allFeaturesEnabled()})
	for _, role := range []domain.UserRole{domain.UserRoleUser, domain.UserRoleAdmin} {
		for _, resource := range domain.PermissionCatalog(domain.UserRoleAdmin) {
			for _, grant := range []string{"none", "read", "write"} {
				user := &domain.User{ID: uuid.New(), OrganizationID: h.orgID, Role: role}
				key := &domain.APIKey{OwnerID: user.ID, OrganizationID: h.orgID, Permissions: map[string]string{resource.Resource: grant}}
				authenticator := auth.APIKeyAuthenticator{Keys: policyKeys{key}, Users: policyUsers{user}, OrganizationID: h.orgID}
				operations := []Operation{}
				for _, action := range []string{"read", "write"} {
					operations = append(operations, Operation{Method: "GET", Path: "/api/policy", Resource: resource.Resource, Action: action, ToolName: action, Handler: func(w http.ResponseWriter, _ *http.Request) { writeJSON(w, 200, map[string]bool{"ok": true}) }})
				}
				endpoint := authenticator.Authenticate(func(next http.Handler) http.Handler { return next })(h.MCPHandler(operations, enabledMCP{}))
				call := func(method string, params any) map[string]any {
					t.Helper()
					body, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": method, "params": params})
					request := httptest.NewRequest("POST", "/mcp", strings.NewReader(string(body)))
					request.Header.Set("Authorization", "Bearer key")
					request.Header.Set("Content-Type", "application/json")
					request.Header.Set("Accept", "application/json, text/event-stream")
					response := httptest.NewRecorder()
					endpoint.ServeHTTP(response, request)
					if response.Code != 200 {
						t.Fatalf("%s: %d %s", method, response.Code, response.Body)
					}
					var envelope map[string]any
					if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
						t.Fatal(err)
					}
					return envelope
				}
				envelope := call("tools/list", map[string]any{})
				result := envelope["result"].(map[string]any)
				names := map[string]bool{}
				if tools, ok := result["tools"].([]any); ok {
					for _, tool := range tools {
						names[tool.(map[string]any)["name"].(string)] = true
					}
				}
				for _, action := range []string{"read", "write"} {
					allowed := key.Allows(user, resource.Resource, action)
					if names[action] != allowed {
						t.Fatalf("discovery: %s %s %s %s", role, resource.Resource, grant, action)
					}
					envelope = call("tools/call", map[string]any{"name": action, "arguments": map[string]any{}})
					result, _ = envelope["result"].(map[string]any)
					failed := envelope["error"] != nil || result["isError"] == true
					if failed == allowed {
						t.Fatalf("invocation: %s %s %s %s: %v", role, resource.Resource, grant, action, envelope)
					}
				}
			}
		}
	}
}

func TestAttachmentDecodedLimitWithBase64Padding(t *testing.T) {
	for _, size := range []int{4, 5, 6} {
		content := base64.StdEncoding.EncodeToString(make([]byte, size))
		decoded, err := decodeAttachmentContent(content, 5)
		if size <= 5 {
			if err != nil || len(decoded) != size {
				t.Fatalf("valid padded file of %d bytes rejected: %v", size, err)
			}
		} else if err == nil {
			t.Fatal("oversize decoded content accepted")
		}
	}
}
