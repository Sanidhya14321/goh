package tools

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Sanidhya14321/goh/internal/sandbox"
	"github.com/Sanidhya14321/goh/internal/testutil"
)

func TestMain(m *testing.M) {
	sandbox.HelperMain() // the test binary doubles as the Linux helper
	os.Exit(m.Run())
}

func TestShellToolRunsInsideSandbox(t *testing.T) {
	testutil.RequireUnixShell(t)
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("no sandbox on this system")
	}
	e, root := engine(t)
	outside := t.TempDir()
	e.Sandbox = &sandbox.Policy{Writable: []string{root}, Network: true}
	v := object(t, call(t, e, "shell", map[string]any{"command": "echo ok > in.txt; echo x > " + filepath.Join(outside, "out.txt") + " 2>&1; echo done"}))
	out, _ := v["output"].(string)
	if strings.Contains(out, "Landlock") {
		if os.Getenv("GOH_REQUIRE_SANDBOX") == "1" {
			t.Fatal("sandbox required but unavailable:", out)
		}
		t.Skip("kernel lacks Landlock:", out)
	}
	if !strings.Contains(out, "done") {
		t.Fatalf("sandboxed shell output %v", v)
	}
	if _, err := os.Stat(filepath.Join(root, "in.txt")); err != nil {
		t.Fatal("project write blocked")
	}
	if _, err := os.Stat(filepath.Join(outside, "out.txt")); err == nil {
		t.Fatal("sandboxed shell wrote outside the project")
	}
}
