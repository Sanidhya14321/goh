package tools

import (
	"strings"
	"testing"

	"github.com/Sanidhya14321/goh/internal/platform"
	"github.com/Sanidhya14321/goh/internal/testutil"
)

// An approved (or injected) command must not be able to read provider keys
// or other secrets from GoHarness's own environment.
func TestShellDoesNotInheritSecrets(t *testing.T) {
	testutil.RequireUnixShell(t)
	t.Setenv("OPENAI_API_KEY", "sk-must-not-leak")
	t.Setenv("GITHUB_TOKEN", "ghp-must-not-leak")
	t.Setenv("PROJECT_FLAG", "visible")
	t.Cleanup(func() { platform.SetEnvPassthrough(nil) })
	platform.SetEnvPassthrough([]string{"PROJECT_*"})
	e, _ := engine(t)
	for _, tool := range []string{"shell"} {
		v := object(t, call(t, e, tool, map[string]any{"command": "env; command -v sh >/dev/null && echo PATH_OK"}))
		out := v["output"].(string)
		if strings.Contains(out, "must-not-leak") {
			t.Fatalf("%s leaked a secret: %s", tool, out)
		}
		if !strings.Contains(out, "PROJECT_FLAG=visible") || !strings.Contains(out, "PATH_OK") {
			t.Fatalf("%s lost the safe environment: %s", tool, out)
		}
	}
}
