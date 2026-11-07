package tools

import (
	"context"
	"encoding/json"
	"errors"
	"example.com/independent-harness/internal/testutil"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func patchText(lines ...string) string {
	return "*** Begin Patch\n" + strings.Join(lines, "\n") + "\n*** End Patch"
}
func patchInput(t *testing.T, text string, hashes map[string]string) json.RawMessage {
	return mustJSON(t, map[string]any{"patchText": text, "expected_sha256": hashes})
}
func TestApplyPatchMultiFileCreateUpdateDeleteMove(t *testing.T) {
	e, root := engine(t)
	files := map[string]string{"update": "alpha\r\nbeta\r\ngamma\r\n", "move": "old move\n", "delete": "obsolete\n"}
	hashes := map[string]string{}
	for p, body := range files {
		if err := os.WriteFile(filepath.Join(root, p), []byte(body), 0751); err != nil {
			t.Fatal(err)
		}
		hashes[p] = hash([]byte(body))
	}
	text := patchText("*** Add File: added", "+created", "*** Update File: update", "@@", " alpha", "-beta", "+BETA", " gamma", "*** Update File: move", "*** Move to: moved", "@@", "-old move", "+new move", "*** Delete File: delete")
	raw := patchInput(t, text, hashes)
	preview, side, err := e.Prepare("apply_patch", raw)
	if err != nil || !side {
		t.Fatal(preview, side, err)
	}
	for _, p := range []string{"added", "update", "move", "moved", "delete"} {
		if !strings.Contains(preview, filepath.Join(root, p)) {
			t.Fatal("missing approval path", p)
		}
	}
	for p, body := range files {
		b, _ := os.ReadFile(filepath.Join(root, p))
		if string(b) != body {
			t.Fatal("Prepare changed", p)
		}
	}
	v := object(t, e.Execute(context.Background(), "apply_patch", raw))
	if v["status"] != "completed" {
		t.Fatal(v)
	}
	for p, want := range map[string]string{"added": "created\n", "update": "alpha\r\nBETA\r\ngamma\r\n", "moved": "new move\n"} {
		b, err := os.ReadFile(filepath.Join(root, p))
		if err != nil || string(b) != want {
			t.Fatal(p, string(b), want, err)
		}
	}
	for _, p := range []string{"move", "delete"} {
		if _, err := os.Stat(filepath.Join(root, p)); !os.IsNotExist(err) {
			t.Fatal("source remains", p)
		}
	}
	for _, p := range []string{"update", "moved"} {
		st, _ := os.Stat(filepath.Join(root, p))
		if testutil.UnixModes() && st.Mode().Perm() != 0751 {
			t.Fatal(p, st.Mode())
		}
	}
	entries, _ := os.ReadDir(root)
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".harness-patch-") {
			t.Fatal("staging file leaked")
		}
	}
}
func TestApplyPatchExactUniqueContextAndWholePatchPrevalidation(t *testing.T) {
	for _, tc := range []struct{ name, source, text string }{
		{"ambiguous", "repeat\nrepeat\n", patchText("*** Add File: premature", "+created", "*** Update File: source", "@@", "-repeat", "+changed")},
		{"whitespace", "  exact  \n", patchText("*** Add File: premature", "+created", "*** Update File: source", "@@", "-exact", "+changed")},
		{"missing", "original\n", patchText("*** Add File: premature", "+created", "*** Update File: missing", "@@", "-old", "+new")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e, root := engine(t)
			os.WriteFile(filepath.Join(root, "source"), []byte(tc.source), 0644)
			raw := patchInput(t, tc.text, map[string]string{"source": hash([]byte(tc.source)), "missing": strings.Repeat("0", 64)})
			if !e.Execute(context.Background(), "apply_patch", raw).Error {
				t.Fatal("unsafe patch accepted")
			}
			if _, err := os.Stat(filepath.Join(root, "premature")); !os.IsNotExist(err) {
				t.Fatal("earlier operation published before validation")
			}
			b, _ := os.ReadFile(filepath.Join(root, "source"))
			if string(b) != tc.source {
				t.Fatal("source mutated")
			}
		})
	}
}
func TestApplyPatchStaleAfterApproval(t *testing.T) {
	e, root := engine(t)
	os.WriteFile(filepath.Join(root, "source"), []byte("old\n"), 0644)
	raw := patchInput(t, patchText("*** Update File: source", "@@", "-old", "+new"), map[string]string{"source": hash([]byte("old\n"))})
	if _, _, err := e.Prepare("apply_patch", raw); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(root, "source"), []byte("old\nexternal\n"), 0644)
	if !e.Execute(context.Background(), "apply_patch", raw).Error {
		t.Fatal("stale patch accepted")
	}
	b, _ := os.ReadFile(filepath.Join(root, "source"))
	if string(b) != "old\nexternal\n" {
		t.Fatal("stale file changed")
	}
}
func TestApplyPatchCollisionsHashesAndPathSafety(t *testing.T) {
	e, root := engine(t)
	os.WriteFile(filepath.Join(root, "source"), []byte("old\n"), 0644)
	os.WriteFile(filepath.Join(root, "existing"), []byte("keep\n"), 0644)
	os.Symlink(t.TempDir(), filepath.Join(root, "outside"))
	validHash := map[string]string{"source": hash([]byte("old\n"))}
	for _, text := range []string{
		patchText("*** Add File: existing", "+overwrite"),
		patchText("*** Update File: source", "*** Move to: existing", "@@", "-old", "+new"),
		patchText("*** Add File: ../escape", "+bad"),
		patchText("*** Add File: outside/escape", "+bad"),
		patchText("*** Add File: nested/new", "+bad"),
		patchText("*** Add File: duplicate", "+one", "*** Add File: duplicate", "+two"),
		patchText("*** Delete File: source", "*** Add File: source", "+new"),
	} {
		if !e.Execute(context.Background(), "apply_patch", patchInput(t, text, validHash)).Error {
			t.Fatal("accepted unsafe patch", text)
		}
	}
	for _, hashes := range []map[string]string{{}, {"source": "bad"}, {"source": hash([]byte("old\n")), "unused": "hash"}} {
		if !e.Execute(context.Background(), "apply_patch", patchInput(t, patchText("*** Delete File: source"), hashes)).Error {
			t.Fatal("accepted missing/stale/unused hash", hashes)
		}
	}
	b, _ := os.ReadFile(filepath.Join(root, "existing"))
	if string(b) != "keep\n" {
		t.Fatal("collision overwritten")
	}
}
func TestPatchHunksEOFAnchorsAndNoFinalNewline(t *testing.T) {
	for _, tc := range []struct {
		before string
		hunks  []patchHunk
		want   string
	}{
		{"a\nb\nc\nd\n", []patchHunk{{lines: []patchLine{{'-', "b"}, {'+', "B"}}}, {lines: []patchLine{{'-', "d"}, {'+', "D"}}}}, "a\nB\nc\nD\n"},
		{"anchor\nrepeat\nother\nrepeat\n", []patchHunk{{anchor: "other", lines: []patchLine{{'-', "repeat"}, {'+', "last"}}}}, "anchor\nrepeat\nother\nlast\n"},
		{"repeat\nrepeat\n", []patchHunk{{eof: true, lines: []patchLine{{'-', "repeat"}, {'+', "last"}}}}, "repeat\nlast\n"},
		{"no newline", []patchHunk{{lines: []patchLine{{'-', "no newline"}, {'+', "changed"}}}}, "changed"},
		{"alpha\r\nomega\r\n", []patchHunk{{lines: []patchLine{{' ', "alpha"}, {'+', "beta"}, {' ', "omega"}}}}, "alpha\r\nbeta\r\nomega\r\n"},
		{"", []patchHunk{{lines: []patchLine{{'+', "created"}}}}, "created\n"},
	} {
		got, err := applyPatchHunks([]byte(tc.before), tc.hunks)
		if err != nil || string(got) != tc.want {
			t.Fatal(string(got), tc.want, err)
		}
	}
}
func TestMalformedPatchRejected(t *testing.T) {
	for _, text := range []string{"", "*** Begin Patch\n*** End Patch", "*** Begin Patch\n*** Add File: a\ninvalid\n*** End Patch", patchText("*** Update File: a", "@@"), patchText("*** Frobnicate File: a"), patchText("*** Delete File: a", "-old"), patchText("*** Update File: a", "*** Move to: a"), patchText("*** Add File: a", "+new") + "\ntrailing"} {
		if _, err := parsePatch(text); err == nil {
			t.Fatal("accepted malformed patch", text)
		}
	}
	for _, raw := range []string{`null`, `{}`, `{"patchText":null}`, `{"patchText":"x","unexpected":true}`, `{"patchText":"x","expected_sha256":null}`} {
		if _, err := decodePatch(json.RawMessage(raw)); err == nil {
			t.Fatal("accepted invalid patch arguments", raw)
		}
	}
}

