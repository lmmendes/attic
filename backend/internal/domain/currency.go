package domain

import (
	"embed"
	"encoding/json"
	"errors"
	"strings"

	"golang.org/x/text/currency"
)

// Current ISO 4217 codes are shared with the frontend. Refresh currencies.json
// from its recorded SIX source when the maintenance agency publishes updates,
// and synchronize frontend/app/data/currencies.json (verified by frontend tests).
//
//go:embed currencies.json
var currencyRegistryFS embed.FS

var currentCurrencyCodes = func() map[string]struct{} {
	data, err := currencyRegistryFS.ReadFile("currencies.json")
	if err != nil {
		panic(err)
	}
	var registry struct {
		Codes []string `json:"codes"`
	}
	if err := json.Unmarshal(data, &registry); err != nil {
		panic(err)
	}
	codes := make(map[string]struct{}, len(registry.Codes))
	for _, code := range registry.Codes {
		codes[code] = struct{}{}
	}
	return codes
}()

// NormalizeCurrency accepts current ISO 4217 codes and preserves support for
// historical codes already recognized by the backend's existing currency table.
func NormalizeCurrency(value string) (string, error) {
	code := strings.ToUpper(strings.TrimSpace(value))
	if code == "XXX" || code == "XTS" {
		return "", errors.New("currency must identify a monetary currency")
	}
	if _, ok := currentCurrencyCodes[code]; ok {
		return code, nil
	}
	unit, err := currency.ParseISO(code)
	if err != nil {
		return "", err
	}
	return unit.String(), nil
}
