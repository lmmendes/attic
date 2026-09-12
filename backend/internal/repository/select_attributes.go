package repository

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/lmmendes/attic/internal/domain"
)

type AttributeChange struct {
	Name          *string                   `json:"name,omitempty"`
	Key           *string                   `json:"key,omitempty"`
	DataType      *domain.AttributeDataType `json:"data_type,omitempty"`
	SelectionMode *string                   `json:"selection_mode,omitempty"`
	Options       *[]domain.AttributeOption `json:"options,omitempty"`
	Label         *string                   `json:"label,omitempty"`
	Value         *string                   `json:"value,omitempty"`
}
type AttributeAction struct {
	Action   string          `json:"action"`
	OptionID *uuid.UUID      `json:"option_id,omitempty"`
	Changes  AttributeChange `json:"changes"`
}
type AttributeImpact struct {
	AffectedAssets          int    `json:"affected_assets"`
	DeletedAssets           int    `json:"deleted_assets"`
	RequiredFieldsLeftEmpty int    `json:"required_fields_left_empty"`
	Current                 string `json:"current"`
	Proposed                string `json:"proposed"`
	ConfirmationToken       string `json:"confirmation_token"`
}
type AttributeError struct {
	Status        int
	Code, Message string
}

func (e *AttributeError) Error() string { return e.Message }

type impactedAsset struct {
	ID         uuid.UUID
	Attributes json.RawMessage
	Deleted    bool
	Required   bool
}
type optionReplacement struct {
	value   string
	changed bool
}

func lockAttributeWrites(ctx context.Context, tx pgx.Tx, org uuid.UUID) error {
	_, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock(hashtextextended($1, 619))", org.String())
	return err
}
func lockAttributeReads(ctx context.Context, tx pgx.Tx, org uuid.UUID) error {
	_, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock_shared(hashtextextended($1, 619))", org.String())
	return err
}
func invalidAttribute(message string) error {
	return &AttributeError{400, "invalid_attribute", message}
}
func conflictAttribute(message string) error {
	return &AttributeError{409, "attribute_conflict", message}
}
func validateDefinition(a *domain.Attribute) error {
	a.Name = strings.TrimSpace(a.Name)
	a.Key = strings.TrimSpace(a.Key)
	if a.Name == "" || utf8.RuneCountInString(a.Name) > 255 || a.Key == "" || utf8.RuneCountInString(a.Key) > 100 {
		return invalidAttribute("field name and key are required and must fit their length limits")
	}
	if a.PluginID == nil && strings.HasPrefix(a.Key, "plugin.") {
		return invalidAttribute("attribute keys beginning with plugin. are reserved for plugins")
	}
	switch a.DataType {
	case domain.AttributeTypeSelect:
		if a.SelectionMode == "" {
			a.SelectionMode = "single"
		}
		if len(a.Options) == 0 {
			return invalidAttribute("select fields require at least one option")
		}
		if a.SelectionMode != "single" && a.SelectionMode != "multiple" {
			return invalidAttribute("selection_mode must be single or multiple")
		}
	case domain.AttributeTypeString, domain.AttributeTypeNumber, domain.AttributeTypeBoolean, domain.AttributeTypeDate, domain.AttributeTypeText:
		if a.SelectionMode != "" || a.Options != nil {
			return invalidAttribute("options and selection_mode require a select field")
		}
	default:
		return invalidAttribute("invalid data_type")
	}
	labels, values := map[string]bool{}, map[string]bool{}
	for i := range a.Options {
		o := &a.Options[i]
		o.Label = strings.TrimSpace(o.Label)
		o.Value = strings.TrimSpace(o.Value)
		if o.Label == "" || o.Value == "" || utf8.RuneCountInString(o.Label) > 255 || utf8.RuneCountInString(o.Value) > 255 {
			return invalidAttribute("option label and value must contain 1 to 255 characters")
		}
		if labels[strings.ToLower(o.Label)] || values[o.Value] {
			return conflictAttribute("option labels and values must be unique within the field")
		}
		labels[strings.ToLower(o.Label)] = true
		values[o.Value] = true
	}
	return nil
}
func (r *AttributeRepository) createConfigured(ctx context.Context, a *domain.Attribute, query string) error {
	if err := validateDefinition(a); err != nil {
		return err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err = lockAttributeWrites(ctx, tx, a.OrganizationID); err != nil {
		return err
	}
	var retired bool
	if err = tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM retired_attribute_keys WHERE organization_id=$1 AND key=$2)", a.OrganizationID, a.Key).Scan(&retired); err != nil {
		return err
	}
	if retired {
		return conflictAttribute("this key was retired; choose a new field key")
	}
	if err = tx.QueryRow(ctx, query, a.ID, a.OrganizationID, a.PluginID, a.Name, a.Key, a.DataType, a.SelectionMode).Scan(&a.CreatedAt, &a.UpdatedAt); err != nil {
		return err
	}
	for i := range a.Options {
		o := &a.Options[i]
		o.ID = uuid.New()
		o.SortOrder = i
		if _, err = tx.Exec(ctx, "INSERT INTO attribute_options(id,attribute_id,label,value,sort_order) VALUES($1,$2,$3,$4,$5)", o.ID, a.ID, o.Label, o.Value, o.SortOrder); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}
