package sandbox

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestMain(m *testing.M) {
	HelperMain() // the test binary doubles as the Linux helper
	os.Exit(m.Run())
}

func sh(t *testing.T, p *Policy, dir, script string) (string, error) {
	t.Helper()
	cmd := exec.Command("/bin/sh", "-c", script)
	cmd.Dir = dir
	if err := Wrap(cmd, p); err != nil {
		return "", err
	}
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func TestSandboxConfinesWritesReadsAndNetwork(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		cmd := exec.Command("cmd")
		if err := Wrap(cmd, &Policy{}); !errors.Is(err, ErrUnsupported) {
			t.Fatal("unsupported system did not refuse:", err)
		}
		return
	}
	// Writes to a sibling of the project must fail even inside temp space.
	base := t.TempDir()
	project := filepath.Join(base, "project")
	secret := filepath.Join(base, "secret")
	outside := filepath.Join(base, "outside")
	for _, d := range []string{project, secret, outside} {
		os.MkdirAll(d, 0o755)
	}
	os.WriteFile(filepath.Join(secret, "token"), []byte("tok"), 0o600)
	os.WriteFile(filepath.Join(outside, "readme"), []byte("public"), 0o644)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.Write([]byte("net-ok")) }))
	defer srv.Close()

	p := &Policy{Writable: []string{project}, Protected: []string{secret}}
	script := `echo in > inside.txt && echo WROTE_PROJECT
echo x > ` + outside + `/escape.txt 2>/dev/null && echo ESCAPED || echo BLOCKED_WRITE
cat ` + secret + `/token 2>/dev/null && echo LEAKED || echo BLOCKED_READ
cat ` + outside + `/readme && echo
ls ` + base + ` >/dev/null && echo LISTED_PARENT
curl -s -m 5 ` + srv.URL + ` 2>/dev/null || echo BLOCKED_NET`
	out, err := sh(t, p, project, script)
	if err != nil && strings.Contains(err.Error(), "Landlock") {
		if os.Getenv("GOH_REQUIRE_SANDBOX") == "1" {
			t.Fatal("sandbox required but unavailable:", err)
		}
		t.Skip("kernel lacks Landlock support:", err)
	}
	for _, want := range []string{"WROTE_PROJECT", "BLOCKED_WRITE", "BLOCKED_READ", "public", "LISTED_PARENT", "BLOCKED_NET"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %s in sandboxed output:\n%s (err %v)", want, out, err)
		}
	}
	for _, bad := range []string{"ESCAPED", "LEAKED", "net-ok"} {
		if strings.Contains(out, bad) {
			t.Fatalf("sandbox let %s through:\n%s", bad, out)
		}
	}
	if _, err := os.Stat(filepath.Join(outside, "escape.txt")); err == nil {
		t.Fatal("file written outside the project")
	}
	// With network allowed, the same request succeeds.
	p.Network = true
	if out, _ := sh(t, p, project, "curl -s -m 5 "+srv.URL); !strings.Contains(out, "net-ok") {
		t.Fatalf("network allowed but blocked: %s", out)
	}
}

// Git runs hooks outside the sandbox, so a sandboxed command must not be
// able to plant one, while the rest of .git stays writable for git itself.
func TestSandboxReadOnlyPathsInsideProject(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("ReadOnly is enforced by Seatbelt only")
	}
	project := t.TempDir()
	os.MkdirAll(filepath.Join(project, ".git", "hooks"), 0o755)
	os.WriteFile(filepath.Join(project, ".git", "config"), []byte("[core]\n"), 0o644)
	hooks, config := filepath.Join(project, ".git", "hooks"), filepath.Join(project, ".git", "config")
	p := &Policy{Writable: []string{project}, ReadOnly: []string{hooks, config}}
	out, _ := sh(t, p, project, `echo x > .git/hooks/pre-commit 2>/dev/null && echo HOOK_WRITTEN || echo HOOK_BLOCKED
echo x >> .git/config 2>/dev/null && echo CONFIG_WRITTEN || echo CONFIG_BLOCKED
echo x > .git/index && echo INDEX_OK
cat .git/config`)
	for _, want := range []string{"HOOK_BLOCKED", "CONFIG_BLOCKED", "INDEX_OK", "[core]"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %s:\n%s", want, out)
		}
	}
}

// Caches tools keep outside the writable folders (Go's build cache, npm,
// pip) point inside the temporary area for sandboxed commands, so a cache
// write succeeds; a value the user set is kept, and the folder is private.
func TestSandboxedCommandsGetWritableCaches(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("no native sandbox")
	}
	project := t.TempDir()
	p := &Policy{Writable: append([]string{project}, TempDirs()...)}
	cmd := exec.Command("/bin/sh", "-c", `mkdir -p "$GOCACHE" && echo x > "$GOCACHE/probe" && echo CACHE_OK "$GOCACHE" "$PIP_CACHE_DIR" "$KEEP"`)
	cmd.Dir = project
	cmd.Env = []string{"PATH=/usr/bin:/bin", "KEEP=1", "npm_config_cache=/mine"}
	if err := Wrap(cmd, p); err != nil {
		t.Fatal(err)
	}
	out, err := cmd.CombinedOutput()
	if err != nil && strings.Contains(string(out), "Landlock") {
		t.Skip("Landlock unavailable")
	}
	if err != nil || !strings.Contains(string(out), "CACHE_OK") || !strings.Contains(string(out), "goh-sandbox-cache-") || !strings.HasSuffix(strings.TrimSpace(string(out)), " 1") {
		t.Fatalf("%v: %s", err, out)
	}
	kept := false
	for _, kv := range cmd.Env {
		kept = kept || kv == "npm_config_cache=/mine"
	}
	if !kept {
		t.Fatal("user's npm_config_cache replaced")
	}
	fields := strings.Fields(string(out))
	if info, err := os.Lstat(filepath.Dir(fields[1])); err != nil || info.Mode().Perm() != 0o700 {
		t.Fatalf("cache folder not private: %v %v", info, err)
	}
}
