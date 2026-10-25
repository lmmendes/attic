package domain

import (
	"encoding/json"
	"regexp"
	"strings"
	"testing"
)

func TestNormalizeCurrencyAcceptsPublishedRegistry(t *testing.T) {
	data, err := currencyRegistryFS.ReadFile("currencies.json")
	if err != nil {
		t.Fatal(err)
	}
	var registry struct {
		Source    string   `json:"source"`
		Published string   `json:"published"`
		Codes     []string `json:"codes"`
	}
	if err := json.Unmarshal(data, &registry); err != nil {
		t.Fatal(err)
	}
	if registry.Source == "" || registry.Published == "" || len(registry.Codes) < 150 {
		t.Fatal("missing currency registry provenance or codes")
	}
	seen := make(map[string]bool)
	valid := regexp.MustCompile(`^[A-Z]{3}$`)
	for _, code := range registry.Codes {
		if !valid.MatchString(code) || seen[code] {
			t.Fatalf("invalid or duplicate currency code: %q", code)
		}
		seen[code] = true
		normalized, err := NormalizeCurrency(" " + strings.ToLower(code) + " ")
		if code == "XXX" || code == "XTS" {
			if err == nil {
				t.Errorf("accepted non-currency sentinel %s", code)
			}
			continue
		}
		if err != nil || normalized != code {
			t.Errorf("currency %s: %q, %v", code, normalized, err)
		}
	}
	for _, code := range []string{"MRU", "SLE", "VES", "XCG", "ZWG", "USD", "EUR", "JPY"} {
		if !seen[code] {
			t.Errorf("current currency %s missing from registry", code)
		}
	}
}

func TestNormalizeCurrencyPreservesHistoricalCodes(t *testing.T) {
	code, err := NormalizeCurrency(" dem ")
	if err != nil || code != "DEM" {
		t.Fatalf("historical currency: %q %v", code, err)
	}
}

func TestNormalizeCurrencyRejectsInvalidCodes(t *testing.T) {
	for _, input := range []string{"", "ZZZ", "EURO", "123", "U$D", "XXX", "XTS"} {
		if _, err := NormalizeCurrency(input); err == nil {
			t.Errorf("accepted invalid currency %q", input)
		}
	}
}