func loadManagedAttribute(ctx context.Context, tx pgx.Tx, org, id uuid.UUID) (*domain.Attribute, error) {
	var a domain.Attribute
	err := tx.QueryRow(ctx, `SELECT id,organization_id,plugin_id,name,key,data_type,created_at,updated_at,COALESCE(selection_mode,''),
 COALESCE((SELECT jsonb_agg(jsonb_build_object('id',o.id,'label',o.label,'value',o.value,'sort_order',o.sort_order) ORDER BY o.sort_order,o.id)
 FROM attribute_options o WHERE o.attribute_id=attributes.id),'[]'::jsonb)
 FROM attributes WHERE id=$1 AND organization_id=$2 AND deleted_at IS NULL`, id, org).Scan(&a.ID, &a.OrganizationID, &a.PluginID, &a.Name, &a.Key, &a.DataType, &a.CreatedAt, &a.UpdatedAt, &a.SelectionMode, &a.Options)
	if err == pgx.ErrNoRows {
		return nil, &AttributeError{404, "not_found", "attribute not found"}
	}
	if err != nil {
		return nil, err
	}
	if a.PluginID != nil {
		return nil, &AttributeError{403, "plugin_owned", "cannot modify a plugin-owned field"}
	}
	return &a, nil
}
func (r *AttributeRepository) AddOption(ctx context.Context, org, id uuid.UUID, o domain.AttributeOption) (*domain.AttributeOption, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	if err = lockAttributeWrites(ctx, tx, org); err != nil {
		return nil, err
	}
	a, err := loadManagedAttribute(ctx, tx, org, id)
	if err != nil {
		return nil, err
	}
	if a.DataType != domain.AttributeTypeSelect {
		return nil, invalidAttribute("options require a select field")
	}
	o.ID = uuid.New()
	o.SortOrder = len(a.Options)
	a.Options = append(a.Options, o)
	if err = validateDefinition(a); err != nil {
		return nil, err
	}
	o = a.Options[len(a.Options)-1]
	_, err = tx.Exec(ctx, "INSERT INTO attribute_options(id,attribute_id,label,value,sort_order) VALUES($1,$2,$3,$4,$5)", o.ID, id, o.Label, o.Value, o.SortOrder)
	if err != nil {
		return nil, err
	}
	if _, err = tx.Exec(ctx, "UPDATE attributes SET updated_at=NOW() WHERE id=$1", id); err != nil {
		return nil, err
	}
	return &o, tx.Commit(ctx)
}
func (r *AttributeRepository) OrderOptions(ctx context.Context, org, id uuid.UUID, ids []uuid.UUID) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err = lockAttributeWrites(ctx, tx, org); err != nil {
		return err
	}
	a, err := loadManagedAttribute(ctx, tx, org, id)
	if err != nil {
		return err
	}
	if a.DataType != domain.AttributeTypeSelect {
		return invalidAttribute("options require a select field")
	}
	known := map[uuid.UUID]bool{}
	for _, o := range a.Options {
		known[o.ID] = true
	}
	if len(ids) != len(known) {
		return invalidAttribute("supply every option exactly once")
	}
	for i, oid := range ids {
		if !known[oid] {
			return invalidAttribute("supply every option exactly once")
		}
		delete(known, oid)
		if _, err = tx.Exec(ctx, "UPDATE attribute_options SET sort_order=$3 WHERE attribute_id=$1 AND id=$2", id, oid, i); err != nil {
			return err
		}
	}
	if _, err = tx.Exec(ctx, "UPDATE attributes SET updated_at=NOW() WHERE id=$1", id); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
