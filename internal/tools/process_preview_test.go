package tools

import (
	"fmt"
	"strings"
	"testing"
)

// Long output keeps its end in the preview: that is where test summaries
// and compiler errors are.
func TestShellPreviewKeepsHeadAndTail(t *testing.T) {
	w := &captureWriter{file: nil, err: fmt.Errorf("no artifact file in this test")}
	for i := 0; i < 20000; i++ {
		fmt.Fprintf(w, "line %05d\n", i)
	}
	fmt.Fprint(w, "FAIL: TestMonthly (expected 2025-09)\n")
	text := w.text("shell-1.log")
	for _, want := range []string{"line 00000\n", "line 19999\n", "FAIL: TestMonthly", "bytes omitted; read_artifact shell-1.log"} {
		if !strings.Contains(text, want) {
			t.Fatalf("preview lacks %q", want)
		}
	}
	if len(text) > previewHead+previewTail+200 {
		t.Fatal("preview exceeds its budget", len(text))
	}
	if !strings.Contains(text, "…]\nline ") {
		t.Fatal("tail should start on a whole line")
	}
	short := &captureWriter{err: fmt.Errorf("none")}
	fmt.Fprint(short, "ok\n")
	if got := short.text("x"); got != "ok\n" {
		t.Fatalf("%q", got)
	}
}
