package runtime

import (
	"context"
	"testing"

	"github.com/Sanidhya14321/goh/internal/core"
	"github.com/Sanidhya14321/goh/internal/testutil"
)

// Checks written as scripts, as models wrote them on Terminal-Bench, count
// when their exit status is the checks' status: an inline script whose body
// is a here-document, and a set -e script of checks. The body is input, not
// commands; under set -e only a final && chain can hide a failure.
func TestScriptChecksAreAttributable(t *testing.T) {
	for _, tc := range []struct {
		cmd  string
		want bool
	}{
		{"python3 -B - <<'PY'\nimport asyncio\nx = a || b; print(1)\nPY", true},
		{"python merge.py && python - <<'PY'\nimport csv\nassert True\nPY", true},
		{"set -eu\nnginx -t\ntest ! -e /etc/nginx/sites-enabled/default\n\npython3 - <<'PY'\nprint(1)\nPY\n", true},
		{"set -euo pipefail\ntest -f a.txt\ncmp a.txt b.txt", true},
		{"set -e\npython3 check.py &&\n  test -f out.txt", true},
		{"cat <<-EOF > in.txt\n\thello\n\tEOF\npython3 t.py", true},
		{"set -e\npython3 check.py && test -f x\necho done", false}, // && before a later line
		{"python3 check.py\necho done", false},                      // no set -e: echo's status
		{"set -e\npython3 check.py || true", false},
		{"set -x\npython3 check.py\necho done", false},
		{"python3 - <<'PY\nprint(1)\nPY", false}, // open quote
	} {
		if got := checkStatusAttributable(tc.cmd); got != tc.want {
			t.Errorf("checkStatusAttributable(%q) = %v, want %v", tc.cmd, got, tc.want)
		}
	}
	if kind := checkKind("python3 -B - <<'PY'\nfrom run import run_tasks\nPY"); kind != "run" {
		t.Errorf("inline script kind %q", kind)
	}
}

func TestInlineScriptCheckVerifiesARun(t *testing.T) {
	testutil.RequireUnixShell(t)
	a, _ := testAgent(t)
	a.Grants["create_file"], a.Grants["shell"] = true, true
	a.Provider = scripted(
		[]core.Call{toolCall("c", "create_file", map[string]any{"path": "out.txt", "content": "42\n"})},
		[]core.Call{toolCall("t", "shell", map[string]any{"command": "set -eu\ntest -s out.txt\nsh - <<'SH'\ngrep -qx 42 out.txt\nSH"})},
	)
	a.Run(context.Background(), "go")
	if got := lastFinished(t, a)["outcome"]; got != OutcomeVerified {
		t.Fatalf("outcome %v", got)
	}
}
