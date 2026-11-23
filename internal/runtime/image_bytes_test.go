package runtime

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Sanidhya14321/goh/internal/core"
	"github.com/Sanidhya14321/goh/internal/provider"
)

func writePNG(t *testing.T, root, name string, size int) {
	t.Helper()
	b := make([]byte, size)
	copy(b, "\x89PNG\r\n\x1a\n")
	if err := os.WriteFile(filepath.Join(root, name), b, 0o644); err != nil {
		t.Fatal(err)
	}
}

func imageBytes(ms []core.Message) (n, count int) {
	for _, m := range ms {
		for _, img := range m.Images {
			n += len(img.Data)
			count++
		}
	}
	return
}

// Handoff P1: two turns that each attach two 3 MiB images fit the token
// window but not the provider's encoded request limit. The second request
// keeps the current images and drops older image data to a path note; the
// journal keeps every image.
func TestImageHistoryBoundedByRequestBytes(t *testing.T) {
	a, root := testAgent(t)
	for _, n := range []string{"a.png", "b.png", "c.png", "d.png"} {
		writePNG(t, root, n, 3<<20)
	}
	var sent [][]core.Message
	a.Provider = modelFunc(func(_ context.Context, h []core.Message, _ []core.Tool, _ func(string)) (core.Reply, error) {
		sent = append(sent, h)
		return answer(), nil
	})
	for _, prompt := range []string{"look at @a.png and @b.png", "now @c.png and @d.png"} {
		if status, err := a.Run(context.Background(), prompt); status != "completed" {
			t.Fatal(status, err)
		}
	}
	first, n1 := imageBytes(sent[0])
	second, n2 := imageBytes(sent[1])
	if n1 != 2 || n2 != 2 {
		t.Fatalf("images sent: %d then %d", n1, n2)
	}
	plain, _ := withoutImages(sent[1])
	b, _ := json.Marshal(plain)
	if second+len(b) > provider.MaxRequestBytes {
		t.Fatalf("second request %d bytes over the limit", second+len(b))
	}
	last := sent[1][len(sent[1])-1]
	if len(last.Images) != 2 || !strings.Contains(last.Content, "@c.png") {
		t.Fatal("current images not kept")
	}
	if !strings.Contains(joinContent(sent[1]), "a.png") {
		t.Fatal("dropped images left no path note")
	}
	_ = first
	// The journal and in-memory history keep all four images.
	if _, n := imageBytes(a.History); n != 4 {
		t.Fatalf("history images %d", n)
	}
}

// A small image history and a text-only conversation are sent unchanged.
func TestSmallImageHistoryUnchanged(t *testing.T) {
	a, root := testAgent(t)
	writePNG(t, root, "s.png", 64<<10)
	var sent [][]core.Message
	a.Provider = modelFunc(func(_ context.Context, h []core.Message, _ []core.Tool, _ func(string)) (core.Reply, error) {
		sent = append(sent, h)
		return answer(), nil
	})
	for _, prompt := range []string{"see @s.png", "and text only"} {
		if status, err := a.Run(context.Background(), prompt); status != "completed" {
			t.Fatal(status, err)
		}
	}
	if _, n := imageBytes(sent[1]); n != 1 {
		t.Fatalf("small history image dropped: %d", n)
	}
	history := []core.Message{{Role: "system", Content: "s"}, {Role: "user", Content: "hi"}}
	if out, dropped := boundImageHistory(history); dropped != 0 || &out[0] != &history[0] {
		t.Fatal("text-only history changed")
	}
}

// A current request that alone exceeds the limit gets an actionable error.
func TestOversizedRequestErrorIsActionable(t *testing.T) {
	c := core.Defaults()
	c.Endpoint = "http://127.0.0.1:1/v1/chat/completions"
	_, err := provider.New(c).Complete(context.Background(), []core.Message{{Role: "user", Content: strings.Repeat("x", provider.MaxRequestBytes)}}, nil, nil)
	if err == nil || !strings.Contains(err.Error(), "/compact") {
		t.Fatalf("error %v", err)
	}
}
