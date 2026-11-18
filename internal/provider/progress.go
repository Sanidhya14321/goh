package provider

import (
	"encoding/json"
	"regexp"
	"strings"

	"github.com/Sanidhya14321/goh/internal/core"
)

// toolTracker follows the tool call being streamed and reports its progress.
// It reads only the first few KiB of the arguments, to find a target.
type toolTracker struct {
	sink   func(core.ToolProgress)
	cur    core.ToolProgress
	head   strings.Builder // the start of the arguments, for the target
	active bool
}

const progressHeadBytes = 4 << 10

// targetField finds a path or command in partial JSON arguments.
var targetField = regexp.MustCompile(`"(path|file_path|filePath|command|url|pattern)"\s*:\s*"((?:[^"\\]|\\.){1,300})"`)

// start begins a new tool call named name.
func (t *toolTracker) start(name string) {
	if t == nil || t.sink == nil {
		return
	}
	t.cur, t.active = core.ToolProgress{Tool: name}, true
	t.head.Reset()
	t.sink(t.cur)
}

// add records more argument text for the current call.
func (t *toolTracker) add(chunk string) {
	if t == nil || t.sink == nil || !t.active || chunk == "" {
		return
	}
	t.cur.Chars += len(chunk)
	if t.cur.Target == "" && t.head.Len() < progressHeadBytes {
		t.head.WriteString(chunk[:min(len(chunk), progressHeadBytes-t.head.Len())])
		if m := targetField.FindStringSubmatch(t.head.String()); m != nil {
			var v string
			if json.Unmarshal([]byte(`"`+m[2]+`"`), &v) == nil {
				t.cur.Target = firstLineOf(v, 120)
			}
		}
	}
	t.sink(t.cur)
}

func firstLineOf(s string, limit int) string {
	s, _, _ = strings.Cut(strings.TrimSpace(s), "\n")
	if r := []rune(s); len(r) > limit {
		s = string(r[:limit]) + "…"
	}
	return s
}
