package rebrickable

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/lmmendes/attic/internal/domain"
)

type transportFunc func(*http.Request) (*http.Response, error)

func (f transportFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func response(status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}
}
func mockClient(t *testing.T, fn transportFunc) *Client {
	t.Helper()
	t.Setenv(apiKeyEnv, "test-key")
	c := NewClient()
	c.interval = 0
	c.httpClient.Transport = fn
	return c
}
func catalogs(c *Client) []*Plugin { return []*Plugin{NewSets(c), NewMinifigs(c)} }
func fixture(id string) string {
	return fmt.Sprintf(`{"set_num":%q,"name":"Catalog item","num_parts":42,"year":2024,"set_img_url":"https://cdn.rebrickable.com/item.jpg","set_url":"https://rebrickable.com/item/"}`, id)
}

func TestCatalogsSearchAndFetch(t *testing.T) {
	for _, catalog := range []string{"sets", "minifigs"} {
		t.Run(catalog, func(t *testing.T) {
			id := "75192-2"
			if catalog == "minifigs" {
				id = "fig-000001"
			}
			calls := 0
			c := mockClient(t, func(r *http.Request) (*http.Response, error) {
				calls++
				if r.Header.Get("Authorization") != "key test-key" || r.Header.Get("Accept") != "application/json" {
					t.Error("missing API headers")
				}
				if r.URL.Query().Get("key") != "" {
					t.Error("credential in URL")
				}
				if r.Method != http.MethodGet {
					t.Error("expected GET")
				}
				if r.URL.Path == "/api/v3/lego/"+catalog+"/" {
					if r.URL.Query().Get("search") != "Darth & Vader" || r.URL.Query().Get("page_size") != "5" {
						t.Errorf("wrong query: %s", r.URL.RawQuery)
					}
					return response(200, `{"next":"https://example.com/unused","results":[`+fixture(id)+`]}`), nil
				}
				if r.URL.Path != "/api/v3/lego/"+catalog+"/"+id+"/" || r.URL.RawQuery != "" {
					t.Errorf("wrong detail URL: %s", r.URL)
				}
				return response(200, fixture(id)), nil
			})
			p := NewSets(c)
			if catalog == "minifigs" {
				p = NewMinifigs(c)
			}
			results, err := p.Search(t.Context(), "name", " Darth & Vader ", 5)
			if err != nil || len(results) != 1 {
				t.Fatalf("search: %#v, %v", results, err)
			}
			if results[0].ExternalID != id || results[0].Title != "Catalog item" || results[0].ImageURL == nil || !strings.Contains(results[0].Subtitle, "42 pieces") {
				t.Fatalf("unexpected result: %#v", results[0])
			}
			exact, err := p.Search(t.Context(), "catalog_number", id, 5)
			if err != nil || !reflect.DeepEqual(exact, results) {
				t.Fatalf("exact: %#v, %v", exact, err)
			}
			data, err := p.Fetch(t.Context(), id)
			if err != nil {
				t.Fatal(err)
			}
			want := map[string]any{"catalog_number": id, "piece_count": 42, "url": "https://rebrickable.com/item/"}
			if catalog == "sets" {
				want["release_year"] = 2024
			}
			if data.Name != "Catalog item" || data.ExternalID != id || data.ImageURL == nil || !reflect.DeepEqual(data.Attributes, want) {
				t.Fatalf("unexpected data: %#v", data)
			}
			if calls != 3 {
				t.Errorf("got %d requests, expected 3", calls)
			}
		})
	}
}

func TestOptionalFieldsAndEmptyResults(t *testing.T) {
	for _, body := range []string{`{"set_num":"123-1","name":"Minimal"}`, `{"set_num":"123-1","name":"Minimal","set_img_url":null,"num_parts":null,"year":null}`, `{"set_num":"123-1","name":"Minimal","set_img_url":""}`} {
		c := mockClient(t, func(*http.Request) (*http.Response, error) { return response(200, body), nil })
		for _, p := range catalogs(c) {
			data, err := p.Fetch(t.Context(), "123-1")
			if err != nil || data.ImageURL != nil || !reflect.DeepEqual(data.Attributes, map[string]any{"catalog_number": "123-1"}) {
				t.Fatalf("optional fields: %#v %v", data, err)
			}
		}
	}
	for _, status := range []int{200, 404} {
		c := mockClient(t, func(*http.Request) (*http.Response, error) { return response(status, `{"results":[]}`), nil })
		for _, p := range catalogs(c) {
			field := "name"
			if status == 404 {
				field = "catalog_number"
			}
			results, err := p.Search(t.Context(), field, "unknown", 10)
			if err != nil || results == nil || len(results) != 0 {
				t.Fatalf("empty results: %#v %v", results, err)
			}
			if status == 404 {
				if _, err := p.Fetch(t.Context(), "unknown"); !errors.Is(err, errNotFound) {
					t.Fatalf("unknown item: %v", err)
				}
			}
		}
	}
	c := mockClient(t, func(*http.Request) (*http.Response, error) {
		return response(200, `{"set_num":"123-1","name":"Zero","num_parts":0}`), nil
	})
	data, err := NewSets(c).Fetch(t.Context(), "123-1")
	if err != nil || data.Attributes["piece_count"] != 0 {
		t.Fatalf("zero piece count lost: %#v %v", data, err)
	}
}

