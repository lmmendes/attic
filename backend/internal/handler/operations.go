package handler

import (
	"github.com/go-chi/chi/v5"
	"github.com/lmmendes/attic/internal/auth"
	"github.com/lmmendes/attic/internal/domain"
	"net/http"
	"strings"
)

// Operation is the single policy and dispatch record for an authenticated REST route.
// Empty ToolName means the operation is explicitly excluded from MCP.
type PermissionRequirement struct{ Resource, Action string }

type Operation struct {
	AdditionalPermissions                   []PermissionRequirement
	Method, Path, Resource, Action, Feature string
	Admin, Interactive                      bool
	ToolName, Description                   string
	Handler                                 http.HandlerFunc
	Body                                    any
	Query                                   []string
}

func Operations(h *Handler, pluginHandler *PluginHandler, userMgmtHandler *UserManagementHandler, authHandler *AuthHandler, keys *APIKeyHandler, version string) []Operation {
	operations := []Operation{
		{Method: "PUT", Path: "/api/auth/password", Handler: authHandler.ChangePassword, Resource: "", Action: "write", Feature: "", Admin: false, Interactive: true, ToolName: "", Description: "Change Password"},
		{Method: "GET", Path: "/api/me", Handler: h.GetCurrentUser, Resource: "", Action: "read", Feature: "", Admin: false, Interactive: false, ToolName: "", Description: "Get Current User"},
		{Method: "GET", Path: "/api/organization/features", Handler: h.GetOrganizationFeatures, Resource: "organization", Action: "read", Feature: "", Admin: false, Interactive: false, ToolName: "get_organization_features", Description: "Get Organization Features"},
		{Method: "PUT", Path: "/api/organization/features", Handler: h.UpdateOrganizationFeatures, Resource: "organization", Action: "write", Feature: "", Admin: true, Interactive: false, ToolName: "update_organization_features", Description: "Update Organization Features"},
		{Method: "GET", Path: "/api/users", Handler: userMgmtHandler.ListUsers, Resource: "users", Action: "read", Feature: "", Admin: true, Interactive: false, ToolName: "list_users", Description: "List Users"},
		{Method: "POST", Path: "/api/users", Handler: userMgmtHandler.CreateUser, Resource: "users", Action: "write", Feature: "", Admin: true, Interactive: true, ToolName: "", Description: "Create User"},
		{Method: "GET", Path: "/api/users/{id}", Handler: userMgmtHandler.GetUser, Resource: "users", Action: "read", Feature: "", Admin: true, Interactive: false, ToolName: "get_user", Description: "Get User"},
		{Method: "PUT", Path: "/api/users/{id}", Handler: userMgmtHandler.UpdateUser, Resource: "users", Action: "write", Feature: "", Admin: true, Interactive: false, ToolName: "update_user", Description: "Update User"},
		{Method: "DELETE", Path: "/api/users/{id}", Handler: userMgmtHandler.DeleteUser, Resource: "users", Action: "write", Feature: "", Admin: true, Interactive: false, ToolName: "delete_user", Description: "Delete User"},
		{Method: "POST", Path: "/api/users/{id}/reset-password", Handler: userMgmtHandler.ResetPassword, Resource: "users", Action: "write", Feature: "", Admin: true, Interactive: true, ToolName: "", Description: "Reset Password"},
		{Method: "GET", Path: "/api/collections", Handler: h.ListCollections, Resource: "collections", Action: "read", Feature: "collections", Admin: false, Interactive: false, ToolName: "list_collections", Description: "List Collections"},
		{Method: "POST", Path: "/api/collections", Handler: h.CreateCollection, Resource: "collections", Action: "write", Feature: "collections", Admin: false, Interactive: false, ToolName: "create_collection", Description: "Create Collection"},
		{Method: "GET", Path: "/api/collections/{id}", Handler: h.GetCollection, Resource: "collections", Action: "read", Feature: "collections", Admin: false, Interactive: false, ToolName: "get_collection", Description: "Get Collection"},
		{Method: "PUT", Path: "/api/collections/{id}", Handler: h.UpdateCollection, Resource: "collections", Action: "write", Feature: "collections", Admin: false, Interactive: false, ToolName: "update_collection", Description: "Update Collection"},
		{Method: "DELETE", Path: "/api/collections/{id}", Handler: h.DeleteCollection, Resource: "collections", Action: "write", Feature: "collections", Admin: false, Interactive: false, ToolName: "delete_collection", Description: "Delete Collection"},
		{Method: "GET", Path: "/api/categories", Handler: h.ListCategories, Resource: "categories", Action: "read", Feature: "categories", Admin: false, Interactive: false, ToolName: "list_categories", Description: "List Categories"},
		{Method: "POST", Path: "/api/categories", Handler: h.CreateCategory, Resource: "categories", Action: "write", Feature: "categories", Admin: false, Interactive: false, ToolName: "create_category", Description: "Create Category"},
		{Method: "GET", Path: "/api/categories/asset-counts", Handler: h.GetCategoryAssetCounts, Resource: "categories", Action: "read", Feature: "categories", Admin: false, Interactive: false, ToolName: "get_category_asset_counts", Description: "Get Category Asset Counts"},
		{Method: "GET", Path: "/api/categories/{id}", Handler: h.GetCategory, Resource: "categories", Action: "read", Feature: "categories", Admin: false, Interactive: false, ToolName: "get_category", Description: "Get Category"},
		{Method: "PUT", Path: "/api/categories/{id}", Handler: h.UpdateCategory, Resource: "categories", Action: "write", Feature: "categories", Admin: false, Interactive: false, ToolName: "update_category", Description: "Update Category"},
		{Method: "DELETE", Path: "/api/categories/{id}", Handler: h.DeleteCategory, Resource: "categories", Action: "write", Feature: "categories", Admin: false, Interactive: false, ToolName: "delete_category", Description: "Delete Category"},
		{Method: "GET", Path: "/api/attributes", Handler: h.ListAttributes, Resource: "attributes", Action: "read", Feature: "attributes", Admin: false, Interactive: false, ToolName: "list_attributes", Description: "List Attributes"},
		{Method: "POST", Path: "/api/attributes", Handler: h.CreateAttribute, Resource: "attributes", Action: "write", Feature: "attributes", Admin: false, Interactive: false, ToolName: "create_attribute", Description: "Create Attribute"},
		{Method: "GET", Path: "/api/attributes/{id}", Handler: h.GetAttribute, Resource: "attributes", Action: "read", Feature: "attributes", Admin: false, Interactive: false, ToolName: "get_attribute", Description: "Get Attribute"},
		{Method: "PUT", Path: "/api/attributes/{id}", Handler: h.UpdateAttribute, Resource: "attributes", Action: "write", Feature: "attributes", Admin: false, Interactive: false, ToolName: "update_attribute", Description: "Update Attribute"},
		{Method: "DELETE", Path: "/api/attributes/{id}", Handler: h.DeleteAttribute, Resource: "attributes", Action: "write", Feature: "attributes", Admin: false, Interactive: false, ToolName: "delete_attribute", Description: "Delete Attribute"},
		{Method: "POST", Path: "/api/attributes/{id}/impact-preview", Handler: h.PreviewAttributeImpact, Resource: "attributes", Action: "read", Feature: "attributes", Admin: true, Interactive: false, ToolName: "preview_attribute_impact", Description: "Preview Attribute Impact"},
		{Method: "GET", Path: "/api/attributes/{id}/options", Handler: h.ListAttributeOptions, Resource: "attributes", Action: "read", Feature: "attributes", Admin: false, Interactive: false, ToolName: "list_attribute_options", Description: "List Attribute Options"},
		{Method: "POST", Path: "/api/attributes/{id}/options", Handler: h.AddAttributeOption, Resource: "attributes", Action: "write", Feature: "attributes", Admin: false, Interactive: false, ToolName: "add_attribute_option", Description: "Add Attribute Option"},
		{Method: "PATCH", Path: "/api/attributes/{id}/options/{option_id}", Handler: h.UpdateAttributeOption, Resource: "attributes", Action: "write", Feature: "attributes", Admin: false, Interactive: false, ToolName: "update_attribute_option", Description: "Update Attribute Option"},
		{Method: "DELETE", Path: "/api/attributes/{id}/options/{option_id}", Handler: h.DeleteAttributeOption, Resource: "attributes", Action: "write", Feature: "attributes", Admin: false, Interactive: false, ToolName: "delete_attribute_option", Description: "Delete Attribute Option"},
		{Method: "PUT", Path: "/api/attributes/{id}/options/order", Handler: h.OrderAttributeOptions, Resource: "attributes", Action: "write", Feature: "attributes", Admin: false, Interactive: false, ToolName: "order_attribute_options", Description: "Order Attribute Options"},
		{Method: "GET", Path: "/api/locations", Handler: h.ListLocations, Resource: "locations", Action: "read", Feature: "locations", Admin: false, Interactive: false, ToolName: "list_locations", Description: "List Locations"},
		{Method: "POST", Path: "/api/locations", Handler: h.CreateLocation, Resource: "locations", Action: "write", Feature: "locations", Admin: false, Interactive: false, ToolName: "create_location", Description: "Create Location"},
		{Method: "GET", Path: "/api/locations/{id}", Handler: h.GetLocation, Resource: "locations", Action: "read", Feature: "locations", Admin: false, Interactive: false, ToolName: "get_location", Description: "Get Location"},
		{Method: "PUT", Path: "/api/locations/{id}", Handler: h.UpdateLocation, Resource: "locations", Action: "write", Feature: "locations", Admin: false, Interactive: false, ToolName: "update_location", Description: "Update Location"},
		{Method: "DELETE", Path: "/api/locations/{id}", Handler: h.DeleteLocation, Resource: "locations", Action: "write", Feature: "locations", Admin: false, Interactive: false, ToolName: "delete_location", Description: "Delete Location"},
		{Method: "GET", Path: "/api/conditions", Handler: h.ListConditions, Resource: "conditions", Action: "read", Feature: "conditions", Admin: false, Interactive: false, ToolName: "list_conditions", Description: "List Conditions"},
		{Method: "POST", Path: "/api/conditions", Handler: h.CreateCondition, Resource: "conditions", Action: "write", Feature: "conditions", Admin: false, Interactive: false, ToolName: "create_condition", Description: "Create Condition"},
		{Method: "GET", Path: "/api/conditions/{id}", Handler: h.GetCondition, Resource: "conditions", Action: "read", Feature: "conditions", Admin: false, Interactive: false, ToolName: "get_condition", Description: "Get Condition"},
		{Method: "PUT", Path: "/api/conditions/{id}", Handler: h.UpdateCondition, Resource: "conditions", Action: "write", Feature: "conditions", Admin: false, Interactive: false, ToolName: "update_condition", Description: "Update Condition"},
		{Method: "DELETE", Path: "/api/conditions/{id}", Handler: h.DeleteCondition, Resource: "conditions", Action: "write", Feature: "conditions", Admin: false, Interactive: false, ToolName: "delete_condition", Description: "Delete Condition"},
		{Method: "GET", Path: "/api/saved-filters", Handler: h.ListSavedFilters, Resource: "saved_filters", Action: "read", Feature: "", Admin: false, Interactive: false, ToolName: "list_saved_filters", Description: "List Saved Filters"},
		{Method: "POST", Path: "/api/saved-filters", Handler: h.CreateSavedFilter, Resource: "saved_filters", Action: "write", Feature: "", Admin: false, Interactive: false, ToolName: "create_saved_filter", Description: "Create Saved Filter"},
		{Method: "GET", Path: "/api/saved-filters/{id}", Handler: h.GetSavedFilter, Resource: "saved_filters", Action: "read", Feature: "", Admin: false, Interactive: false, ToolName: "get_saved_filter", Description: "Get Saved Filter"},
		{Method: "PUT", Path: "/api/saved-filters/{id}", Handler: h.UpdateSavedFilter, Resource: "saved_filters", Action: "write", Feature: "", Admin: false, Interactive: false, ToolName: "update_saved_filter", Description: "Update Saved Filter"},
		{Method: "DELETE", Path: "/api/saved-filters/{id}", Handler: h.DeleteSavedFilter, Resource: "saved_filters", Action: "write", Feature: "", Admin: false, Interactive: false, ToolName: "delete_saved_filter", Description: "Delete Saved Filter"},
		{Method: "GET", Path: "/api/tags", Handler: h.ListTags, Resource: "tags", Action: "read", Feature: "tags", Admin: false, Interactive: false, ToolName: "list_tags", Description: "List Tags"},
		{Method: "POST", Path: "/api/tags", Handler: h.CreateTag, Resource: "tags", Action: "write", Feature: "tags", Admin: false, Interactive: false, ToolName: "create_tag", Description: "Create Tag"},
		{Method: "GET", Path: "/api/tags/{id}", Handler: h.GetTag, Resource: "tags", Action: "read", Feature: "tags", Admin: false, Interactive: false, ToolName: "get_tag", Description: "Get Tag"},
		{Method: "PUT", Path: "/api/tags/{id}", Handler: h.UpdateTag, Resource: "tags", Action: "write", Feature: "tags", Admin: false, Interactive: false, ToolName: "update_tag", Description: "Update Tag"},
		{Method: "DELETE", Path: "/api/tags/{id}", Handler: h.DeleteTag, Resource: "tags", Action: "write", Feature: "tags", Admin: false, Interactive: false, ToolName: "delete_tag", Description: "Delete Tag"},
		{Method: "GET", Path: "/api/assets", Handler: h.ListAssets, Resource: "assets", Action: "read", Feature: "", Admin: false, Interactive: false, ToolName: "list_assets", Description: "List Assets"},
		{Method: "GET", Path: "/api/assets/stats", Handler: h.GetAssetStats, Resource: "assets", Action: "read", Feature: "", Admin: false, Interactive: false, ToolName: "get_asset_stats", Description: "Get Asset Stats"},
		{Method: "POST", Path: "/api/assets/search", Handler: h.SearchAssets, Resource: "assets", Action: "read", Feature: "", Admin: false, Interactive: false, ToolName: "search_assets", Description: "Search Assets"},
		{Method: "POST", Path: "/api/assets", Handler: h.CreateAsset, Resource: "assets", Action: "write", Feature: "", Admin: false, Interactive: false, ToolName: "create_asset", Description: "Create Asset"},
		{Method: "GET", Path: "/api/assets/{id}", Handler: h.GetAsset, Resource: "assets", Action: "read", Feature: "", Admin: false, Interactive: false, ToolName: "get_asset", Description: "Get Asset"},
		{Method: "PUT", Path: "/api/assets/{id}", Handler: h.UpdateAsset, Resource: "assets", Action: "write", Feature: "", Admin: false, Interactive: false, ToolName: "update_asset", Description: "Update Asset"},
		{Method: "DELETE", Path: "/api/assets/{id}", Handler: h.DeleteAsset, Resource: "assets", Action: "write", Feature: "", Admin: false, Interactive: false, ToolName: "delete_asset", Description: "Delete Asset"},
		{Method: "GET", Path: "/api/assets/{id}/warranty", Handler: h.GetWarranty, Resource: "warranties", Action: "read", Feature: "warranties", Admin: false, Interactive: false, ToolName: "get_warranty", Description: "Get Warranty"},
		{Method: "POST", Path: "/api/assets/{id}/warranty", Handler: h.CreateWarranty, Resource: "warranties", Action: "write", Feature: "warranties", Admin: false, Interactive: false, ToolName: "create_warranty", Description: "Create Warranty"},
		{Method: "PUT", Path: "/api/assets/{id}/warranty", Handler: h.UpdateWarranty, Resource: "warranties", Action: "write", Feature: "warranties", Admin: false, Interactive: false, ToolName: "update_warranty", Description: "Update Warranty"},
		{Method: "DELETE", Path: "/api/assets/{id}/warranty", Handler: h.DeleteWarranty, Resource: "warranties", Action: "write", Feature: "warranties", Admin: false, Interactive: false, ToolName: "delete_warranty", Description: "Delete Warranty"},
		{Method: "GET", Path: "/api/assets/{id}/attachments", Handler: h.ListAttachments, Resource: "attachments", Action: "read", Feature: "", Admin: false, Interactive: false, ToolName: "list_attachments", Description: "List Attachments"},
		{Method: "POST", Path: "/api/assets/{id}/attachments", Handler: h.UploadAttachment, Resource: "attachments", Action: "write", Feature: "", Admin: false, Interactive: false, ToolName: "upload_attachment", Description: "Upload Attachment"},
		{Method: "GET", Path: "/api/assets/{id}/events", Handler: h.ListAssetEvents, Resource: "asset_events", Action: "read", Feature: "", Admin: false, Interactive: false, ToolName: "list_asset_events", Description: "List Asset Events"},
		{Method: "POST", Path: "/api/assets/{id}/events", Handler: h.CreateAssetEvent, Resource: "asset_events", Action: "write", Feature: "", Admin: false, Interactive: false, ToolName: "create_asset_event", Description: "Create Asset Event"},
		{Method: "PUT", Path: "/api/assets/{id}/events/{eventId}", Handler: h.UpdateAssetEvent, Resource: "asset_events", Action: "write", Feature: "", Admin: false, Interactive: false, ToolName: "update_asset_event", Description: "Update Asset Event"},
		{Method: "DELETE", Path: "/api/assets/{id}/events/{eventId}", Handler: h.DeleteAssetEvent, Resource: "asset_events", Action: "write", Feature: "", Admin: false, Interactive: false, ToolName: "delete_asset_event", Description: "Delete Asset Event"},
		{Method: "PUT", Path: "/api/assets/{id}/main-image/{attachmentId}", Handler: h.SetMainAttachment, Resource: "attachments", Action: "write", Feature: "", Admin: false, Interactive: false, ToolName: "set_main_attachment", Description: "Set Main Attachment"},
		{Method: "DELETE", Path: "/api/assets/{id}/main-image", Handler: h.ClearMainAttachment, Resource: "attachments", Action: "write", Feature: "", Admin: false, Interactive: false, ToolName: "clear_main_attachment", Description: "Clear Main Attachment"},
		{Method: "GET", Path: "/api/attachments/{attachmentId}", Handler: h.GetAttachment, Resource: "attachments", Action: "read", Feature: "", Admin: false, Interactive: false, ToolName: "get_attachment", Description: "Get Attachment"},
		{Method: "DELETE", Path: "/api/attachments/{attachmentId}", Handler: h.DeleteAttachment, Resource: "attachments", Action: "write", Feature: "", Admin: false, Interactive: false, ToolName: "delete_attachment", Description: "Delete Attachment"},
		{Method: "GET", Path: "/api/warranties", Handler: h.ListWarranties, Resource: "warranties", Action: "read", Feature: "warranties", Admin: false, Interactive: false, ToolName: "list_warranties", Description: "List Warranties"},
		{Method: "GET", Path: "/api/warranties/expiring", Handler: h.ListExpiringWarranties, Resource: "warranties", Action: "read", Feature: "warranties", Admin: false, Interactive: false, ToolName: "list_expiring_warranties", Description: "List Expiring Warranties"},
		{Method: "GET", Path: "/api/plugins", Handler: pluginHandler.ListPlugins, Resource: "plugins", Action: "read", Feature: "plugins", Admin: false, Interactive: false, ToolName: "list_plugins", Description: "List Plugins"},
		{Method: "GET", Path: "/api/plugins/{pluginId}", Handler: pluginHandler.GetPlugin, Resource: "plugins", Action: "read", Feature: "plugins", Admin: false, Interactive: false, ToolName: "get_plugin", Description: "Get Plugin"},
		{Method: "GET", Path: "/api/plugins/{pluginId}/search", Handler: pluginHandler.Search, Resource: "plugins", Action: "read", Feature: "plugins", Admin: false, Interactive: false, ToolName: "search", Description: "Search an external import plugin catalog"},
		{Method: "POST", Path: "/api/plugins/{pluginId}/import", Handler: pluginHandler.Import, Resource: "plugins", Action: "write", Feature: "plugins", Admin: false, Interactive: false, ToolName: "import", Description: "Import an asset from an external plugin", AdditionalPermissions: []PermissionRequirement{{Resource: "assets", Action: "write"}}},
	}
	operations = append(operations,
		Operation{Method: "GET", Path: "/api/", Handler: func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, 200, map[string]string{"status": "ok", "version": version})
		}},
		Operation{Method: "GET", Path: "/api/me/api-keys", Handler: keys.List, Interactive: true},
		Operation{Method: "GET", Path: "/api/me/api-keys/permissions", Handler: keys.Catalog, Interactive: true},
		Operation{Method: "POST", Path: "/api/me/api-keys", Handler: keys.Create, Interactive: true},
		Operation{Method: "PATCH", Path: "/api/me/api-keys/{id}", Handler: keys.Rename, Interactive: true},
		Operation{Method: "DELETE", Path: "/api/me/api-keys/{id}", Handler: keys.Revoke, Interactive: true},
		Operation{Method: "GET", Path: "/api/organization/mcp", Handler: keys.GetMCP, Resource: "organization", Action: "read", ToolName: "get_mcp_status", Description: "Get MCP status"},
		Operation{Method: "PUT", Path: "/api/organization/mcp", Handler: keys.UpdateMCP, Resource: "organization", Action: "write", Admin: true, ToolName: "update_mcp_status", Description: "Enable or disable MCP"},
	)
	for i := range operations {
		operations[i].Body, operations[i].Query = operationInput(operations[i].ToolName)
	}
	return operations
}

