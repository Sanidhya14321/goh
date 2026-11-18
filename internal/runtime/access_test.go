package runtime

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/Sanidhya14321/goh/internal/core"
	"github.com/Sanidhya14321/goh/internal/sandbox"
	"github.com/Sanidhya14321/goh/internal/session"
)

// accessHarness returns a permission probe that reports whether a request
// ran without asking, and the folder a command was confined to.
func accessHarness(t *testing.T) (*Agent, string, func(tool string, args any) (asked bool, confine string)) {
	t.Helper()
	a, root := testAgent(t)
	if r, err := filepath.EvalSymlinks(root); err == nil {
		root = r
	}
	asked := false
	a.Approve = func(context.Context, core.Permission) (core.Decision, error) {
		asked = true
		return core.Decision{Allow: true}, nil
	}
	return a, root, func(tool string, args any) (bool, string) {
		t.Helper()
		asked = false
		b, _ := json.Marshal(args)
		d, err := a.Permission(context.Background(), core.Permission{Tool: tool, Arguments: b})
		if err != nil || !d.Allow {
			t.Fatalf("%s: %+v %v", tool, d, err)
		}
		return asked, d.Confine
	}
}

func TestAccessLevels(t *testing.T) {
	saved := sandboxSupported
	t.Cleanup(func() { sandboxSupported = saved })
	sandboxSupported = func() bool { return true }
	edit := map[string]string{"path": "src/a.go", "content": "x"}
	shell := map[string]string{"command": "npm test"}
	fetch := map[string]string{"url": "https://example.com"}
	for _, tc := range []struct {
		level                       core.AccessLevel
		askEdit, askShell, askFetch bool
		sandboxed                   bool
	}{
		{core.AccessAsk, true, true, true, false},
		{core.AccessEdits, false, true, true, false},
		{core.AccessAuto, false, false, true, true},
		{core.AccessFull, false, false, false, false},
	} {
		a, root, probe := accessHarness(t)
		a.SetSessionAccess(tc.level)
		if asked, _ := probe("create_file", edit); asked != tc.askEdit {
			t.Errorf("%s: edit asked=%v", tc.level, asked)
		}
		asked, confine := probe("shell", shell)
		if asked != tc.askShell || (confine == root) != tc.sandboxed {
			t.Errorf("%s: shell asked=%v confine=%q", tc.level, asked, confine)
		}
		if asked, _ := probe("webfetch", fetch); asked != tc.askFetch {
			t.Errorf("%s: webfetch asked=%v", tc.level, asked)
		}
		if asked, _ := probe("child:create_file", edit); asked != tc.askEdit {
			t.Errorf("%s: a subagent's edit asked=%v", tc.level, asked)
		}
		if got := a.ProjectAccess(); got != tc.level {
			t.Errorf("project access %s, want %s", got, tc.level)
		}
	}
}

// A level saved for one folder covers only that folder; the deepest saved
// folder wins, and the session level replaces only the project's own.
func TestAccessIsScopedToFolders(t *testing.T) {
	saved := sandboxSupported
	t.Cleanup(func() { sandboxSupported = saved })
	sandboxSupported = func() bool { return true }
	a, root, probe := accessHarness(t)
	os.MkdirAll(filepath.Join(root, "web", "vendor"), 0o755)
	a.SetAccess([]core.FolderAccess{
		{Path: filepath.Join(root, "web"), Level: core.AccessAuto},
		{Path: filepath.Join(root, "web", "vendor"), Level: core.AccessAsk},
	})
	if asked, _ := probe("create_file", map[string]string{"path": "web/app.js", "content": "x"}); asked {
		t.Error("edit inside the auto folder asked")
	}
	if asked, _ := probe("create_file", map[string]string{"path": "main.go", "content": "x"}); !asked {
		t.Error("edit outside the folder did not ask")
	}
	if asked, _ := probe("create_file", map[string]string{"path": "web/vendor/lib.js", "content": "x"}); !asked {
		t.Error("edit in a deeper ask folder did not ask")
	}
	patch := "*** Begin Patch\n*** Add File: web/a.js\n+x\n*** Add File: root.txt\n+y\n*** End Patch"
	if asked, _ := probe("apply_patch", map[string]string{"patchText": patch}); !asked {
		t.Error("a patch reaching outside the folder did not ask")
	}
	if asked, confine := probe("shell", map[string]string{"command": "npm test", "cwd": "web"}); asked || confine != filepath.Join(root, "web") {
		t.Errorf("command in the auto folder: asked=%v confine=%q", asked, confine)
	}
	if asked, _ := probe("shell", map[string]string{"command": "npm test"}); !asked {
		t.Error("command at the project root did not ask")
	}
	a.SetSessionAccess(core.AccessEdits)
	if asked, _ := probe("create_file", map[string]string{"path": "main.go", "content": "x"}); asked {
		t.Error("session level did not apply at the root")
	}
	if asked, _ := probe("create_file", map[string]string{"path": "web/vendor/lib.js", "content": "x"}); !asked {
		t.Error("session level overrode a deeper saved folder")
	}
}

