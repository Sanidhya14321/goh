package session

import (
	"strings"
	"testing"
)

func TestRedactSecrets(t *testing.T) {
	in := "key sk-proj-abcdefghijklmnopqrstuvwx1234 gh ghp_abcdefghijklmnopqrstuvwxyz0123456789 aws AKIAABCDEFGHIJKLMNOP\n" +
		"Authorization: Bearer eyJhbGciOiJIUzI1NiJ9.payload.sig\n-----BEGIN RSA PRIVATE KEY-----\nMIIE\n-----END RSA PRIVATE KEY-----\nskip-this sk-short"
	out := RedactSecrets(in)
	for _, secret := range []string{"abcdefghijklmnopqrstuvwx1234", "abcdefghijklmnopqrstuvwxyz0123456789", "ABCDEFGHIJKLMNOP", "eyJhbGci", "MIIE"} {
		if strings.Contains(out, secret) {
			t.Fatalf("%q leaked:\n%s", secret, out)
		}
	}
	for _, kept := range []string{"sk-proj-[redacted]", "ghp_[redacted]", "Bearer [redacted]", "[redacted private key]", "skip-this sk-short"} {
		if !strings.Contains(out, kept) {
			t.Fatalf("missing %q:\n%s", kept, out)
		}
	}
}
