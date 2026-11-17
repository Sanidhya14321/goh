package provider

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Sanidhya14321/goh/internal/core"
)

// A 413 that states the provider's per-request cap (Groq's free tier) is
// reported with the two numbers only; the rest of the body never surfaces.
func TestRequestTooLargeReportsStatedCap(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusRequestEntityTooLarge)
		w.Write([]byte(`{"error":{"message":"Request too large for model ` + "`m`" + ` in organization ` + "`org_private-reflected`" + ` service tier on_demand on tokens per minute (TPM): Limit 8000, Requested 11532, please reduce your message size and try again.","type":"tokens","code":"rate_limit_exceeded"}}`))
	}))
	defer server.Close()
	c := New(core.Config{BaseURL: server.URL + "/v1/", Model: "m", Protocol: "chat_completions", OutputBudget: 128})
	_, err := c.Complete(context.Background(), []core.Message{{Role: "user", Content: "hi"}}, nil, nil)
	var h *HTTPError
	if !errors.As(err, &h) || h.Status != 413 || h.Limit != 8000 || h.Requested != 11532 {
		t.Fatalf("%#v %v", h, err)
	}
	if msg := err.Error(); !strings.Contains(msg, "at most 8000 tokens per request") || !strings.Contains(msg, "needed 11532") || strings.Contains(msg, "org_private") {
		t.Fatal(msg)
	}
}

func TestRequestCapParsing(t *testing.T) {
	for body, want := range map[string][2]int{
		`Limit 8000, Requested 9512`:              {8000, 9512},
		`{"limit": 30000, "requested": 41000}`:    {0, 0}, // quoted JSON keys are not the stated form
		`request entity too large`:                {0, 0},
		`limit=6000 requested=7000`:               {6000, 7000},
		`Limit 12 (too small to be a token cap)`:  {0, 0},
		strings.Repeat("x", 9<<10) + `Limit 8000`: {0, 0}, // beyond the bounded read
	} {
		limit, requested, _ := requestCap(strings.NewReader(body))
		if limit != want[0] || requested != want[1] {
			t.Errorf("%.40q: got %d %d", body, limit, requested)
		}
	}
}

// A daily quota (Groq: "tokens per day … Please try again in 28m37.6s")
// fails at once with the reset time instead of three short retries.
func TestQuotaResetFailsWithoutRetrying(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"error":{"message":"Rate limit reached for model ` + "`m`" + ` on tokens per day (TPD): Limit 200000, Used 173934, Requested 30042. Please try again in 28m37.631999999s. Need more tokens?","type":"tokens","code":"rate_limit_exceeded"}}`))
	}))
	defer server.Close()
	c := New(core.Config{BaseURL: server.URL + "/v1/", Model: "m", Protocol: "chat_completions", OutputBudget: 128})
	_, err := c.Complete(context.Background(), []core.Message{{Role: "user", Content: "hi"}}, nil, nil)
	var h *HTTPError
	if !errors.As(err, &h) || h.Status != 429 || h.TooLarge() || calls != 1 {
		t.Fatal(err, calls)
	}
	if msg := err.Error(); !strings.Contains(msg, "resets in about 29m (") {
		t.Fatal(msg)
	}
	for body, want := range map[string]time.Duration{
		`Please try again in 450ms.`: 450 * time.Millisecond,
		`try again in 1h2m`:          time.Hour + 2*time.Minute,
		`try again in soon`:          0,
	} {
		if _, _, wait := requestCap(strings.NewReader(body)); wait != want {
			t.Errorf("%q: %v", body, wait)
		}
	}
}
