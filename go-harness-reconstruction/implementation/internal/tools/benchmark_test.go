package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// These benchmarks measure this implementation against deterministic local
// fixtures. They do not establish a performance comparison with OpenCode.
func BenchmarkSearchProject(b *testing.B) {
	root := b.TempDir()
	e, err := New(root, b.TempDir())
	if err != nil {
		b.Fatal(err)
	}
	content := strings.Repeat("ordinary project line with no match\n", 512) + "needle42\n"
	for i := 0; i < 128; i++ {
		if err := os.WriteFile(filepath.Join(root, fmt.Sprintf("fixture-%03d.go", i)), []byte(content), 0644); err != nil {
			b.Fatal(err)
		}
	}
	args := json.RawMessage(`{"pattern":"needle42","limit":200}`)
	b.SetBytes(int64(128 * len(content)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if r := e.Execute(context.Background(), "search", args); r.Error {
			b.Fatal(r.Content)
		}
	}
}
func BenchmarkEditExact(b *testing.B) {
	root := b.TempDir()
	e, err := New(root, b.TempDir())
	if err != nil {
		b.Fatal(err)
	}
	prefix := strings.Repeat("unchanged project contents\r\n", 4096)
	before := prefix + "TARGET_A\r\n"
	after := prefix + "TARGET_B\r\n"
	if err := os.WriteFile(filepath.Join(root, "fixture.txt"), []byte(before), 0644); err != nil {
		b.Fatal(err)
	}
	first, _ := json.Marshal(map[string]any{"path": "fixture.txt", "old_text": "TARGET_A", "new_text": "TARGET_B", "expected_sha256": hash([]byte(before))})
	second, _ := json.Marshal(map[string]any{"path": "fixture.txt", "old_text": "TARGET_B", "new_text": "TARGET_A", "expected_sha256": hash([]byte(after))})
	b.SetBytes(int64(len(before)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		args := first
		if i%2 != 0 {
			args = second
		}
		if r := e.Execute(context.Background(), "edit_file", args); r.Error {
			b.Fatal(r.Content)
		}
	}
}
