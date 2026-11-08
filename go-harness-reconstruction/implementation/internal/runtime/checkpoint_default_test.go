package runtime

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"example.com/independent-harness/internal/core"
)

func TestDefaultCheckpointsNeverBlockEdits(t *testing.T) {
	for _, strict := range []bool{false, true} {
		a, root := testAgent(t)
		a.Config.Snapshots, a.Config.SnapshotsStrict = true, strict
		big := []byte("before\n")
		path := filepath.Join(root, "big.txt")
		os.WriteFile(path, big, 0o600)
		// Make checkpoint storage unusable: a file where its folder belongs.
		os.WriteFile(a.snapshotDirectory(), []byte("not a folder"), 0o600)
		sum := sha256.Sum256(big)
		args, _ := json.Marshal(map[string]string{"path": "big.txt", "content": "small now\n", "expected_sha256": hex.EncodeToString(sum[:])})
		c := core.Call{ID: "w1", Type: "function"}
		c.Function.Name, c.Function.Arguments = "write_file", string(args)
		r := a.executeMutation(context.Background(), c, []string{"big.txt"})
		got, _ := os.ReadFile(path)
		if strict {
			if !r.Error || string(got) != string(big) {
				t.Fatal("strict checkpoints must refuse an edit they cannot capture")
			}
			continue
		}
		if r.Error || string(got) != "small now\n" || !strings.Contains(r.Content, "checkpoint_skipped") {
			t.Fatalf("best-effort checkpoints blocked or hid the edit: %s", r.Content)
		}
	}
}
