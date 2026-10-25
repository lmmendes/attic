package domain

import (
	"encoding/json"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"
)

func TestCurrentCurrencyRegistryMetadata(t *testing.T) {
	data, err := currencyRegistryFS.ReadFile("currencies.json")
	if err != nil {
		t.Fatal(err)
	}
	var registry currencyRegistry
	if err := json.Unmarshal(data, &registry); err != nil {
		t.Fatal(err)
	}
	if registry.Source == "" || !regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`).MatchString(registry.Published) || len(registry.Currencies) < 150 {
		t.Fatal("missing currency registry provenance or entries")
	}
	seen := make(map[string]bool)
	codes := make([]string, 0, len(registry.Currencies))
	for _, entry := range registry.Currencies {
		if !regexp.MustCompile(`^[A-Z]{3}$`).MatchString(entry.Code) || seen[entry.Code] || entry.Code == "XXX" || entry.Code == "XTS" {
			t.Fatalf("invalid, duplicate, or non-monetary code: %q", entry.Code)
		}
		seen[entry.Code] = true
		if strings.TrimSpace(entry.Name) == "" || len(entry.Entities) == 0 || !sort.StringsAreSorted(entry.Entities) {
			t.Fatalf("invalid metadata for %s: %+v", entry.Code, entry)
		}
		entities := make(map[string]bool)
		for _, entity := range entry.Entities {
			if strings.TrimSpace(entity) == "" || entities[entity] {
				t.Fatalf("empty or repeated entity for %s", entry.Code)
			}
			entities[entity] = true
		}
		if entry.Code == "EUR" && (entry.Name != "Euro" || !entities["AUSTRIA"] || !entities["BELGIUM"]) {
			t.Fatalf("expected currency name and multiple associated entities: %+v", entry)
		}
		codes = append(codes, entry.Code)
		normalized, err := NormalizeCurrency(" " + strings.ToLower(entry.Code) + " ")
		if err != nil || normalized != entry.Code {
			t.Errorf("currency %s: %q, %v", entry.Code, normalized, err)
		}
	}
	if !sort.StringsAreSorted(codes) || !reflect.DeepEqual(codes, AvailableCurrencies()) {
		t.Fatal("registry and available codes must have the same sorted contents")
	}
	for _, code := range []string{"MRU", "SLE", "VES", "XCG", "ZWG", "USD", "EUR", "JPY"} {
		if !seen[code] {
			t.Errorf("current currency %s missing from registry", code)
		}
	}
}

func TestAvailableCurrenciesReturnsIndependentSlice(t *testing.T) {
	original := AvailableCurrencies()
	copy := AvailableCurrencies()
	copy[0] = "ZZZ"
	if !reflect.DeepEqual(original, AvailableCurrencies()) {
		t.Fatal("caller modified the supported currency set")
	}
}

func TestNormalizeCurrencyRejectsHistoricalAndInvalidCodes(t *testing.T) {
	for _, input := range []string{"DEM", " frf ", "", "ZZZ", "EURO", "123", "U$D", "XXX", "XTS"} {
		if _, err := NormalizeCurrency(input); err == nil {
			t.Errorf("accepted unsupported currency %q", input)
		}
	}
}
