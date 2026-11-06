package provider

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"example.com/independent-harness/internal/core"
)

func TestProviderRedirectNeverForwardsPromptOrCredential(t *testing.T) {
	for _, protocol := range []string{"chat_completions", "anthropic_messages"} {
		t.Run(protocol, func(t *testing.T) {
			targetCalls := atomic.Int32{}
			target := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { targetCalls.Add(1) }))
			defer target.Close()
			originCalls := atomic.Int32{}
			origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				originCalls.Add(1)
				if protocol == "anthropic_messages" {
					if r.URL.Path != "/proxy/v1/messages" || r.Header.Get("x-api-key") != "isolated-test-secret" {
						t.Error("native route or credential missing")
					}
				} else if r.URL.Path != "/proxy/v1/chat/completions" || r.Header.Get("Authorization") != "Bearer isolated-test-secret" {
					t.Error("Chat Completions route or credential missing")
				}
				http.Redirect(w, r, target.URL, http.StatusTemporaryRedirect)
			}))
			defer origin.Close()
			t.Setenv("FIXTURE_API_KEY", "isolated-test-secret")
			c := New(core.Config{BaseURL: origin.URL + "/proxy/v1/", Model: "fixture", Protocol: protocol, APIKeyEnv: "FIXTURE_API_KEY", OutputBudget: 128})
			_, err := c.Complete(context.Background(), []core.Message{{Role: "user", Content: "private prompt"}}, nil, nil)
			var status *HTTPError
			if !errors.As(err, &status) || status.Kind != "redirect_refused" || status.Retryable || targetCalls.Load() != 0 || originCalls.Load() != 1 {
				t.Fatal(err, targetCalls.Load(), originCalls.Load())
			}
		})
	}
}
func TestHTTPClassificationAndErrorBodyRedaction(t *testing.T) {
	for _, code := range []int{400, 401, 403, 408, 413, 429, 500, 529} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			requests := atomic.Int32{}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				w.Header().Set("Retry-After", "0")
				w.WriteHeader(code)
				io.WriteString(w, "BODY_SECRET and credential reflection")
			}))
			defer server.Close()
			c := New(core.Config{Endpoint: server.URL, Model: "fixture"})
			_, err := c.Complete(context.Background(), nil, nil, nil)
			var status *HTTPError
			if !errors.As(err, &status) || status.Status != code || strings.Contains(err.Error(), "BODY_SECRET") {
				t.Fatal(err)
			}
			want := int32(1)
			if status.Retryable {
				want = 3
			}
			if requests.Load() != want {
				t.Fatal(requests.Load(), want)
			}
		})
	}
}
func TestRetryBackoffCancellationAndUnknownProtocolNoRequest(t *testing.T) {
	entered := make(chan struct{})
	var count atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if count.Add(1) == 1 {
			close(entered)
		}
		w.Header().Set("Retry-After", "5")
		w.WriteHeader(429)
	}))
	defer server.Close()
	c := New(core.Config{Endpoint: server.URL, Model: "fixture"})
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { _, err := c.Complete(ctx, nil, nil, nil); done <- err }()
	<-entered
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("backoff ignored cancel")
	}
	c.Config.Protocol = "unsupported"
	if _, err := c.Complete(context.Background(), nil, nil, nil); err == nil || count.Load() != 1 {
		t.Fatal("invalid protocol contacted endpoint", err, count.Load())
	}
}
func TestChatTerminalAndMalformedPayloadDoNotReflectData(t *testing.T) {
	if _, err := Parse(strings.NewReader(textFrame("ok")+finish("stop")+"data: [DONE]\n\ndata: [DONE]\n\n"), nil); err == nil {
		t.Fatal("duplicate DONE accepted")
	}
	if _, err := Parse(strings.NewReader("data: {\"error_secret\":SECRET_PAYLOAD}\n\n"), nil); err == nil || strings.Contains(err.Error(), "SECRET_PAYLOAD") {
		t.Fatal(err)
	}
}

func TestProviderEndpointConfigurationRejectedWithoutEcho(t *testing.T) {
	for _, endpoint := range []string{"file:///secret-prompt", "http://user:DO_NOT_ECHO@localhost", "http://fixture/path#secret", "http://fixture/\x00SECRET"} {
		c := New(core.Config{Endpoint: endpoint})
		if _, err := c.Complete(context.Background(), nil, nil, nil); err == nil || strings.Contains(err.Error(), "SECRET") || strings.Contains(err.Error(), "DO_NOT_ECHO") {
			t.Fatal(err)
		}
	}
}
