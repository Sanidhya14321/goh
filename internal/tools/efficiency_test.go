package tools

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/Sanidhya14321/goh/internal/testutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func decodeEfficiency(t testing.TB, r Result) map[string]any {
	t.Helper()
	if r.Error {
		t.Fatal(r.Content)
	}
	var out map[string]any
	if err := json.Unmarshal([]byte(r.Content), &out); err != nil {
		t.Fatal(err)
	}
	return out
}
func TestArtifactPagingTailExactBytesAndOwnership(t *testing.T) {
	root, artifacts := t.TempDir(), t.TempDir()
	e, err := New(root, artifacts)
	if err != nil {
		t.Fatal(err)
	}
	raw := []byte("begin😀middle\x00\x01end\xff")
	os.WriteFile(filepath.Join(artifacts, "shell-12345.log"), raw, 0600)
	got := []byte{}
	offset := 0
	for {
		args, _ := json.Marshal(map[string]any{"id": "shell-12345.log", "offset": offset, "limit": 3, "encoding": "base64"})
		result := decodeEfficiency(t, e.Execute(context.Background(), "read_artifact", args))
		part, err := base64.StdEncoding.DecodeString(result["content"].(string))
		if err != nil {
			t.Fatal(err)
		}
		got = append(got, part...)
		offset = int(result["next_offset"].(float64))
		if result["sha256"] != hash(raw) {
			t.Fatal("artifact identity lost")
		}
		if result["eof"] == true {
			break
		}
	}
	if string(got) != string(raw) {
		t.Fatal(got, raw)
	}
	out := decodeEfficiency(t, e.Execute(context.Background(), "read_artifact", json.RawMessage(`{"id":"shell-12345.log","tail":true,"limit":4,"encoding":"base64"}`)))
	tail, _ := base64.StdEncoding.DecodeString(out["content"].(string))
	if string(tail) != string(raw[len(raw)-4:]) {
		t.Fatal(out)
	}
	out = decodeEfficiency(t, e.Execute(context.Background(), "read_artifact", json.RawMessage(`{"id":"shell-12345.log","tail":true,"limit":1}`)))
	if out["utf8_repaired"] != true {
		t.Fatal(out)
	}
	other, _ := New(root, t.TempDir())
	if result := other.Execute(context.Background(), "read_artifact", json.RawMessage(`{"id":"shell-12345.log"}`)); !result.Error {
		t.Fatal("foreign-session output exposed")
	}
	for _, args := range []string{`{"id":"../shell-12345.log"}`, `{"id":"/etc/passwd"}`, `{"id":"shell-12345.log","offset":1000}`, `{"id":"shell-12345.log","tail":true,"offset":1}`, `{"id":"shell-12345.log","limit":0}`, `{"id":"shell-12345.log","limit":8193}`, `{"id":null}`, `{"id":"shell-12345.log","path":"secret"}`, `{"id":"shell-12345.log"} {}`} {
		if r := e.Execute(context.Background(), "read_artifact", json.RawMessage(args)); !r.Error {
			t.Fatal("invalid artifact args admitted", args)
		}
	}
	os.Symlink(filepath.Join(artifacts, "shell-12345.log"), filepath.Join(artifacts, "shell-222.log"))
	if r := e.Execute(context.Background(), "read_artifact", json.RawMessage(`{"id":"shell-222.log"}`)); !r.Error {
		t.Fatal("artifact link read")
	}
	if err = testutil.Mkfifo(t, filepath.Join(artifacts, "shell-333.log")); err != nil {
		t.Fatal(err)
	}
	if r := e.Execute(context.Background(), "read_artifact", json.RawMessage(`{"id":"shell-333.log"}`)); !r.Error {
		t.Fatal("artifact FIFO read")
	}
	os.WriteFile(filepath.Join(artifacts, "shell-444.log"), []byte(strings.Repeat("x", maxArtifact+1)), 0600)
	if r := e.Execute(context.Background(), "read_artifact", json.RawMessage(`{"id":"shell-444.log"}`)); !r.Error {
		t.Fatal("oversized artifact read")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if r := e.Execute(ctx, "read_artifact", json.RawMessage(`{"id":"shell-12345.log"}`)); !r.Error {
		t.Fatal("cancel ignored")
	}
}
func TestShellArtifactIdentityAndBoundedEscaping(t *testing.T) {
	e, err := New(t.TempDir(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	out := decodeEfficiency(t, e.Execute(context.Background(), "shell", json.RawMessage(`{"command":"printf 'first\\nlast\\n'","timeout_seconds":3}`)))
	id, ok := out["artifact_id"].(string)
	if !ok || !artifactName.MatchString(id) {
		t.Fatal(out)
	}
	args, _ := json.Marshal(map[string]any{"id": id, "tail": true, "limit": 5})
	page := decodeEfficiency(t, e.Execute(context.Background(), "read_artifact", args))
	if page["content"] != "last\n" {
		t.Fatal(page)
	}
	os.WriteFile(filepath.Join(e.artifacts, "shell-999.log"), []byte(strings.Repeat("\x00", 8192)), 0600)
	r := e.Execute(context.Background(), "read_artifact", json.RawMessage(`{"id":"shell-999.log","limit":8192}`))
	if r.Error || len(r.Content) > MaxInline {
		t.Fatal("escaped page exceeds result bound", len(r.Content), r.Error)
	}
}
func TestRepoMapDeclarationsCacheInvalidationAndBounds(t *testing.T) {
	root := t.TempDir()
	e, _ := New(root, t.TempDir())
	content := "package demo\nimport \"fmt\"\ntype Worker struct { Hidden string }\ntype Runner interface { Run() }\nconst Version=1\nvar counter int\nfunc (w *Worker) Run(a int) error { fmt.Println(\"BODY_SECRET\"); return nil }\nfunc helper[T any](v T) T { return v }\n"
	path := filepath.Join(root, "source.go")
	os.WriteFile(path, []byte(content), 0600)
	os.Mkdir(filepath.Join(root, "vendor"), 0700)
	os.WriteFile(filepath.Join(root, "vendor", "hidden.go"), []byte("package hidden\nfunc VendorOnly() {}"), 0600)
	os.WriteFile(filepath.Join(root, "broken.go"), []byte("package ?"), 0600)
	os.Symlink(path, filepath.Join(root, "link.go"))
	first := e.Execute(context.Background(), "repo_map", json.RawMessage(`{}`))
	out := decodeEfficiency(t, first)
	if out["files_scanned"] != float64(2) || out["parse_errors"] != float64(1) || out["parse_cache_hits"] != float64(0) || len(out["symbols"].([]any)) != 6 || strings.Contains(first.Content, "BODY_SECRET") || strings.Contains(first.Content, "VendorOnly") {
		t.Fatal(first.Content)
	}
	second := decodeEfficiency(t, e.Execute(context.Background(), "repo_map", json.RawMessage(`{}`)))
	if second["parse_cache_hits"] != float64(2) {
		t.Fatal(second)
	}
	st, _ := os.Stat(path)
	changed := strings.Replace(content, "helper", "worker", 1)
	os.WriteFile(path, []byte(changed), 0600)
	os.Chtimes(path, st.ModTime(), st.ModTime())
	out = decodeEfficiency(t, e.Execute(context.Background(), "repo_map", json.RawMessage(`{"query":"worker"}`)))
	if strings.Contains(fmt.Sprint(out), "helper") || out["parse_cache_hits"] != float64(1) || out["matched_symbols"] != float64(3) {
		t.Fatal(out)
	}
	out = decodeEfficiency(t, e.Execute(context.Background(), "repo_map", json.RawMessage(`{"path":"source.go","limit":1}`)))
	if len(out["symbols"].([]any)) != 1 || out["truncated"] != true {
		t.Fatal(out)
	}
	for i := 0; i < 40; i++ {
		os.WriteFile(filepath.Join(root, fmt.Sprintf("extra-%02d.go", i)), []byte(fmt.Sprintf("package extra\nfunc Extra%d() {}", i)), 0600)
	}
	out = decodeEfficiency(t, e.Execute(context.Background(), "repo_map", json.RawMessage(`{}`)))
	if len(e.mapCache) > mapCacheEntries {
		t.Fatal("unbounded map cache")
	}
	for _, raw := range []string{`null`, `{"path":"../outside"}`, `{"path":"link.go"}`, `{"limit":501}`, `{"query":null}`, `{"limit":0}`, `{"unknown":true}`} {
		if r := e.Execute(context.Background(), "repo_map", json.RawMessage(raw)); !r.Error {
			t.Fatal("invalid map admitted", raw)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if r := e.Execute(ctx, "repo_map", json.RawMessage(`{}`)); !r.Error {
		t.Fatal("map cancel ignored")
	}
}
func TestRepoMapConcurrentAndResultBudget(t *testing.T) {
	root := t.TempDir()
	e, _ := New(root, t.TempDir())
	var body strings.Builder
	body.WriteString("package dense\n")
	for i := 0; i < 1000; i++ {
		fmt.Fprintf(&body, "func F%d(a string) string { return a }\n", i)
	}
	os.WriteFile(filepath.Join(root, "dense.go"), []byte(body.String()), 0600)
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r := e.Execute(context.Background(), "repo_map", json.RawMessage(`{"limit":500}`))
			out := decodeEfficiency(t, r)
			if len(r.Content) > mapMaxResult || out["truncated"] != true {
				t.Error("map result bound", len(r.Content))
			}
		}()
	}
	wg.Wait()
}
func BenchmarkRepoMapWarm(b *testing.B) {
	root := b.TempDir()
	e, _ := New(root, b.TempDir())
	for i := 0; i < 16; i++ {
		os.WriteFile(filepath.Join(root, fmt.Sprintf("file-%02d.go", i)), []byte("package fixture\n"+strings.Repeat("// comment content\n", 300)+fmt.Sprintf("func Entry%d() int { return 1 }", i)), 0600)
	}
	args := json.RawMessage(`{}`)
	if r := e.Execute(context.Background(), "repo_map", args); r.Error {
		b.Fatal(r)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if r := e.Execute(context.Background(), "repo_map", args); r.Error {
			b.Fatal(r)
		}
	}
}

func BenchmarkRepoMapCold(b *testing.B) {
	root := b.TempDir()
	e, _ := New(root, b.TempDir())
	for i := 0; i < 16; i++ {
		os.WriteFile(filepath.Join(root, fmt.Sprintf("file-%02d.go", i)), []byte("package fixture\n"+strings.Repeat("// comment content\n", 300)+fmt.Sprintf("func Entry%d() int { return 1 }", i)), 0600)
	}
	args := json.RawMessage(`{}`)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		e.mapMu.Lock()
		e.mapCache = nil
		e.mapOrder = nil
		e.mapMu.Unlock()
		if r := e.Execute(context.Background(), "repo_map", args); r.Error {
			b.Fatal(r)
		}
	}
}