func (r *AttributeRepository) ChangeWithImpact(ctx context.Context, org, user, id uuid.UUID, action AttributeAction, token string, preview bool) (*AttributeImpact, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	if err = lockAttributeWrites(ctx, tx, org); err != nil {
		return nil, err
	}
	a, err := loadManagedAttribute(ctx, tx, org, id)
	if err != nil {
		return nil, err
	}
	next := *a
	next.Options = append([]domain.AttributeOption(nil), a.Options...)
	var oldOption, newOption *domain.AttributeOption
	isOption := action.Action == "update_option" || action.Action == "delete_option"
	if isOption {
		if action.OptionID == nil || a.DataType != domain.AttributeTypeSelect {
			return nil, invalidAttribute("select option ID is required")
		}
		for i := range a.Options {
			if a.Options[i].ID == *action.OptionID {
				oldOption = &a.Options[i]
				newOption = &next.Options[i]
			}
		}
		if oldOption == nil {
			return nil, &AttributeError{404, "not_found", "option not found"}
		}
		if action.Changes.Name != nil || action.Changes.Key != nil || action.Changes.DataType != nil || action.Changes.SelectionMode != nil || action.Changes.Options != nil {
			return nil, invalidAttribute("invalid option changes")
		}
		if action.Changes.Label != nil {
			newOption.Label = *action.Changes.Label
		}
		if action.Changes.Value != nil {
			newOption.Value = *action.Changes.Value
		}
	} else {
		if action.OptionID != nil || action.Changes.Label != nil || action.Changes.Value != nil {
			return nil, invalidAttribute("invalid field changes")
		}
		if action.Changes.Name != nil {
			next.Name = *action.Changes.Name
		}
		if action.Changes.Key != nil {
			next.Key = *action.Changes.Key
		}
		if action.Changes.DataType != nil {
			next.DataType = *action.Changes.DataType
		}
		if action.Changes.SelectionMode != nil {
			next.SelectionMode = *action.Changes.SelectionMode
		}
		if action.Changes.Options != nil {
			if next.DataType != domain.AttributeTypeSelect {
				return nil, invalidAttribute("options require a select field")
			}
			next.Options = append([]domain.AttributeOption(nil), (*action.Changes.Options)...)
			seen := map[uuid.UUID]bool{}
			for i := range next.Options {
				if next.Options[i].ID == uuid.Nil || seen[next.Options[i].ID] {
					return nil, invalidAttribute("every option must have a unique ID")
				}
				seen[next.Options[i].ID] = true
				next.Options[i].SortOrder = i
			}
		}
		if next.DataType != domain.AttributeTypeSelect && a.DataType == domain.AttributeTypeSelect {
			next.SelectionMode = ""
			next.Options = nil
		}
	}
	switch action.Action {
	case "update_attribute", "update_option", "delete_attribute", "delete_option":
	default:
		return nil, invalidAttribute("invalid impact action")
	}
	if err = validateDefinition(&next); err != nil {
		return nil, err
	}
	optionReplacements := map[string]optionReplacement{}
	if action.Action == "update_attribute" && action.Changes.Options != nil {
		nextByID := map[uuid.UUID]domain.AttributeOption{}
		for _, option := range next.Options {
			nextByID[option.ID] = option
		}
		for _, option := range a.Options {
			replacement, found := nextByID[option.ID]
			optionReplacements[option.Value] = optionReplacement{
				value:   replacement.Value,
				changed: !found || option.Label != replacement.Label || option.Value != replacement.Value,
			}
			if !found {
				optionReplacements[option.Value] = optionReplacement{changed: true}
			}
		}
	}
	rows, err := tx.Query(ctx, `SELECT s.id,s.attributes,s.deleted_at IS NOT NULL,
 COALESCE((WITH RECURSIVE parents AS (
 SELECT c.id,c.parent_id,0 depth,ARRAY[c.id] path FROM categories c WHERE c.id=s.category_id AND c.organization_id=$1 AND c.deleted_at IS NULL
 UNION ALL SELECT c.id,c.parent_id,p.depth+1,p.path||c.id FROM categories c JOIN parents p ON c.id=p.parent_id WHERE c.organization_id=$1 AND c.deleted_at IS NULL AND NOT c.id=ANY(p.path)
 ) SELECT ca.required FROM parents p JOIN category_attributes ca ON ca.category_id=p.id WHERE ca.attribute_id=$3 ORDER BY p.depth LIMIT 1),false)
 FROM assets s WHERE s.organization_id=$1 AND s.attributes ? $2 ORDER BY s.id`, org, a.Key, a.ID)
	if err != nil {
		return nil, err
	}
	assets := []impactedAsset{}
	for rows.Next() {
		var s impactedAsset
		if err = rows.Scan(&s.ID, &s.Attributes, &s.Deleted, &s.Required); err != nil {
			rows.Close()
			return nil, err
		}
		assets = append(assets, s)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	if len(assets) > 0 && (a.DataType != next.DataType || a.SelectionMode != next.SelectionMode) {
		return nil, conflictAttribute("cannot change the type or selection mode of a populated field")
	}
	if next.Key != a.Key {
		var conflict bool
		err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM attributes WHERE organization_id=$1 AND key=$2 AND id<>$3)
  OR EXISTS(SELECT 1 FROM retired_attribute_keys WHERE organization_id=$1 AND key=$2)`, org, next.Key, a.ID).Scan(&conflict)
		if err != nil {
			return nil, err
		}
		if conflict {
			return nil, conflictAttribute("field key is already used or retired")
		}
		for _, s := range assets {
			var v map[string]json.RawMessage
			json.Unmarshal(s.Attributes, &v)
			if _, ok := v[next.Key]; ok {
				return nil, conflictAttribute("an affected asset already contains the destination key")
			}
		}
	}
	affected := []impactedAsset{}
	changed := map[uuid.UUID]json.RawMessage{}
	impact := &AttributeImpact{Current: a.Name, Proposed: next.Name}
	if isOption {
		impact.Current = oldOption.Label
		impact.Proposed = newOption.Label
	}
	fieldWideChange := action.Action == "delete_attribute" || a.Name != next.Name || a.Key != next.Key || a.DataType != next.DataType || a.SelectionMode != next.SelectionMode
	batchValuesChanged := false
	for oldValue, replacement := range optionReplacements {
		if replacement.changed && oldValue != replacement.value {
			batchValuesChanged = true
		}
	}
	for _, s := range assets {
		values, decodeErr := decodeAttributeValues(s.Attributes)
		if decodeErr != nil {
			return nil, decodeErr
		}
		v := values[a.Key]
		matches := fieldWideChange
		if isOption {
			matches = selectContains(v, oldOption.Value)
		} else if action.Action == "update_attribute" && !matches {
			for oldValue, replacement := range optionReplacements {
				if replacement.changed && selectContains(v, oldValue) {
					matches = true
					break
				}
			}
		}
		if !matches {
			continue
		}
		affected = append(affected, s)
		if s.Deleted {
			impact.DeletedAssets++
		}
		switch action.Action {
		case "delete_attribute":
			delete(values, a.Key)
		case "update_attribute":
			if action.Changes.Options != nil {
				value := replaceSelectValues(v, optionReplacements)
				if value == nil {
					delete(values, a.Key)
					v = nil
					if s.Required {
						impact.RequiredFieldsLeftEmpty++
					}
				} else {
					values[a.Key] = value
					v = value
				}
			}
			if a.Key != next.Key {
				delete(values, a.Key)
				if v != nil {
					values[next.Key] = v
				}
			}
		case "update_option", "delete_option":
			if action.Action == "delete_option" || oldOption.Value != newOption.Value {
				replacement := newOption.Value
				if action.Action == "delete_option" {
					replacement = ""
				}
				value := replaceSelectValue(v, oldOption.Value, replacement)
				if value == nil {
					delete(values, a.Key)
					if s.Required {
						impact.RequiredFieldsLeftEmpty++
					}
				} else {
					values[a.Key] = value
				}
			}
		}
		raw, _ := json.Marshal(values)
		changed[s.ID] = raw
	}
	impact.AffectedAssets = len(affected)
	expires := time.Now().Add(10 * time.Minute).Unix()
	if !preview {
		parts := strings.Split(token, ".")
		if len(parts) != 2 {
			return nil, confirmationRequired()
		}
		expires, err = strconv.ParseInt(parts[0], 10, 64)
		if err != nil || expires < time.Now().Unix() {
			return nil, confirmationRequired()
		}
	}
	payload, _ := json.Marshal([]any{org, user, action, a, affected, expires})
	var signingKey []byte
	if err = tx.QueryRow(ctx, "SELECT secret FROM attribute_impact_secrets WHERE organization_id=$1", org).Scan(&signingKey); err != nil {
		return nil, err
	}
	mac := hmac.New(sha256.New, signingKey)
	mac.Write(payload)
	signature := hex.EncodeToString(mac.Sum(nil))
	impact.ConfirmationToken = strconv.FormatInt(expires, 10) + "." + signature
	if preview {
		return impact, nil
	}
	if !hmac.Equal([]byte(token), []byte(impact.ConfirmationToken)) {
		return nil, confirmationRequired()
	}
	switch action.Action {
	case "delete_attribute":
		if _, err = tx.Exec(ctx, "DELETE FROM category_attributes WHERE attribute_id=$1", id); err != nil {
			return nil, err
		}
		if _, err = tx.Exec(ctx, "DELETE FROM attribute_options WHERE attribute_id=$1", id); err != nil {
			return nil, err
		}
		_, err = tx.Exec(ctx, "UPDATE attributes SET deleted_at=NOW() WHERE id=$1", id)
	case "update_attribute":
		if next.DataType != domain.AttributeTypeSelect {
			if _, err = tx.Exec(ctx, "DELETE FROM attribute_options WHERE attribute_id=$1", id); err != nil {
				return nil, err
			}
		}
		if next.DataType == domain.AttributeTypeSelect && action.Changes.Options != nil {
			if _, err = tx.Exec(ctx, "DELETE FROM attribute_options WHERE attribute_id=$1", id); err != nil {
				return nil, err
			}
			for _, option := range next.Options {
				if _, err = tx.Exec(ctx, "INSERT INTO attribute_options(id,attribute_id,label,value,sort_order) VALUES($1,$2,$3,$4,$5)", option.ID, id, option.Label, option.Value, option.SortOrder); err != nil {
					return nil, err
				}
			}
		}
		_, err = tx.Exec(ctx, "UPDATE attributes SET name=$2,key=$3,data_type=$4,selection_mode=NULLIF($5,'') WHERE id=$1", id, next.Name, next.Key, next.DataType, next.SelectionMode)
	case "update_option":
		_, err = tx.Exec(ctx, "UPDATE attribute_options SET label=$2,value=$3 WHERE id=$1", oldOption.ID, newOption.Label, newOption.Value)
	case "delete_option":
		_, err = tx.Exec(ctx, "DELETE FROM attribute_options WHERE id=$1", oldOption.ID)
	}
	if err != nil {
		return nil, err
	}
	if isOption {
		if _, err = tx.Exec(ctx, "UPDATE attributes SET updated_at=NOW() WHERE id=$1", id); err != nil {
			return nil, err
		}
	}
	if action.Action == "delete_attribute" || a.Key != next.Key {
		if _, err = tx.Exec(ctx, "INSERT INTO retired_attribute_keys(organization_id,key) VALUES($1,$2) ON CONFLICT DO NOTHING", org, a.Key); err != nil {
			return nil, err
		}
	}
	rewrite := action.Action == "delete_attribute" || a.Key != next.Key || batchValuesChanged || action.Action == "delete_option" || (isOption && oldOption.Value != newOption.Value)
	if rewrite {
		for _, s := range affected {
			if _, err = tx.Exec(ctx, "UPDATE assets SET attributes=$2 WHERE id=$1", s.ID, changed[s.ID]); err != nil {
				return nil, err
			}
		}
	}
	return impact, tx.Commit(ctx)
}
func confirmationRequired() error {
	return &AttributeError{409, "impact_preview_required", "The field or affected assets changed. Review a fresh impact preview and confirm again."}
}
func decodeAttributeValues(raw json.RawMessage) (map[string]any, error) {
	// Preserve arbitrary-precision legacy numbers when rewriting select values.
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var values map[string]any
	err := decoder.Decode(&values)
	return values, err
}

func selectContains(v any, want string) bool {
	if s, ok := v.(string); ok {
		return s == want
	}
	if items, ok := v.([]any); ok {
		for _, s := range items {
			if s == want {
				return true
			}
		}
	}
	return false
}
func replaceSelectValue(v any, old, replacement string) any {
	if s, ok := v.(string); ok {
		if s == old {
			if replacement == "" {
				return nil
			}
			return replacement
		}
		return s
	}
	if items, ok := v.([]any); ok {
		result := []any{}
		for _, s := range items {
			if s == old {
				if replacement != "" {
					result = append(result, replacement)
				}
			} else {
				result = append(result, s)
			}
		}
		if len(result) == 0 {
			return nil
		}
		return result
	}
	return v
}

func replaceSelectValues(v any, replacements map[string]optionReplacement) any {
	replace := func(value string) (string, bool) {
		replacement, found := replacements[value]
		if !found || !replacement.changed {
			return value, true
		}
		return replacement.value, replacement.value != ""
	}
	if value, ok := v.(string); ok {
		result, keep := replace(value)
		if !keep {
			return nil
		}
		return result
	}
	if items, ok := v.([]any); ok {
		result := make([]any, 0, len(items))
		seen := map[string]bool{}
		for _, item := range items {
			value, ok := item.(string)
			if !ok {
				result = append(result, item)
				continue
			}
			value, keep := replace(value)
			if keep && !seen[value] {
				result = append(result, value)
				seen[value] = true
			}
		}
		if len(result) == 0 {
			return nil
		}
		return result
	}
	return v
}

// validateSelectAsset runs inside the same transaction and organization lock as
// the asset write, including writes made by import plugins.
func validateSelectAsset(ctx context.Context, tx pgx.Tx, a *domain.Asset) error {
	if err := lockAttributeReads(ctx, tx, a.OrganizationID); err != nil {
		return err
	}
	var values map[string]any
	if len(a.Attributes) == 0 {
		a.Attributes = json.RawMessage("{}")
	}
	var decodeErr error
	values, decodeErr = decodeAttributeValues(a.Attributes)
	if decodeErr != nil {
		return invalidAttribute("attributes must be a JSON object")
	}
	if values == nil {
		values = map[string]any{}
	}
	var retired bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM retired_attribute_keys WHERE organization_id=$1 AND $2::jsonb ? key)`, a.OrganizationID, a.Attributes).Scan(&retired); err != nil {
		return err
	}
	if retired {
		return conflictAttribute("a field was renamed or deleted; reload the asset before saving")
	}
	var fieldsEnabled bool
	if err := tx.QueryRow(ctx, "SELECT COALESCE((SELECT attributes FROM organization_feature_settings WHERE organization_id=$1),true)", a.OrganizationID).Scan(&fieldsEnabled); err != nil {
		return err
	}
	rows, err := tx.Query(ctx, `WITH RECURSIVE parents AS (
 SELECT c.id,c.parent_id,0 depth,ARRAY[c.id] path FROM categories c WHERE c.id=$2 AND c.organization_id=$1 AND c.deleted_at IS NULL
 UNION ALL SELECT c.id,c.parent_id,p.depth+1,p.path||c.id FROM categories c JOIN parents p ON c.id=p.parent_id WHERE c.organization_id=$1 AND c.deleted_at IS NULL AND NOT c.id=ANY(p.path)
 )
 SELECT a.key,a.selection_mode,COALESCE((SELECT ca.required FROM parents p JOIN category_attributes ca ON ca.category_id=p.id WHERE ca.attribute_id=a.id ORDER BY p.depth LIMIT 1),false),
 COALESCE((SELECT jsonb_agg(value) FROM attribute_options WHERE attribute_id=a.id),'[]'::jsonb)
 FROM attributes a WHERE a.organization_id=$1 AND a.deleted_at IS NULL AND a.data_type='select'`, a.OrganizationID, a.CategoryID)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var key, mode string
		var required bool
		var allowed []string
		if err = rows.Scan(&key, &mode, &required, &allowed); err != nil {
			return err
		}
		required = required && fieldsEnabled
		v, present := values[key]
		if !present {
			if required {
				return invalidAttribute(fmt.Sprintf("%s requires a selection", key))
			}
			continue
		}
		known := map[string]bool{}
		for _, s := range allowed {
			known[s] = true
		}
		if mode == "single" {
			s, ok := v.(string)
			if !ok || !known[s] {
				return invalidAttribute(fmt.Sprintf("%s must contain one configured option", key))
			}
		} else {
			items, ok := v.([]any)
			if !ok {
				return invalidAttribute(fmt.Sprintf("%s must be an array of options", key))
			}
			if len(items) == 0 {
				if required {
					return invalidAttribute(fmt.Sprintf("%s requires a selection", key))
				}
				delete(values, key)
			}
			seen := map[string]bool{}
			for _, item := range items {
				s, ok := item.(string)
				if !ok || !known[s] || seen[s] {
					return invalidAttribute(fmt.Sprintf("%s contains an unknown or duplicate option", key))
				}
				seen[s] = true
			}
		}
	}
	if err = rows.Err(); err != nil {
		return err
	}
	a.Attributes, err = json.Marshal(values)
	return err
}
