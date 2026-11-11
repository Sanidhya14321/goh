package tui

import "testing"

func TestHumanizers(t *testing.T) {
	for _, c := range []struct {
		got, want string
	}{
		{humanTokens(0), ""}, {humanTokens(512), "512 tokens"}, {humanTokens(12345), "12.3k tokens"},
		{humanTokens(2_500_000), "2.50M tokens"}, {humanDuration(0), ""}, {humanDuration(333), "333 ms"},
		{humanDuration(4200), "4.2s"}, {humanDuration(125_000), "2m 05s"},
	} {
		if c.got != c.want {
			t.Errorf("got %q want %q", c.got, c.want)
		}
	}
}

func TestStatusLineStates(t *testing.T) {
	m := &model{status: "completed", durationMS: 1500}
	if got := m.statusLine(); got != "completed  ·  1.5s" {
		t.Fatalf("idle: %q", got)
	}
	m = &model{active: true, pending: &permissionMsg{}}
	if got := m.statusLine(); got != "● Waiting for your approval" {
		t.Fatalf("approval: %q", got)
	}
}

func TestRelativizePreview(t *testing.T) {
	got := relativizePreview("--- /p/x/a.go\nDirectory: /p/x\nTimeout: 5s", "/p/x")
	if got != "--- a.go\nDirectory: . (project root)\nTimeout: 5s" {
		t.Fatalf("%q", got)
	}
}

func TestRelativizePreviewLeavesCommandsAndContentExact(t *testing.T) {
	in := "Command: cat /repo/file\nDirectory: /repo/sub\nTimeout: 5s"
	if got := relativizePreview(in, "/repo"); got != "Command: cat /repo/file\nDirectory: sub\nTimeout: 5s" {
		t.Fatalf("%q", got)
	}
	diff := "--- /repo/a.go\n+++ /repo/a.go\n-x := \"/repo/b\"\n+x := \"/repo/c\""
	if got := relativizePreview(diff, "/repo"); got != "--- a.go\n+++ a.go\n-x := \"/repo/b\"\n+x := \"/repo/c\"" {
		t.Fatalf("%q", got)
	}
	patch := "File 1: MOVE /repo/a.go -> /repo/b.go\nFile 1 before (-):\n-/repo/x"
	if got := relativizePreview(patch, "/repo"); got != "File 1: MOVE a.go -> b.go\nFile 1 before (-):\n-/repo/x" {
		t.Fatalf("%q", got)
	}
}

func TestShortPathKeepsTail(t *testing.T) {
	if got := shortPath("/very/long/path/to/project", 10); got != "…o/project" {
		t.Fatalf("%q", got)
	}
}