func TestInvalidResponsesAndFailures(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status int
		body   string
	}{
		{"unauthorized", 401, "secret provider text"}, {"forbidden", 403, "secret provider text"}, {"provider error", 500, "secret provider text"}, {"throttled", 429, ""},
		{"invalid json", 200, "{"}, {"null", 200, "null"}, {"missing results", 200, `{}`}, {"null results", 200, `{"results":null}`}, {"wrong results type", 200, `{"results":{}}`}, {"invalid item", 200, `{"results":[{"set_num":"123-1"}]}`}, {"trailing data", 200, `{"results":[]} garbage`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			c := mockClient(t, func(*http.Request) (*http.Response, error) { calls++; return response(tc.status, tc.body), nil })
			for _, p := range catalogs(c) {
				if _, err := p.Search(t.Context(), "name", "test", 10); err == nil || strings.Contains(err.Error(), "secret provider text") {
					t.Fatalf("expected safe error: %v", err)
				}
			}
			if calls != 2 {
				t.Fatalf("unexpected retries: %d", calls)
			}
		})
	}
	for _, body := range []string{`{}`, `null`, `{"set_num":"other","name":"Mismatch"}`, `{"set_num":"123-1","name":" "}`, `{"set_num":"123-1","name":"Test","num_parts":"42"}`} {
		c := mockClient(t, func(*http.Request) (*http.Response, error) { return response(200, body), nil })
		for _, p := range catalogs(c) {
			if _, err := p.Fetch(t.Context(), "123-1"); err == nil {
				t.Fatalf("accepted invalid detail: %s", body)
			}
		}
	}
	c := mockClient(t, func(*http.Request) (*http.Response, error) { return nil, errors.New("network failure") })
	if _, err := NewSets(c).Fetch(t.Context(), "123-1"); err == nil {
		t.Fatal("expected transport failure")
	}
}

func TestCredentialsAndInputValidation(t *testing.T) {
	calls := 0
	c := mockClient(t, func(*http.Request) (*http.Response, error) { calls++; return response(200, `{}`), nil })
	for _, key := range []string{"", " \t "} {
		t.Setenv(apiKeyEnv, key)
		for _, p := range catalogs(c) {
			if p.Enabled() || !strings.Contains(p.DisabledReason(), apiKeyEnv) {
				t.Fatal("missing disabled setup guidance")
			}
			if _, err := p.Fetch(t.Context(), "123-1"); err == nil {
				t.Fatal("expected missing credentials error")
			}
		}
	}
	t.Setenv(apiKeyEnv, "test-key")
	for _, p := range catalogs(c) {
		if !p.Enabled() || p.DisabledReason() != "" {
			t.Fatal("configured plugin disabled")
		}
		for _, id := range []string{"", "../sets", "123/parts", "123?key=x", "123#x", "123%2fparts"} {
			if _, err := p.Fetch(t.Context(), id); err == nil {
				t.Errorf("accepted ID: %s", id)
			}
		}
		if _, err := p.Search(t.Context(), "unsupported", "test", 10); err == nil {
			t.Fatal("accepted unsupported field")
		}
	}
	if calls != 0 {
		t.Fatal("invalid input made outgoing requests")
	}
}

func TestSharedRateLimit(t *testing.T) {
	var mu sync.Mutex
	var starts []time.Time
	c := mockClient(t, func(*http.Request) (*http.Response, error) {
		mu.Lock()
		starts = append(starts, time.Now())
		mu.Unlock()
		return response(200, `{"results":[]}`), nil
	})
	c.interval = time.Second
	var wg sync.WaitGroup
	for _, p := range catalogs(c) {
		wg.Go(func() {
			if _, err := p.Search(t.Context(), "name", "test", 10); err != nil {
				t.Error(err)
			}
		})
	}
	wg.Wait()
	if len(starts) != 2 || starts[1].Sub(starts[0]) < time.Second-5*time.Millisecond {
		t.Fatalf("requests were not spaced: %v", starts)
	}
}

func TestRetryAfterAndCancellation(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	for _, tc := range []struct {
		header string
		want   time.Time
	}{
		{"2", now.Add(2 * time.Second)}, {now.Add(10 * time.Second).Format(http.TimeFormat), now.Add(10 * time.Second)}, {"invalid", now}, {"-1", now},
	} {
		if got := retryAfter(tc.header, now); !got.Equal(tc.want) {
			t.Errorf("Retry-After %q: %v", tc.header, got)
		}
	}
	calls := 0
	c := mockClient(t, func(*http.Request) (*http.Response, error) {
		calls++
		r := response(429, "")
		r.Header.Set("Retry-After", "2")
		return r, nil
	})
	before := time.Now()
	if _, err := NewSets(c).Search(t.Context(), "name", "test", 10); err == nil {
		t.Fatal("expected throttle error")
	}
	if c.nextRequest.Before(before.Add(2 * time.Second)) {
		t.Fatal("missing throttle cooldown")
	}
	ctx, cancel := context.WithTimeout(t.Context(), 25*time.Millisecond)
	defer cancel()
	if _, err := NewMinifigs(c).Search(ctx, "name", "test", 10); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("cooldown cancellation: %v", err)
	}
	if calls != 1 {
		t.Fatal("request was retried or cooldown ignored")
	}
	// Cancellation while queued behind another request must also be immediate.
	c.gate <- struct{}{}
	ctx2, cancel2 := context.WithCancel(t.Context())
	cancel2()
	if _, err := NewSets(c).Search(ctx2, "name", "test", 10); !errors.Is(err, context.Canceled) {
		t.Fatalf("gate cancellation: %v", err)
	}
	<-c.gate
	c.nextRequest = time.Time{}
	c.httpClient.Transport = transportFunc(func(r *http.Request) (*http.Response, error) { <-r.Context().Done(); return nil, r.Context().Err() })
	c.httpClient.Timeout = 25 * time.Millisecond
	if _, err := NewSets(c).Fetch(t.Context(), "123-1"); err == nil {
		t.Fatal("expected request timeout")
	}
}

var _ domain.ImportPlugin = (*Plugin)(nil)
