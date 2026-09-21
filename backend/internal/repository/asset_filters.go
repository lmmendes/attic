package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lmmendes/attic/internal/domain"
)

type FilterError struct{ Issues []domain.FilterIssue }

func (e *FilterError) Error() string {
	return "Some filter rules are unavailable or invalid. Review the highlighted rules."
}

type filterCompiler struct {
	organization uuid.UUID
	attributes   map[string]domain.Attribute
	references   map[string]map[string]bool
	features     domain.OrganizationFeatures
	args         []any
	start        int
	rules        int
	issues       []domain.FilterIssue
}

// FilterValidator shares a live catalog snapshot while compiling independent definitions.
type FilterValidator struct{ template filterCompiler }

func (v *FilterValidator) Compile(criteria domain.FilterCriteria, start int) (string, []any, error) {
	c := v.template
	c.start = start
	return c.compile(criteria)
}

// CompileCriteria validates live references and produces only parameterized predicates.
func (r *AssetRepository) CompileCriteria(ctx context.Context, org uuid.UUID, criteria domain.FilterCriteria, features domain.OrganizationFeatures, start int) (string, []any, error) {
	v, err := r.NewFilterValidator(ctx, org, features)
	if err != nil {
		return "", nil, err
	}
	return v.Compile(criteria, start)
}

func (r *AssetRepository) NewFilterValidator(ctx context.Context, org uuid.UUID, features domain.OrganizationFeatures) (*FilterValidator, error) {
	attrs, err := NewAttributeRepository(r.pool).List(ctx, org)
	if err != nil {
		return nil, err
	}
	c := filterCompiler{organization: org, attributes: map[string]domain.Attribute{}, references: map[string]map[string]bool{}, features: features}
	for _, attr := range attrs {
		c.attributes[attr.ID.String()] = attr
	}
	// Table and field names are fixed, never supplied by the caller.
	for field, table := range map[string]string{"category": "categories", "location": "locations", "condition": "conditions", "collections": "collections", "tags": "tags"} {
		query := "SELECT id::text FROM " + table + " WHERE organization_id=$1"
		if field != "collections" && field != "tags" {
			query += " AND deleted_at IS NULL"
		}
		if field == "category" && !features.Plugins {
			query += " AND plugin_id IS NULL"
		}
		rows, err := r.pool.Query(ctx, query, org)
		if err != nil {
			return nil, err
		}
		c.references[field] = map[string]bool{}
		for rows.Next() {
			var id string
			if err := rows.Scan(&id); err != nil {
				rows.Close()
				return nil, err
			}
			c.references[field][id] = true
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return nil, err
		}
	}
	return &FilterValidator{template: c}, nil
}

func (c *filterCompiler) compile(criteria domain.FilterCriteria) (string, []any, error) {
	if criteria.Version != 1 {
		c.issue("version", "Unsupported filter version")
	}
	parts := []string{}
	for _, entry := range []struct{ field, value, path string }{
		{"category", criteria.CategoryID, "category_id"}, {"location", criteria.LocationID, "location_id"},
		{"condition", criteria.ConditionID, "condition_id"}, {"collections", criteria.CollectionID, "collection_id"},
	} {
		if entry.value != "" {
			parts = append(parts, c.node(domain.FilterNode{Kind: "rule", Field: entry.field, Operator: "any", Values: []string{entry.value}}, entry.path, 0))
		}
	}
	if criteria.Query != "" {
		parts = append(parts, c.node(domain.FilterNode{Kind: "rule", Field: "q", Operator: "search", Value: criteria.Query}, "q", 0))
	}
	if criteria.AttributeQuery != "" {
		parts = append(parts, c.node(domain.FilterNode{Kind: "rule", Field: "attribute_q", Operator: "contains", Value: criteria.AttributeQuery}, "attribute_q", 0))
	}
	if len(criteria.TagIDs) > 0 {
		match := criteria.TagMatch
		if match == "" {
			match = "any"
		}
		parts = append(parts, c.node(domain.FilterNode{Kind: "rule", Field: "tags", Operator: match, Values: criteria.TagIDs}, "tag_ids", 0))
	} else if criteria.TagMatch != "" {
		c.issue("tag_match", "Tag matching requires at least one tag")
	}
	if criteria.Expression != nil {
		parts = append(parts, c.node(*criteria.Expression, "expression", 0))
	}
	if len(c.issues) > 0 {
		return "", nil, &FilterError{Issues: c.issues}
	}
	if len(parts) == 0 {
		return "TRUE", c.args, nil
	}
	return "(" + strings.Join(parts, " AND ") + ")", c.args, nil
}