type patchObservedContext struct {
	context.Context
	once    sync.Once
	trigger func() bool
	action  func()
	cancel  context.CancelFunc
}

func (c *patchObservedContext) Err() error {
	if c.trigger() {
		c.once.Do(func() {
			if c.action != nil {
				c.action()
			}
			if c.cancel != nil {
				c.cancel()
			}
		})
	}
	return c.Context.Err()
}
func TestPatchCancellationAndLateFailureReportPartialPublication(t *testing.T) {
	for _, mode := range []string{"cancel", "collision"} {
		t.Run(mode, func(t *testing.T) {
			e, root := engine(t)
			base, cancel := context.WithCancel(context.Background())
			defer cancel()
			ctx := &patchObservedContext{Context: base, trigger: func() bool { _, err := os.Stat(filepath.Join(root, "first")); return err == nil }}
			if mode == "cancel" {
				ctx.cancel = cancel
			} else {
				ctx.action = func() { os.WriteFile(filepath.Join(root, "second"), []byte("external\n"), 0644) }
			}
			raw := patchInput(t, patchText("*** Add File: first", "+one", "*** Add File: second", "+two"), map[string]string{})
			r := e.Execute(ctx, "apply_patch", raw)
			if !r.Error {
				t.Fatal("partial patch reported success")
			}
			var v struct {
				Status    string   `json:"status"`
				Published []string `json:"published"`
			}
			if err := json.Unmarshal([]byte(r.Content), &v); err != nil || v.Status != "interrupted" || len(v.Published) != 1 || v.Published[0] != "first" {
				t.Fatal(r, err)
			}
			b, _ := os.ReadFile(filepath.Join(root, "first"))
			if string(b) != "one\n" {
				t.Fatal("first publication missing")
			}
			if mode == "cancel" {
				if _, err := os.Stat(filepath.Join(root, "second")); !os.IsNotExist(err) {
					t.Fatal("second operation executed after cancellation")
				}
			} else {
				b, _ := os.ReadFile(filepath.Join(root, "second"))
				if string(b) != "external\n" {
					t.Fatal("late external file overwritten")
				}
			}
		})
	}
}
func TestPatchCancellationBeforeFirstPublication(t *testing.T) {
	e, root := engine(t)
	base, cancel := context.WithCancel(context.Background())
	defer cancel()
	ctx := &cancelAfterFirstCheck{Context: base, cancel: cancel}
	r := e.Execute(ctx, "apply_patch", patchInput(t, patchText("*** Add File: cancelled", "+effect"), map[string]string{}))
	if !r.Error {
		t.Fatal("cancelled patch succeeded")
	}
	if _, err := os.Stat(filepath.Join(root, "cancelled")); !os.IsNotExist(err) {
		t.Fatal("new effect after cancellation")
	}
	if !errors.Is(ctx.Err(), context.Canceled) {
		t.Fatal("context not cancelled")
	}
}
func FuzzPatchGrammar(f *testing.F) {
	for _, seed := range []string{patchText("*** Add File: a", "+text"), patchText("*** Update File: a", "@@", "-old", "+new"), "", "*** Begin Patch\n*** End Patch"} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, text string) {
		if len(text) > 1<<20 {
			return
		}
		_, _ = parsePatch(text)
	})
}

