package provider

import (
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
