package runtime

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Sanidhya14321/goh/internal/core"
	"github.com/Sanidhya14321/goh/internal/session"
	"github.com/Sanidhya14321/goh/internal/testutil"
)

func listingAgent(t *testing.T, files int) *Agent {
	t.Helper()
	root := testutil.TempDir(t)
	for i := 0; i < files; i++ {
		os.WriteFile(filepath.Join(root, fmt.Sprintf("f%03d.go", i)), []byte("package f\n"), 0o644)
	}
	c := core.Defaults()
	c.SessionDir, c.ContextWindow, c.Model = t.TempDir(), 1<<20, "fixture"
	s, err := session.New(c.SessionDir, root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	a, err := New(c, s)
	if err != nil {
		t.Fatal(err)
	}
	return a
}

// The listing is sent after the system message on every request, unchanged,
// and only for a small project.
func TestProjectListing(t *testing.T) {
	for _, tc := range []struct {
		name  string
		files int
		want  bool
	}{{"small project", 3, true}, {"large project", listingMaxFiles + 1, false}} {
		t.Run(tc.name, func(t *testing.T) {
			a := listingAgent(t, tc.files)
			var seen []string
			calls := 0
			a.Provider = modelFunc(func(_ context.Context, h []core.Message, _ []core.Tool, _ func(string)) (core.Reply, error) {
				seen = append(seen, h[1].Content)
				calls++
				if calls == 1 {
					return answer(toolCall("1", "read_file", map[string]any{"path": "f000.go"})), nil
				}
				return answer(), nil
			})
			if _, err := a.Run(context.Background(), "go"); err != nil {
				t.Fatal(err)
			}
			has := strings.HasPrefix(seen[0], projectListingPrefix)
			if has != tc.want {
				t.Fatalf("listing sent %v, want %v: %.120q", has, tc.want, seen[0])
			}
			if has && (!strings.Contains(seen[0], "f002.go") || seen[1] != seen[0]) {
				t.Fatalf("listing incomplete or changed between requests: %q / %q", seen[0], seen[1])
			}
			for _, r := range a.Store.Records() {
				if strings.Contains(string(r.Data), "Files in this project when the session started") {
					t.Fatal("listing was journaled")
				}
			}
		})
	}
}
