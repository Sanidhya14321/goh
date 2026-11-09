package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/Sanidhya14321/goh/internal/core"
)

func TestAuthListNamesSourcesNotSecrets(t *testing.T) {
	for _, k := range []string{"HOME", "XDG_CONFIG_HOME", "XDG_STATE_HOME", "USERPROFILE", "APPDATA", "LOCALAPPDATA"} {
		t.Setenv(k, t.TempDir())
	}
	t.Setenv("OPENAI_API_KEY", "sk-official-secret-1111")
	t.Setenv("GW_KEY", "gw-secret-2222")
	s := func(v string) *string { return &v }
	c := core.Defaults()
	c.Protocol, c.Endpoint, c.Model = "chat_completions", "https://api.openai.com/v1/chat/completions", "m"
	c.Profiles = map[string]core.ModelProfile{
		"gateway":   {Protocol: s("chat_completions"), Endpoint: s("https://gw.example/v1/chat/completions"), APIKeyEnv: s("GW_KEY")},
		"untrusted": {Protocol: s("chat_completions"), Endpoint: s("https://other.example/v1/chat/completions")},
	}
	var b bytes.Buffer
	if err := writeAuthList(&b, c); err != nil {
		t.Fatal(err)
	}
	out := b.String()
	for _, secret := range []string{"sk-official-secret-1111", "gw-secret-2222"} {
		if strings.Contains(out, secret) {
			t.Fatal("secret printed:", out)
		}
	}
	want := map[string]string{"default": "environment OPENAI_API_KEY", "gateway": "environment GW_KEY", "untrusted": "none", "chatgpt": "ChatGPT not signed in"}
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		f := strings.Fields(line)
		if w, ok := want[f[0]]; ok && !strings.HasSuffix(line, w) {
			t.Errorf("%s: %q, want %q", f[0], line, w)
		}
		delete(want, f[0])
	}
	if len(want) != 0 {
		t.Fatal("missing rows", want, out)
	}
}
