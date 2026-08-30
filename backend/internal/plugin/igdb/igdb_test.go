package igdb

import (
	"testing"
)

func Test_parseExternalID_GameAndPlatform_ParsesBoth(t *testing.T) {
	gameID, platformID, err := parseExternalID("7346:130")

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if gameID != 7346 {
		t.Errorf("expected gameID=7346, got %d", gameID)
	}
	if platformID != 130 {
		t.Errorf("expected platformID=130, got %d", platformID)
	}
}

func Test_parseExternalID_GameOnly_ReturnsZeroPlatform(t *testing.T) {
	gameID, platformID, err := parseExternalID("7346")

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if gameID != 7346 {
		t.Errorf("expected gameID=7346, got %d", gameID)
	}
	if platformID != 0 {
		t.Errorf("expected platformID=0, got %d", platformID)
	}
}

func Test_parseExternalID_InvalidGameID_ReturnsError(t *testing.T) {
	tests := []string{"", "abc", "abc:1", "-1", "0", "0:1"}

	for _, in := range tests {
		_, _, err := parseExternalID(in)
		if err == nil {
			t.Errorf("expected error for input %q", in)
		}
	}
}

func Test_parseExternalID_InvalidPlatformID_ReturnsError(t *testing.T) {
	_, _, err := parseExternalID("7346:abc")
	if err == nil {
		t.Fatal("expected error for non-numeric platform id")
	}
}

func Test_formatSubtitle_PlatformAndYear_Combined(t *testing.T) {
	tests := []struct {
		platform string
		year     string
		want     string
	}{
		{"PlayStation 5", "2020", "PlayStation 5 (2020)"},
		{"PC", "", "PC"},
		{"", "2017", "(2017)"},
		{"", "", ""},
	}

	for _, tt := range tests {
		got := formatSubtitle(tt.platform, tt.year)
		if got != tt.want {
			t.Errorf("formatSubtitle(%q, %q) = %q; want %q", tt.platform, tt.year, got, tt.want)
		}
	}
}

func Test_coverURL_BuildsImageURL(t *testing.T) {
	got := coverURL("co5w8p", "t_cover_big")
	want := "https://images.igdb.com/igdb/image/upload/t_cover_big/co5w8p.jpg"

	if got != want {
		t.Errorf("coverURL mismatch:\n got: %s\nwant: %s", got, want)
	}
}

func Test_formatDate_UnixToISO(t *testing.T) {
	// 2017-03-03 00:00:00 UTC
	got := formatDate(1488499200)
	want := "2017-03-03"
	if got != want {
		t.Errorf("formatDate = %q; want %q", got, want)
	}

	if formatDate(0) != "" {
		t.Errorf("expected empty string for zero unix timestamp")
	}
}

func Test_selectTopPlatforms_SortsByReleaseDateDescending(t *testing.T) {
	g := searchGame{
		Platforms: []namedRef{
			{ID: 1, Name: "Old"},
			{ID: 2, Name: "New"},
			{ID: 3, Name: "Mid"},
		},
		ReleaseDates: []releaseDate{
			{Platform: 1, Date: 1000},
			{Platform: 2, Date: 3000},
			{Platform: 3, Date: 2000},
		},
	}

	got := selectTopPlatforms(g)

	if len(got) != 3 {
		t.Fatalf("expected 3 platforms, got %d", len(got))
	}
	if got[0].Name != "New" || got[1].Name != "Mid" || got[2].Name != "Old" {
		t.Errorf("unexpected order: %+v", got)
	}
}

func Test_selectTopPlatforms_CapsToMax(t *testing.T) {
	g := searchGame{}
	for i := 1; i <= maxPlatformsPerGame+3; i++ {
		g.Platforms = append(g.Platforms, namedRef{ID: int64(i), Name: "P"})
	}

	got := selectTopPlatforms(g)

	if len(got) != maxPlatformsPerGame {
		t.Errorf("expected %d platforms, got %d", maxPlatformsPerGame, len(got))
	}
}

