package repository

import (
	"fmt"
	"net/url"
	"strings"
)

// validateURLValue validates and normalizes the structured value without fetching its destination.
func validateURLValue(value any, required bool) (map[string]string, error) {
	empty := func() (map[string]string, error) {
		if required {
			return nil, fmt.Errorf("a URL is required")
		}
		return nil, nil
	}
	if value == nil || value == "" {
		return empty()
	}
	object, ok := value.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("expected an object containing url and optional label")
	}
	for key := range object {
		if key != "url" && key != "label" {
			return nil, fmt.Errorf("unknown URL field %s", key)
		}
	}
	destination, ok := object["url"].(string)
	if !ok {
		return nil, fmt.Errorf("url must be a string")
	}
	label := ""
	if raw, exists := object["label"]; exists {
		var valid bool
		label, valid = raw.(string)
		if !valid {
			return nil, fmt.Errorf("label must be a string")
		}
	}
	destination, label = strings.TrimSpace(destination), strings.TrimSpace(label)
	if destination == "" {
		if label != "" {
			return nil, fmt.Errorf("link text requires a URL")
		}
		return empty()
	}
	parsed, err := url.Parse(destination)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Hostname() == "" || strings.ContainsAny(destination, "\\\r\n\t ") {
		return nil, fmt.Errorf("enter an absolute HTTP or HTTPS URL with a hostname")
	}
	result := map[string]string{"url": destination}
	if label != "" {
		result["label"] = label
	}
	return result, nil
}
