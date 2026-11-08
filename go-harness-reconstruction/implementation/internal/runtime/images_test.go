package runtime

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"example.com/independent-harness/internal/core"
)

var tinyPNG = []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR")

func TestPromptImagesAttachAndReachModel(t *testing.T) {
	a, root := testAgent(t)
	os.WriteFile(filepath.Join(root, "shot.png"), tinyPNG, 0o600)
	var seen []core.Message
	a.Provider = modelFunc(func(_ context.Context, h []core.Message, _ []core.Tool, _ func(string)) (core.Reply, error) {
		seen = h
		return answer(), nil
	})
	if status, err := a.Run(context.Background(), "what is wrong in @shot.png?"); status != "completed" || err != nil {
		t.Fatal(status, err)
	}
	var user core.Message
	for _, m := range seen {
		if m.Role == "user" {
			user = m
		}
	}
	if len(user.Images) != 1 || user.Images[0].Path != "shot.png" || user.Images[0].MediaType != "image/png" || user.Content != "what is wrong in @shot.png?" {
		t.Fatalf("%+v", user)
	}
	if estimate(a.History, nil) > 20000 {
		t.Fatal("image data counted as text in the context estimate")
	}
	plain, n := withoutImages(a.History)
	if n != 1 || !strings.Contains(plain[len(plain)-2].Content, "[attached images: shot.png]") || len(a.History[len(a.History)-2].Images) != 1 {
		t.Fatal("withoutImages wrong or mutated history")
	}
}

func TestPromptImagesRefusals(t *testing.T) {
	a, root := testAgent(t)
	os.WriteFile(filepath.Join(root, "fake.png"), []byte("not an image"), 0o600)
	os.WriteFile(filepath.Join(root, "big.png"), append(append([]byte{}, tinyPNG...), make([]byte, maxImageBytes)...), 0o600)
	outside := filepath.Join(t.TempDir(), "o.png")
	os.WriteFile(outside, tinyPNG, 0o600)
	cases := []string{"@fake.png", "@big.png", "@../o.png", "@missing.png", "@a.png @b.png @c.png @d.png @e.png"}
	if runtime.GOOS != "windows" {
		os.Symlink(outside, filepath.Join(root, "link.png"))
		cases = append(cases, "@link.png")
	}
	for _, prompt := range cases {
		if _, err := a.promptImages(prompt); err == nil {
			t.Fatal("accepted", prompt)
		}
	}
	if imgs, err := a.promptImages("email me@example.com about notes.txt"); err != nil || imgs != nil {
		t.Fatal("non-image mentions must be ignored", err)
	}
	if status, _ := a.Run(context.Background(), "look at @fake.png"); status != "failed" {
		t.Fatal("bad attachment should fail before any model call")
	}
}
