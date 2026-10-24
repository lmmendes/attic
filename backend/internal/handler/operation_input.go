package handler

import (
	"encoding/json"
	"reflect"
	"strings"

	"github.com/google/uuid"
	"github.com/lmmendes/attic/internal/domain"
	"github.com/lmmendes/attic/internal/repository"
)

// Request types are shared with REST so tool schemas follow the actual contract.
func operationInput(name string) (any, []string) {
	switch name {
	case "create_asset":
		return CreateAssetRequest{}, nil
	case "update_asset":
		return UpdateAssetRequest{}, nil
	case "search_assets":
		return assetSearchRequest{}, nil
	case "create_category":
		return CreateCategoryRequest{}, nil
	case "update_category":
		return UpdateCategoryRequest{}, nil
	case "create_location":
		return CreateLocationRequest{}, nil
	case "update_location":
		return UpdateLocationRequest{}, nil
	case "create_collection", "update_collection":
		return CollectionRequest{}, nil
	case "create_tag", "update_tag":
		return tagRequest{}, nil
	case "create_condition":
		return CreateConditionRequest{}, nil
	case "update_condition":
		return UpdateConditionRequest{}, nil
	case "create_attribute":
		return CreateAttributeRequest{}, nil
	case "update_attribute":
		return repository.AttributeChange{}, nil
	case "preview_attribute_impact":
		return repository.AttributeAction{}, nil
	case "update_attribute_option":
		return repository.AttributeChange{}, nil
	case "add_attribute_option":
		return domain.AttributeOption{}, nil
	case "order_attribute_options":
		return struct {
			IDs []uuid.UUID `json:"ids"`
		}{}, nil
	case "create_warranty":
		return CreateWarrantyRequest{}, nil
	case "update_warranty":
		return UpdateWarrantyRequest{}, nil
	case "create_asset_event", "update_asset_event":
		return AssetEventRequest{}, nil
	case "create_saved_filter", "update_saved_filter":
		return savedFilterRequest{}, nil
	case "import":
		return ImportRequest{}, nil
	case "update_user":
		return UpdateUserRequest{}, nil
	case "update_organization_features":
		return domain.OrganizationFeatures{}, nil
	case "update_mcp_status":
		return struct {
			Enabled bool `json:"enabled"`
		}{}, nil
	case "upload_attachment":
		return struct {
			Filename    string `json:"filename"`
			ContentType string `json:"content_type"`
			Description string `json:"description,omitempty"`
			Content     string `json:"content"`
		}{}, nil
	case "list_assets":
		return nil, []string{"q", "attribute_q", "category_id", "location_id", "condition_id", "collection_id", "tag_id", "tag_match", "limit", "offset"}
	case "get_asset_stats":
		return nil, []string{"location_id"}
	case "list_categories", "list_locations":
		return nil, []string{"tree"}
	case "get_category":
		return nil, []string{"inherited"}
	case "list_expiring_warranties":
		return nil, []string{"days"}
	case "search":
		return nil, []string{"field", "q", "limit"}
	}
	return nil, nil
}

func operationSchema(op Operation) map[string]any {
	definitions := map[string]any{}
	properties := map[string]any{}
	required := []string{}
	for _, part := range strings.Split(op.Path, "/") {
		if strings.HasPrefix(part, "{") {
			name := strings.Trim(part, "{}")
			properties[name] = map[string]any{"type": "string"}
			if name != "pluginId" {
				properties[name].(map[string]any)["format"] = "uuid"
			}
			required = append(required, name)
		}
	}
	if op.Body != nil {
		properties["body"] = typeSchema(reflect.TypeOf(op.Body), definitions)
		required = append(required, "body")
	}
	if len(op.Query) > 0 {
		query := map[string]any{}
		for _, key := range op.Query {
			if key == "tag_id" {
				query[key] = map[string]any{"type": "array", "items": map[string]any{"type": "string"}}
			} else {
				query[key] = map[string]any{"type": "string"}
			}
		}
		properties["query"] = map[string]any{"type": "object", "properties": query, "additionalProperties": false}
	}
	if op.Resource == "attributes" && op.Action == "write" {
		properties["impact_token"] = map[string]any{"type": "string", "description": "Confirmation token returned by preview_attribute_impact for this exact change."}
	}
	schema := map[string]any{"type": "object", "properties": properties, "additionalProperties": false}
	if len(required) > 0 {
		schema["required"] = required
	}
	if len(definitions) > 0 {
		schema["$defs"] = definitions
	}
	return schema
}
func typeSchema(t reflect.Type, defs map[string]any) map[string]any {
	if t.Kind() == reflect.Pointer {
		return map[string]any{"anyOf": []any{typeSchema(t.Elem(), defs), map[string]any{"type": "null"}}}
	}
	if t == reflect.TypeOf(uuid.UUID{}) {
		return map[string]any{"type": "string", "format": "uuid"}
	}
	if t == reflect.TypeOf(json.RawMessage{}) {
		return map[string]any{}
	}
	switch t.Kind() {
	case reflect.String:
		return map[string]any{"type": "string"}
	case reflect.Bool:
		return map[string]any{"type": "boolean"}
	case reflect.Int, reflect.Int32, reflect.Int64, reflect.Uint:
		return map[string]any{"type": "integer"}
	case reflect.Float32, reflect.Float64:
		return map[string]any{"type": "number"}
	case reflect.Slice, reflect.Array:
		return map[string]any{"type": "array", "items": typeSchema(t.Elem(), defs)}
	case reflect.Map:
		return map[string]any{"type": "object", "additionalProperties": typeSchema(t.Elem(), defs)}
	case reflect.Struct:
		name := t.Name()
		if name != "" {
			if _, ok := defs[name]; ok {
				return map[string]any{"$ref": "#/$defs/" + name}
			}
			defs[name] = map[string]any{}
		}
		props := map[string]any{}
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			tag := field.Tag.Get("json")
			if tag == "-" || field.PkgPath != "" {
				continue
			}
			key := strings.Split(tag, ",")[0]
			if key == "" {
				key = field.Name
			}
			props[key] = typeSchema(field.Type, defs)
		}
		result := map[string]any{"type": "object", "properties": props, "additionalProperties": false}

		if name != "" {
			defs[name] = result
			return map[string]any{"$ref": "#/$defs/" + name}
		}
		return result
	}
	return map[string]any{}
}
