package domain

import (
	"embed"
	"encoding/json"
	"errors"
	"sort"
	"strings"
)

// Refresh the single registry from its recorded SIX source when the ISO 4217
// maintenance agency publishes updates. Names and entities are retained for
// future use; validation and the settings helper expose current monetary codes.
//
//go:embed currencies.json
var currencyRegistryFS embed.FS

type currencyRecord struct {
	Code     string   `json:"code"`
	Name     string   `json:"name"`
	Entities []string `json:"entities"`
}

type currencyRegistry struct {
	Source     string           `json:"source"`
	Published  string           `json:"published"`
	Currencies []currencyRecord `json:"currencies"`
}

var currentCurrencyCodes = func() map[string]struct{} {
	data, err := currencyRegistryFS.ReadFile("currencies.json")
	if err != nil {
		panic(err)
	}
	var registry currencyRegistry
	if err := json.Unmarshal(data, &registry); err != nil {
		panic(err)
	}
	codes := make(map[string]struct{}, len(registry.Currencies))
	for _, entry := range registry.Currencies {
		if entry.Code != "XXX" && entry.Code != "XTS" {
			codes[entry.Code] = struct{}{}
		}
	}
	return codes
}()

// AvailableCurrencies returns the current monetary codes accepted by validation.
// A new sorted slice is returned so callers cannot modify the registry.
func AvailableCurrencies() []string {
	codes := make([]string, 0, len(currentCurrencyCodes))
	for code := range currentCurrencyCodes {
		codes = append(codes, code)
	}
	sort.Strings(codes)
	return codes
}

// NormalizeCurrency accepts only current monetary codes from the registry.
func NormalizeCurrency(value string) (string, error) {
	code := strings.ToUpper(strings.TrimSpace(value))
	if _, ok := currentCurrencyCodes[code]; !ok {
		return "", errors.New("unsupported current currency")
	}
	return code, nil
}