func (c *filterCompiler) issue(path, message string) string {
	c.issues = append(c.issues, domain.FilterIssue{Path: path, Message: message})
	return "FALSE"
}
func (c *filterCompiler) param(value any) string {
	c.args = append(c.args, value)
	return fmt.Sprintf("$%d", c.start+len(c.args)-1)
}

func (c *filterCompiler) node(n domain.FilterNode, path string, depth int) string {
	if n.Kind == "group" {
		if depth >= 5 {
			return c.issue(path, "Use at most five levels of groups")
		}
		if n.Match != "all" && n.Match != "any" {
			return c.issue(path, "Choose Match all or Match any")
		}
		if len(n.Children) == 0 || len(n.Children) > 50 {
			return c.issue(path, "Groups require between one and 50 children")
		}
		if n.Field != "" || n.Operator != "" || n.Value != nil || len(n.Values) > 0 || n.AttributeID != "" || n.DataType != "" || n.SelectionMode != "" || n.Upper != nil {
			return c.issue(path, "A group cannot contain rule fields")
		}
		parts := []string{}
		for i, child := range n.Children {
			parts = append(parts, c.node(child, fmt.Sprintf("%s.children[%d]", path, i), depth+1))
		}
		join := " AND "
		if n.Match == "any" {
			join = " OR "
		}
		return "(" + strings.Join(parts, join) + ")"
	}
	if n.Kind != "rule" || len(n.Children) > 0 || n.Match != "" {
		return c.issue(path, "Expected a rule or a group")
	}
	c.rules++
	if c.rules > 50 {
		return c.issue(path, "Use at most 50 rules")
	}
	if len(n.Values) > 100 {
		return c.issue(path, "Select at most 100 values")
	}
	if n.Field != "attribute" && (n.AttributeID != "" || n.DataType != "" || n.SelectionMode != "") {
		return c.issue(path, "Only attribute rules may reference an attribute definition")
	}
	if n.Field == "q" || n.Field == "attribute_q" {
		s, ok := n.Value.(string)
		if !ok || strings.TrimSpace(s) == "" || len(s) > 1000 || n.Upper != nil || len(n.Values) > 0 {
			return c.issue(path, "Enter search text (up to 1000 bytes)")
		}
		if n.Field == "q" && n.Operator == "search" {
			p := c.param(s)
			if !c.features.Tags {
				return `a.search_vector @@ attic_prefix_tsquery(` + p + `)`
			}
			return `(a.search_vector @@ attic_prefix_tsquery(` + p + `)
 OR EXISTS (SELECT 1 FROM asset_tags at JOIN tags t ON t.id=at.tag_id
 WHERE at.asset_id=a.id AND t.organization_id=a.organization_id
 AND to_tsvector('english',t.name) @@ attic_prefix_tsquery(` + p + `)))`
		}
		if n.Field == "attribute_q" && n.Operator == "contains" {
			if !c.features.Attributes {
				return c.issue(path, "Attributes are disabled")
			}
			return c.quick(s)
		}
		return c.issue(path, "Unsupported search operator")
	}
	if n.Field == "attribute" {
		return c.attribute(n, path)
	}
	enabled := map[string]bool{"collections": c.features.Collections, "tags": c.features.Tags, "category": c.features.Categories, "location": c.features.Locations, "condition": c.features.Conditions}
	if !enabled[n.Field] {
		return c.issue(path, "This field is unavailable or disabled")
	}
	if n.Value != nil || n.Upper != nil || len(n.Values) == 0 || (n.Operator != "any" && !((n.Field == "collections" || n.Field == "tags") && n.Operator == "all")) {
		return c.issue(path, "Choose a supported membership operator and at least one value")
	}
	parts := []string{}
	for _, id := range n.Values {
		if n.Field == "category" && id == "uncategorized" {
			parts = append(parts, "a.category_id IS NULL")
			continue
		}
		if !c.references[n.Field][id] {
			c.issue(path, "A selected "+n.Field+" was deleted or is unavailable")
			continue
		}
		p := c.param(id)
		switch n.Field {
		case "collections":
			parts = append(parts, "EXISTS (SELECT 1 FROM asset_collections ac JOIN collections cl ON cl.id=ac.collection_id AND cl.organization_id=a.organization_id WHERE ac.asset_id=a.id AND ac.collection_id="+p+"::uuid)")
		case "tags":
			parts = append(parts, "EXISTS (SELECT 1 FROM asset_tags at JOIN tags t ON t.id=at.tag_id AND t.organization_id=a.organization_id WHERE at.asset_id=a.id AND at.tag_id="+p+"::uuid)")
		case "category":
			visible := ""
			if !c.features.Plugins {
				visible = " AND child.plugin_id IS NULL"
			}
			parts = append(parts, `EXISTS (WITH RECURSIVE descendants AS (
 SELECT id FROM categories WHERE id=`+p+`::uuid AND organization_id=a.organization_id AND deleted_at IS NULL
 UNION SELECT child.id FROM categories child JOIN descendants parent ON child.parent_id=parent.id
 WHERE child.organization_id=a.organization_id AND child.deleted_at IS NULL`+visible+`
 ) SELECT 1 FROM descendants WHERE descendants.id=a.category_id)`)
		default:
			parts = append(parts, "a."+n.Field+"_id="+p+"::uuid")
		}
	}
	join := " OR "
	if n.Operator == "all" {
		join = " AND "
	}
	return "(" + strings.Join(parts, join) + ")"
}

