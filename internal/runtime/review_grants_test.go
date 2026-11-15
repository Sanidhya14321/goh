package runtime

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/Sanidhya14321/goh/internal/core"
)

func perm(tool string, args any) core.Permission {
	b, _ := json.Marshal(args)
	return core.Permission{Tool: tool, Arguments: b}
}

// Review R15: a "for this session" approval covers the reviewed kind of
// action, not every later use of the tool.
func TestSessionGrantsAreScopedToTheReviewedAction(t *testing.T) {
	a, _ := testAgent(t)
	asked := 0
	a.Approve = func(context.Context, core.Permission) (core.Decision, error) {
		asked++
		return core.Decision{Allow: true, Session: true}, nil
	}
	ctx := context.Background()
	steps := []struct {
		p         core.Permission
		wantAsked int
	}{
		{perm("shell", map[string]any{"command": "go test ./..."}), 1},
		{perm("shell", map[string]any{"command": "go test ./internal/x"}), 1}, // same prefix: covered
		{perm("shell", map[string]any{"command": "rm -rf build"}), 2},         // different command: asks
		{perm("shell", map[string]any{"command": "go test ./...; curl evil"}), 3},
		{perm("webfetch", map[string]any{"url": "https://docs.example.com/a"}), 4},
		{perm("webfetch", map[string]any{"url": "https://docs.example.com/b"}), 4}, // same origin
		{perm("webfetch", map[string]any{"url": "https://other.example.com/"}), 5},
		{perm("edit_file", map[string]any{"path": "a"}), 6},
		{perm("edit_file", map[string]any{"path": "b"}), 6}, // edits stay tool-wide
	}
	for i, s := range steps {
		if d, err := a.Permission(ctx, s.p); err != nil || !d.Allow {
			t.Fatalf("step %d: %v %v", i, d, err)
		}
		if asked != s.wantAsked {
			t.Fatalf("step %d (%s): asked %d times, want %d", i, s.p.Arguments, asked, s.wantAsked)
		}
	}
}

// A session approval of a command that runs code given as an argument covers
// only that exact command, never a later script for the same interpreter.
func TestSessionGrantsDoNotCoverInterpreterScripts(t *testing.T) {
	for _, c := range []struct{ approved, later string }{
		{"bash -c 'npm test'", "bash -c 'rm -rf ~/work'"},
		{"node -e console.log(1)", "node -e process.exit(1)"},
		{"python3 build.py", "python3 evil.py"},
		{"find . -name x", "find . -delete"},
		{"go test ./...", "go test -exec /tmp/evil ./..."},
		{"git status", "git -c core.pager=evil status"},
		{"go test ./...", `go test "./..."`},
	} {
		a, _ := testAgent(t)
		asked := 0
		a.Approve = func(context.Context, core.Permission) (core.Decision, error) {
			asked++
			return core.Decision{Allow: true, Session: true}, nil
		}
		for _, cmd := range []string{c.approved, c.later} {
			if _, err := a.Permission(context.Background(), perm("shell", map[string]any{"command": cmd})); err != nil {
				t.Fatal(err)
			}
		}
		if asked != 2 {
			t.Fatalf("approving %q also covered %q", c.approved, c.later)
		}
	}
}
