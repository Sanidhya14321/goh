//go:build windows

package tools

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// Native coverage for the cmd.exe shell path, which the POSIX-syntax shell
// tests skip on Windows.
func TestWindowsShellOutputExitAndQuoting(t *testing.T) {
	e, _ := engine(t)
	v := object(t, call(t, e, "shell", map[string]any{"command": `echo out& echo err 1>&2`}))
	out, _ := v["output"].(string)
	if v["exit_code"] != float64(0) || !strings.Contains(out, "out") || !strings.Contains(out, "err") {
		t.Fatal(v)
	}
	// Quotes reach cmd.exe unchanged (CmdLine is passed verbatim).
	v = object(t, call(t, e, "shell", map[string]any{"command": `echo "a b"`}))
	if out, _ := v["output"].(string); !strings.Contains(out, `"a b"`) {
		t.Fatal("quoting changed:", v)
	}
	r := call(t, e, "shell", map[string]any{"command": "exit /b 7"})
	var failed map[string]any
	json.Unmarshal([]byte(r.Content), &failed)
	if !r.Error || failed["exit_code"] != float64(7) {
		t.Fatal("nonzero exit not reported", r)
	}
}

func TestWindowsShellTimeoutStopsProcessTree(t *testing.T) {
	e, _ := engine(t)
	started := time.Now()
	// ping waits ~30s; the timeout must stop it and its cmd.exe parent.
	r := call(t, e, "shell", map[string]any{"command": "ping -n 30 127.0.0.1 >NUL", "timeout_seconds": 1})
	var v map[string]any
	json.Unmarshal([]byte(r.Content), &v)
	if !r.Error || v["status"] != "timed_out" {
		t.Fatal("expected timed_out", r)
	}
	if elapsed := time.Since(started); elapsed > 15*time.Second {
		t.Fatalf("timeout cleanup took %v", elapsed)
	}
}
