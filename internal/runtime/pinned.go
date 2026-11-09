package runtime

import (
	"encoding/json"
	"strings"

	"github.com/Sanidhya14321/goh/internal/core"
)

// pinnedMarker starts the message that carries earlier user instructions
// verbatim across compaction.
const pinnedMarker = "Standing user instructions from earlier in this session (verbatim, oldest first; later instructions take precedence over earlier ones):\n"

const (
	maxPinned      = 30
	maxPinnedEach  = 2 << 10
	maxPinnedTotal = 16 << 10
)

// pinnedInstructions collects the user's earlier messages (including any
// previously pinned ones) into one user message, newest kept when bounded.
func pinnedInstructions(history []core.Message) (core.Message, bool) {
	var all []string
	for _, m := range history {
		if m.Role != "user" {
			continue
		}
		if rest, ok := strings.CutPrefix(m.Content, pinnedMarker); ok {
			var prior []string
			if json.Unmarshal([]byte(rest), &prior) == nil {
				all = append(all, prior...)
			}
			continue
		}
		if text := strings.TrimSpace(m.Content); text != "" {
			if len(text) > maxPinnedEach {
				text = text[:maxPinnedEach] + " […]"
			}
			all = append(all, text)
		}
	}
	if len(all) > maxPinned {
		all = all[len(all)-maxPinned:]
	}
	for total(all) > maxPinnedTotal && len(all) > 1 {
		all = all[1:]
	}
	if len(all) == 0 {
		return core.Message{}, false
	}
	b, _ := json.Marshal(all)
	return core.Message{Role: "user", Content: pinnedMarker + string(b)}, true
}

func total(items []string) int {
	n := 0
	for _, s := range items {
		n += len(s)
	}
	return n
}