func (c *filterCompiler) quick(s string) string {
	p := c.param(s)
	org := c.param(c.organization)
	plugin := ""
	if !c.features.Plugins {
		plugin = " AND d.plugin_id IS NULL"
	}
	// Scalar values are searched as text, while configured selections use labels.
	// strpos treats %, _ and backslashes literally instead of as SQL wildcards.
	return `EXISTS (SELECT 1 FROM jsonb_each(CASE WHEN jsonb_typeof(a.attributes)='object' THEN a.attributes ELSE '{}'::jsonb END) v(key,value)
 JOIN attributes d ON d.key=v.key AND d.organization_id=` + org + `::uuid AND d.deleted_at IS NULL` + plugin + `
 WHERE (d.data_type<>'select' AND jsonb_typeof(v.value) IN ('string','number','boolean')
 AND strpos(lower(v.value #>> '{}'),lower(` + p + `::text))>0)
 OR (d.data_type='select' AND EXISTS (SELECT 1 FROM attribute_options o WHERE o.attribute_id=d.id
 AND (v.value=to_jsonb(o.value) OR (jsonb_typeof(v.value)='array' AND v.value @> jsonb_build_array(o.value)))
 AND strpos(lower(o.label),lower(` + p + `::text))>0)))`
}

func (c *filterCompiler) attribute(n domain.FilterNode, path string) string {
	a, ok := c.attributes[n.AttributeID]
	if !ok || !c.features.Attributes || (a.PluginID != nil && !c.features.Plugins) {
		return c.issue(path, "This attribute was deleted or is unavailable")
	}
	if n.DataType != a.DataType || n.SelectionMode != a.SelectionMode {
		return c.issue(path, "The attribute type changed; choose its comparison again")
	}
	p := c.param(a.Key)
	raw := "(a.attributes -> " + p + "::text)"
	txt := "(a.attributes ->> " + p + "::text)"
	if n.Operator == "empty" || n.Operator == "not_empty" {
		if n.Value != nil || n.Upper != nil || len(n.Values) > 0 {
			return c.issue(path, "Empty checks do not accept values")
		}
		empty := "(" + raw + " IS NULL OR " + raw + " IN ('null'::jsonb,'\"\"'::jsonb,'[]'::jsonb))"
		if n.Operator == "not_empty" {
			return "NOT " + empty
		}
		return empty
	}
	if a.DataType == domain.AttributeTypeSelect {
		if n.Value != nil || n.Upper != nil || len(n.Values) == 0 || (n.Operator != "any" && !(n.Operator == "all" && a.SelectionMode == "multiple")) {
			return c.issue(path, "Choose a supported selection operator and options")
		}
		options := map[string]string{}
		for _, o := range a.Options {
			options[o.ID.String()] = o.Value
		}
		parts := []string{}
		for _, id := range n.Values {
			v, exists := options[id]
			if !exists {
				c.issue(path, "A selected option was deleted or is unavailable")
				continue
			}
			value := c.param(v)
			if a.SelectionMode == "multiple" {
				parts = append(parts, "(jsonb_typeof("+raw+")='array' AND "+raw+" @> jsonb_build_array("+value+"::text))")
			} else {
				parts = append(parts, raw+"=to_jsonb("+value+"::text)")
			}
		}
		join := " OR "
		if n.Operator == "all" {
			join = " AND "
		}
		return "(" + strings.Join(parts, join) + ")"
	}
	if len(n.Values) > 0 {
		return c.issue(path, "This comparison requires one value")
	}
	if a.DataType == domain.AttributeTypeString || a.DataType == domain.AttributeTypeText {
		s, valid := n.Value.(string)
		if !valid || len(s) > 1000 || n.Upper != nil {
			return c.issue(path, "Enter a text value (up to 1000 bytes)")
		}
		value := c.param(s)
		guard := "jsonb_typeof(" + raw + ")='string' AND "
		if n.Operator == "eq" {
			return "(" + guard + "lower(" + txt + ")=lower(" + value + "::text))"
		}
		if n.Operator == "contains" {
			return "(" + guard + "strpos(lower(" + txt + "),lower(" + value + "::text))>0)"
		}
		return c.issue(path, "Unsupported text comparison")
	}
	if a.DataType == domain.AttributeTypeBoolean {
		v, valid := n.Value.(bool)
		if !valid || n.Operator != "eq" || n.Upper != nil {
			return c.issue(path, "Choose true or false")
		}
		return raw + "=to_jsonb(" + c.param(v) + "::boolean)"
	}
	ops := map[string]string{"eq": "=", "lt": "<", "lte": "<=", "gt": ">", "gte": ">="}
	if ops[n.Operator] == "" && n.Operator != "between" {
		return c.issue(path, "Unsupported comparison")
	}
	if n.Operator != "between" && n.Upper != nil {
		return c.issue(path, "Only a range accepts an upper value")
	}
	var expr, lo, hi string
	if a.DataType == domain.AttributeTypeNumber {
		v, valid := filterNumber(n.Value)
		if !valid {
			return c.issue(path, "Enter a finite number")
		}
		lo = c.param(v) + "::numeric"
		expr = "(CASE WHEN jsonb_typeof(" + raw + ")='number' THEN " + txt + "::numeric END)"
		if n.Operator == "between" {
			upper, valid := filterNumber(n.Upper)
			if !valid || compareFilterNumbers(upper, v) < 0 {
				return c.issue(path, "Enter an upper number at least as large as the lower number")
			}
			hi = c.param(upper) + "::numeric"
		}
	} else if a.DataType == domain.AttributeTypeDate {
		v, valid := filterDate(n.Value)
		if !valid {
			return c.issue(path, "Enter a valid date as YYYY-MM-DD")
		}
		lo = c.param(v) + "::text"
		// PostgreSQL 16 input validation guards malformed legacy dates without casting them.
		expr = "(CASE WHEN jsonb_typeof(" + raw + ")='string' AND " + txt + " ~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}$' AND pg_input_is_valid(" + txt + ",'date') THEN " + txt + " END)"
		if n.Operator == "between" {
			upper, valid := filterDate(n.Upper)
			if !valid || upper < v {
				return c.issue(path, "Enter an upper date on or after the lower date")
			}
			hi = c.param(upper) + "::text"
		}
	} else {
		return c.issue(path, "Unsupported attribute type")
	}
	if n.Operator == "between" {
		return "(" + expr + ">=" + lo + " AND " + expr + "<=" + hi + ")"
	}
	return expr + ops[n.Operator] + lo
}

var filterNumberPattern = regexp.MustCompile(`^-?(0|[1-9][0-9]*)(\.[0-9]+)?([eE][+-]?[0-9]+)?$`)

func filterNumber(v any) (string, bool) {
	var s string
	switch n := v.(type) {
	case float64:
		if math.IsNaN(n) || math.IsInf(n, 0) {
			return "", false
		}
		s = strconv.FormatFloat(n, 'g', -1, 64)
	case json.Number:
		s = string(n)
	default:
		return "", false
	}
	if len(s) > 1000 || !filterNumberPattern.MatchString(s) {
		return "", false
	}
	if i := strings.IndexAny(s, "eE"); i >= 0 {
		exponent, err := strconv.Atoi(s[i+1:])
		if err != nil || exponent < -1000 || exponent > 1000 {
			return "", false
		}
	}
	return s, true
}

func compareFilterNumbers(a, b string) int {
	left, _ := new(big.Rat).SetString(a)
	right, _ := new(big.Rat).SetString(b)
	return left.Cmp(right)
}
func filterDate(v any) (string, bool) {
	s, ok := v.(string)
	if !ok {
		return "", false
	}
	_, err := time.Parse("2006-01-02", s)
	return s, len(s) == 10 && err == nil
}
