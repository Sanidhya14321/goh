package provider

import (
	"errors"
	"strings"
	"testing"
)

// In-stream errors keep their code and a bounded message, so a usage limit
// is distinguishable from an access problem (found during the live benchmark).
func TestResponsesStreamErrorShowsCode(t *testing.T) {
	cases := map[string]string{
		`{"type":"error","code":"usage_limit","message":"limit reached, try later"}`:                 "(usage_limit): limit reached, try later",
		`{"type":"error","error":{"code":"rate_limit_exceeded","message":"slow down"}}`:              "(rate_limit_exceeded): slow down",
		`{"type":"response.failed","response":{"error":{"code":"server_error","message":"x\n  y"}}}`: "(server_error): x y",
		`{"type":"error"}`: "check model access and account usage",
	}
	for raw, want := range cases {
		if got := responsesStreamError(raw).Error(); !strings.Contains(got, want) {
			t.Fatalf("%s -> %s", raw, got)
		}
	}
	long := `{"type":"error","message":"` + strings.Repeat("a", 1000) + `"}`
	if len(responsesStreamError(long).Error()) > 400 {
		t.Fatal("message not bounded")
	}
}

func TestResponsesStreamErrorDropsCredentialShapedCodes(t *testing.T) {
	for _, secret := range []string{"sk-abcdefghijklmnopqrstu", "AKIAABCDEFGHIJKLMNOP", "ghp_abcdefghijklmnopqrstuvwxyz0123456789", "abcdefghijklmnopqrstuvwxyz0123", "tok:Zm9vYmFy"} {
		raw := `{"type":"error","code":"` + secret + `","message":"m"}`
		if got := responsesStreamError(raw).Error(); strings.Contains(got, secret) {
			t.Fatalf("code %q kept: %s", secret, got)
		}
	}
}

// An overloaded or failing service is marked ErrBusy so the turn is sent
// again after a pause (server_is_overloaded ended 5 of 19 benchmark runs);
// usage limits and request errors are not, since a resend cannot help.
func TestBusyStreamErrors(t *testing.T) {
	busy := map[string]error{
		"responses overloaded": responsesStreamError(`{"type":"response.failed","response":{"error":{"code":"server_is_overloaded","message":"Our servers are currently overloaded."}}}`),
		"responses server":     responsesStreamError(`{"type":"error","error":{"code":"server_error","message":"x"}}`),
		"chat overloaded":      streamError([]byte(`{"message":"m","type":"server_error","code":"overloaded"}`)),
	}
	_, anthropicErr := ParseAnthropic(strings.NewReader("event: error\ndata: {\"type\":\"error\",\"error\":{\"type\":\"overloaded_error\",\"message\":\"SECRET_ECHO\"}}\n\n"), nil)
	busy["anthropic overloaded"] = anthropicErr
	for name, err := range busy {
		if !errors.Is(err, ErrBusy) || strings.Contains(err.Error(), "SECRET_ECHO") {
			t.Errorf("%s: %v", name, err)
		}
	}
	if !strings.Contains(anthropicErr.Error(), "overloaded_error") {
		t.Errorf("anthropic error lost its type: %v", anthropicErr)
	}
	for _, raw := range []string{
		`{"type":"error","code":"subscription_sharing_usage_limit_exceeded","message":"limit"}`,
		`{"type":"error","code":"usage_limit","message":"limit"}`,
		`{"type":"error","error":{"code":"rate_limit_exceeded","message":"slow down"}}`,
		`{"type":"error"}`,
	} {
		if err := responsesStreamError(raw); errors.Is(err, ErrBusy) {
			t.Errorf("%s marked busy", raw)
		}
	}
}