func (h *Handler) RegisterOperations(r chi.Router, operations []Operation) {
	for _, op := range operations {
		for _, path := range operationPaths(op) {
			r.Method(op.Method, path, h.AuthorizeOperation(op, op.Handler))
		}
	}
}

func (h *Handler) AuthorizeOperation(op Operation, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := auth.GetUser(r.Context())
		if user == nil || user.OrganizationID != h.orgID {
			writeError(w, 401, "authentication required")
			return
		}
		key := auth.GetAPIKey(r.Context())
		if key != nil {
			if op.Interactive {
				writeError(w, 403, "interactive account authentication required")
				return
			}
			if !op.keyAllows(key, user) {
				writeError(w, 403, "API key permission denied")
				return
			}
		}
		if op.Admin && user.Role != "admin" {
			writeError(w, 403, "admin access required")
			return
		}
		if op.Feature != "" {
			enabled, err := h.featureEnabled(r, op.Feature)
			if err != nil {
				writeError(w, 500, "failed to read organization features")
				return
			}
			if !enabled {
				writeError(w, 404, "feature is not enabled")
				return
			}
		}
		if key != nil && !h.checkOperationOrganization(w, r, op) {
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (op Operation) keyAllows(key *domain.APIKey, user *domain.User) bool {
	if op.Resource != "" && !key.Allows(user, op.Resource, op.Action) {
		return false
	}
	for _, requirement := range op.AdditionalPermissions {
		if !key.Allows(user, requirement.Resource, requirement.Action) {
			return false
		}
	}
	return true
}

// Preserve the collection-root aliases previously provided by chi subrouters.
func operationPaths(op Operation) []string {
	paths := []string{op.Path}
	if op.Path == "/api/" {
		return append(paths, "/api")
	}
	if strings.Count(op.Path, "/") == 2 && op.Resource != "" {
		paths = append(paths, op.Path+"/")
	}
	return paths
}
