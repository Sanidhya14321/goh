package runtime

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Sanidhya14321/goh/internal/core"
)

// pngBytes is the smallest header that passes the image magic check.
var pngBytes = []byte("\x89PNG\r\n\x1a\n0000")

func TestAttachmentsHonourFileRulesInEverySpelling(t *testing.T) {
	a, root := testAgent(t)
	a.Config.FileRules = []core.ShellRule{{Match: "private/*", Action: "deny"}}
	os.Mkdir(filepath.Join(root, "private"), 0o700)
	os.WriteFile(filepath.Join(root, "private", "key.txt"), []byte("SECRET\n"), 0o600)
	os.WriteFile(filepath.Join(root, "private", "shot.png"), pngBytes, 0o600)
	os.Symlink("private", filepath.Join(root, "link"))
	os.WriteFile(filepath.Join(root, "notes.txt"), []byte("see @private/key.txt and @private/shot.png\n"), 0o600)

	prompts := []string{"@private/key.txt", "@link/key.txt"}
	if _, err := os.Stat(filepath.Join(root, "PRIVATE", "key.txt")); err == nil {
		prompts = append(prompts, "@PRIVATE/KEY.TXT") // case-insensitive filesystem
	}
	for _, p := range prompts {
		if got, files := a.withFileMentions("read " + p); len(files) != 0 || strings.Contains(got, "SECRET") {
			t.Fatalf("%s attached %v", p, files)
		}
	}
	for _, p := range []string{"@private/shot.png", "@link/shot.png"} {
		if imgs, err := a.promptImages("look " + p); err == nil || len(imgs) != 0 {
			t.Fatalf("%s: images %d err %v", p, len(imgs), err)
		}
	}
	// Mentions inside an attached file are content, not requests.
	got, files := a.withFileMentions("summarise @notes.txt")
	if strings.Join(files, ",") != "notes.txt" || strings.Contains(got, "SECRET") {
		t.Fatalf("files %v", files)
	}
	if imgs, err := a.promptImages(got); err != nil || len(imgs) != 0 {
		t.Fatalf("image from attached content: %d %v", len(imgs), err)
	}
}
