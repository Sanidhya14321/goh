package platform

import (
	"slices"
	"testing"
)

func TestFilterEnv(t *testing.T) {
	environ := []string{
		"PATH=/bin", "HOME=/h", "LC_ALL=C", "XDG_CONFIG_HOME=/x", "GOPATH=/g",
		"OPENAI_API_KEY=sk-secret", "ANTHROPIC_API_KEY=sk-ant", "AWS_SECRET_ACCESS_KEY=a",
		"GITHUB_TOKEN=ghp", "DATABASE_URL=postgres://u:p@h/db", "MY_APP_MODE=dev", "HTTPS_PROXY=http://u:p@proxy",
		"Path=C:\\w", "=C:=C:\\", "BROKEN",
	}
	t.Cleanup(func() { SetEnvPassthrough(nil) })
	SetEnvPassthrough(nil)
	got := filterEnv(environ)
	want := []string{"PATH=/bin", "HOME=/h", "LC_ALL=C", "XDG_CONFIG_HOME=/x", "GOPATH=/g", "Path=C:\\w"}
	if !slices.Equal(got, want) {
		t.Fatalf("default env %q", got)
	}
	SetEnvPassthrough([]string{"DATABASE_URL", "my_app_*"})
	got = filterEnv(environ)
	if !slices.Contains(got, "DATABASE_URL=postgres://u:p@h/db") || !slices.Contains(got, "MY_APP_MODE=dev") || slices.Contains(got, "OPENAI_API_KEY=sk-secret") {
		t.Fatalf("passthrough env %q", got)
	}
	SetEnvPassthrough([]string{"*"})
	if got = filterEnv(environ); !slices.Contains(got, "OPENAI_API_KEY=sk-secret") {
		t.Fatalf("explicit inherit-all %q", got)
	}
}