func TestPatchRechecksEverySourceAfterStagingBeforeAnyPublication(t *testing.T) {
	e, root := engine(t)
	os.WriteFile(filepath.Join(root, "source"), []byte("old\n"), 0644)
	ctx := &patchObservedContext{Context: context.Background(), trigger: func() bool {
		entries, _ := os.ReadDir(root)
		for _, entry := range entries {
			if strings.HasPrefix(entry.Name(), ".harness-patch-") {
				return true
			}
		}
		return false
	}, action: func() { os.WriteFile(filepath.Join(root, "source"), []byte("external\n"), 0644) }}
	raw := patchInput(t, patchText("*** Add File: first", "+new", "*** Update File: source", "@@", "-old", "+changed"), map[string]string{"source": hash([]byte("old\n"))})
	r := e.Execute(ctx, "apply_patch", raw)
	if !r.Error || !strings.Contains(r.Content, "stale") {
		t.Fatal(r)
	}
	if _, err := os.Stat(filepath.Join(root, "first")); !os.IsNotExist(err) {
		t.Fatal("first publication happened despite changed source")
	}
	b, _ := os.ReadFile(filepath.Join(root, "source"))
	if string(b) != "external\n" {
		t.Fatal("external change overwritten")
	}
}
func TestPatchPreviewAndResultRemainBounded(t *testing.T) {
	e, root := engine(t)
	var lines []string
	hashes := map[string]string{}
	for i := 0; i < 20; i++ {
		name := fmt.Sprintf("file-%02d", i)
		body := strings.Repeat("old\n", 2000)
		os.WriteFile(filepath.Join(root, name), []byte(body), 0644)
		hashes[name] = hash([]byte(body))
		lines = append(lines, "*** Update File: "+name, "@@", "-old", "+new", "*** End of File")
	}
	preview, side, err := e.Prepare("apply_patch", patchInput(t, patchText(lines...), hashes))
	if err != nil || !side || len(preview) > MaxInline {
		t.Fatal(len(preview), side, err)
	}
	for i := 0; i < 20; i++ {
		if !strings.Contains(preview, filepath.Join(root, fmt.Sprintf("file-%02d", i))) {
			t.Fatal("approval omits a target")
		}
	}
}

