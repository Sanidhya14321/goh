package lspclient

import (
	"encoding/json"
	"testing"
)

// Review P2: Windows drive and UNC paths keep their identity in file URIs.
func TestFileURIForWindows(t *testing.T) {
	for in, want := range map[string]string{
		`C:\work\app\main.go`:    "file:///C:/work/app/main.go",
		`\\server\share\x\a.go`:  "file://server/share/x/a.go",
		`C:\dir with space\a.go`: "file:///C:/dir%20with%20space/a.go",
	} {
		if got := fileURIFor(in, true); got != want {
			t.Errorf("%s -> %s, want %s", in, got, want)
		}
	}
	if got := fileURIFor("/home/u/a b.go", false); got != "file:///home/u/a%20b.go" {
		t.Error(got)
	}
}

// Review P2: updates follow the negotiated document sync.
func TestContentChangeFollowsSyncKind(t *testing.T) {
	caps := func(raw string) map[string]json.RawMessage {
		return map[string]json.RawMessage{"textDocumentSync": json.RawMessage(raw)}
	}
	if syncKind(caps(`2`)) != 2 || syncKind(caps(`{"change":2,"openClose":true}`)) != 2 || syncKind(caps(`0`)) != 0 || syncKind(nil) != 1 {
		t.Fatal("sync kind")
	}
	c := contentChange(2, "a\nbé😀", "new")
	r := c["range"].(map[string]any)
	if end := r["end"].(map[string]int); end["line"] != 1 || end["character"] != 4 || c["text"] != "new" {
		t.Fatal(c)
	}
	if _, ok := contentChange(1, "a", "b")["range"]; ok {
		t.Fatal("full sync got a range")
	}
}