// Without a sandbox, auto never runs commands unasked; explicit ask and deny
// rules win over every level; the loop guard always asks.
func TestAccessKeepsSafetyRules(t *testing.T) {
	saved := sandboxSupported
	t.Cleanup(func() { sandboxSupported = saved })
	sandboxSupported = func() bool { return false }
	a, _, probe := accessHarness(t)
	a.SetSessionAccess(core.AccessAuto)
	if asked, _ := probe("shell", map[string]string{"command": "npm test"}); !asked {
		t.Error("auto ran a command without a sandbox")
	}
	a.SetSessionAccess(core.AccessFull)
	a.Config.FileRules = []core.ShellRule{{Match: "go.mod", Action: "ask"}, {Match: "*.env", Action: "deny"}}
	if asked, _ := probe("create_file", map[string]string{"path": "go.mod", "content": "x"}); !asked {
		t.Error("an ask rule did not ask under full access")
	}
	b, _ := json.Marshal(map[string]string{"path": "prod.env", "content": "x"})
	if d, _ := a.Permission(context.Background(), core.Permission{Tool: "create_file", Arguments: b}); d.Allow {
		t.Error("a deny rule allowed under full access")
	}
	if asked, _ := probe("repeated_call", map[string]string{}); !asked {
		t.Error("the repeat guard did not ask")
	}
}

// A command allowed by auto really is confined: it can write in its folder
// and cannot write next to it.
func TestAutoAccessCommandsAreSandboxed(t *testing.T) {
	if !sandbox.Supported() {
		t.Skip("no sandbox on this system")
	}
	// Temporary folders are always writable inside the sandbox, so this
	// project must live outside them.
	wd, _ := os.Getwd()
	root, err := os.MkdirTemp(wd, ".access-test-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(root) })
	if r, err := filepath.EvalSymlinks(root); err == nil {
		root = r
	}
	os.MkdirAll(filepath.Join(root, "web"), 0o755)
	c := core.Defaults()
	c.SessionDir = t.TempDir()
	c.ContextWindow = 1 << 20
	c.Model = "fixture"
	off := false
	c.CompletionCheck = &off
	s, err := session.New(c.SessionDir, root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	a, err := New(c, s)
	if err != nil {
		t.Fatal(err)
	}
	a.SetAccess([]core.FolderAccess{{Path: filepath.Join(root, "web"), Level: core.AccessAuto}})
	calls := []core.Call{
		toolCall("c1", "shell", map[string]string{"command": "echo in > inside.txt", "cwd": "web"}),
		toolCall("c2", "shell", map[string]string{"command": "echo out > ../outside.txt", "cwd": "web"}),
	}
	turn := 0
	a.Provider = modelFunc(func(context.Context, []core.Message, []core.Tool, func(string)) (core.Reply, error) {
		turn++
		if turn <= len(calls) {
			return answer(calls[turn-1]), nil
		}
		return answer(), nil
	})
	if status, err := a.Run(context.Background(), "go"); status != "completed" {
		t.Fatal(status, err)
	}
	if _, err := os.Stat(filepath.Join(root, "web", "inside.txt")); err != nil {
		t.Error("write inside the folder failed:", err)
	}
	if _, err := os.Stat(filepath.Join(root, "outside.txt")); err == nil {
		t.Error("a command allowed by auto wrote outside its folder")
	}
}

// Auto-edit also runs commands that only read inside the project; Ask does
// not, and a reading command that leaves the project still asks.
func TestAutoEditRunsReadOnlyCommands(t *testing.T) {
	a, _, probe := accessHarness(t)
	look := map[string]string{"command": "pwd; ls -d web 2>/dev/null; command -v node"}
	if asked, _ := probe("shell", look); !asked {
		t.Error("ask level ran a command unasked")
	}
	a.SetSessionAccess(core.AccessEdits)
	if asked, confine := probe("shell", look); asked || confine != "" {
		t.Errorf("read-only command asked=%v confine=%q", asked, confine)
	}
	if asked, _ := probe("shell", map[string]string{"command": "ls /Applications"}); !asked {
		t.Error("a command reading outside the project ran unasked")
	}
	if asked, _ := probe("shell", map[string]string{"command": "npm install"}); !asked {
		t.Error("a writing command ran unasked under auto-edit")
	}
}