func Test_splitCompanies_PartitionsByRole(t *testing.T) {
	input := []involvedCompany{
		{Developer: true, Company: struct {
			Name string `json:"name"`
		}{Name: "CD Projekt Red"}},
		{Publisher: true, Company: struct {
			Name string `json:"name"`
		}{Name: "CD Projekt"}},
		{Developer: true, Publisher: true, Company: struct {
			Name string `json:"name"`
		}{Name: "Both"}},
		{Company: struct {
			Name string `json:"name"`
		}{Name: "Neither"}}, // Should be ignored
	}

	devs, pubs := splitCompanies(input)

	if devs != "CD Projekt Red, Both" {
		t.Errorf("unexpected developers: %q", devs)
	}
	if pubs != "CD Projekt, Both" {
		t.Errorf("unexpected publishers: %q", pubs)
	}
}

func Test_joinNamed_IgnoresEmptyNames(t *testing.T) {
	input := []namedRef{
		{Name: "Action"},
		{Name: ""},
		{Name: "Adventure"},
	}

	got := joinNamed(input)
	want := "Action, Adventure"
	if got != want {
		t.Errorf("joinNamed = %q; want %q", got, want)
	}
}

func Test_categoryLabel_KnownAndUnknown(t *testing.T) {
	if got := categoryLabel(0); got != "Main Game" {
		t.Errorf("category 0 = %q; want Main Game", got)
	}
	if got := categoryLabel(8); got != "Remake" {
		t.Errorf("category 8 = %q; want Remake", got)
	}
	if got := categoryLabel(999); got != "" {
		t.Errorf("unknown category should return empty, got %q", got)
	}
}

func Test_pickDescription_PrefersSummary(t *testing.T) {
	if got := pickDescription("summary", "storyline"); got != "summary" {
		t.Errorf("expected summary preferred, got %q", got)
	}
	if got := pickDescription("", "storyline"); got != "storyline" {
		t.Errorf("expected fallback to storyline, got %q", got)
	}
	if got := pickDescription("", ""); got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

func Test_roundTo_RoundsValue(t *testing.T) {
	if got := roundTo(87.456, 1); got != 87.5 {
		t.Errorf("roundTo(87.456, 1) = %v; want 87.5", got)
	}
	if got := roundTo(87.44, 1); got != 87.4 {
		t.Errorf("roundTo(87.44, 1) = %v; want 87.4", got)
	}
}

func Test_Plugin_MetadataValues(t *testing.T) {
	p := New()

	if p.ID() != "igdb_games" {
		t.Errorf("ID = %q; want igdb_games", p.ID())
	}
	if p.CategoryName() != "Video Games" {
		t.Errorf("CategoryName = %q; want Video Games", p.CategoryName())
	}

	if len(p.SearchFields()) == 0 {
		t.Error("expected at least one search field")
	}

	attrs := p.Attributes()
	if len(attrs) == 0 {
		t.Fatal("expected attributes to be defined")
	}

	// Sanity check: every attribute uses the games.* namespace.
	for _, a := range attrs {
		if len(a.Key) < len("games.") || a.Key[:len("games.")] != "games." {
			t.Errorf("attribute %q is not in the games.* namespace", a.Key)
		}
	}
}

func Test_Plugin_DisabledWhenCredentialsMissing(t *testing.T) {
	t.Setenv(clientIDEnvVar, "")
	t.Setenv(clientSecretEnvVar, "")
	ClientID, ClientSecret = "", ""

	p := New()
	if p.Enabled() {
		t.Error("plugin should be disabled without credentials")
	}
	reason := p.DisabledReason()
	if reason == "" {
		t.Error("expected disabled reason when credentials missing")
	}
}

func Test_Plugin_EnabledWithEnvCredentials(t *testing.T) {
	t.Setenv(clientIDEnvVar, "fake-id")
	t.Setenv(clientSecretEnvVar, "fake-secret")

	p := New()
	if !p.Enabled() {
		t.Errorf("plugin should be enabled with credentials, got reason: %s", p.DisabledReason())
	}
}
