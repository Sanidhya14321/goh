//go:build windows

package tools

import (
	"os"
	"strings"
	"testing"

	"github.com/Sanidhya14321/goh/internal/platform"
)

// Temporary CI diagnostic: why PowerShell children fail with ChildEnv.
func TestDiagPowerShellChildEnv(t *testing.T) {
	kept := map[string]bool{}
	for _, kv := range platform.ChildEnv() {
		kept[strings.SplitN(kv, "=", 2)[0]] = true
	}
	var dropped []string
	for _, kv := range os.Environ() {
		name := strings.SplitN(kv, "=", 2)[0]
		if !kept[name] {
			dropped = append(dropped, name)
		}
	}
	t.Logf("dropped names: %v", dropped)
	e, _ := engine(t)
	r := call(t, e, "shell", map[string]any{"command": `powershell -NoProfile -Command "Write-Output ok-$PID"`, "timeout_seconds": 60})
	t.Logf("result error=%v content=%s", r.Error, r.Content)
	t.Fail()
}