func TestPatchPreviewBudgetsBothSidesForEveryFile(t *testing.T) {
	changes := []patchChange{}
	for i := 0; i < 50; i++ {
		changes = append(changes, patchChange{kind: "update", path: fmt.Sprintf("file-%02d", i), before: []byte(strings.Repeat(fmt.Sprintf("BEFORE_%02d\n", i), 10000)), after: []byte(strings.Repeat(fmt.Sprintf("AFTER_%02d\n", i), 10000))})
	}
	preview := patchPreview("/fixture", changes)
	if preview == "" || len(preview) > MaxInline {
		t.Fatal("preview outside bounds", len(preview))
	}
	for i := 0; i < 50; i++ {
		if !strings.Contains(preview, fmt.Sprintf("-BEFORE_%02d", i)) || !strings.Contains(preview, fmt.Sprintf("+AFTER_%02d", i)) {
			t.Fatal("missing before/after excerpt", i)
		}
	}
}
func TestPatchResultBudgetRejectsBeforeMutation(t *testing.T) {
	if !testutil.UnixModes() {
		t.Skip("Windows forbids control characters in filenames; such patches are refused earlier")
	}
	e, root := engine(t)
	var lines []string
	// Control characters are valid filenames but escape to six bytes in JSON. The permission
	// path list fits; its planned success result exceeds the inline JSON budget.
	for i := 0; i < 100; i++ {
		name := fmt.Sprintf("%03d-", i) + strings.Repeat("\x01", 120)
		lines = append(lines, "*** Add File: "+name, "+new")
	}
	raw := patchInput(t, patchText(lines...), map[string]string{})
	if _, _, err := e.Prepare("apply_patch", raw); err == nil || !strings.Contains(err.Error(), "planned patch result") {
		t.Fatal("oversized planned result was not rejected", err)
	}
	r := e.Execute(context.Background(), "apply_patch", raw)
	if !r.Error || !strings.Contains(r.Content, "planned patch result") {
		t.Fatal(r)
	}
	entries, _ := os.ReadDir(root)
	if len(entries) != 0 {
		t.Fatal("mutation before result budget rejection", len(entries))
	}
}
func TestPatchPartialFailureAlwaysPreservesBoundedUncertainty(t *testing.T) {
	paths := make([]string, 100)
	for i := range paths {
		paths[i] = strings.Repeat("\t", 1000)
	}
	r := patchExecutionFailure(errors.New(strings.Repeat("\x00", 200000)), paths)
	if !r.Error || len(r.Content) > MaxInline {
		t.Fatal("unbounded error", len(r.Content))
	}
	var report struct {
		Status         string `json:"status"`
		PublishedCount int    `json:"published_count"`
		Truncated      bool   `json:"published_truncated"`
		Effects        string `json:"effects"`
	}
	if err := json.Unmarshal([]byte(r.Content), &report); err != nil || report.Status != "interrupted" || report.PublishedCount != 100 || !report.Truncated || report.Effects == "" {
		t.Fatal(report, err)
	}
}

func TestPatchHugeInvalidHeaderErrorIsBounded(t *testing.T) {
	e, _ := engine(t)
	raw := patchInput(t, patchText(strings.Repeat("invalid", 50000)), map[string]string{})
	if _, _, err := e.Prepare("apply_patch", raw); err == nil || len(err.Error()) > 2200 {
		t.Fatal("unbounded approval error")
	}
	r := e.Execute(context.Background(), "apply_patch", raw)
	if !r.Error || len(r.Content) > 2200 {
		t.Fatal("unbounded execution error", len(r.Content))
	}
}
