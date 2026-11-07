package core

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// DefaultKeybinds lists chat-level actions and their default keys. Values are
// comma-separated alternatives; "<leader>x" means the leader key, then x.
var DefaultKeybinds = map[string]string{
	"leader":          "ctrl+x",
	"command_palette": "ctrl+p,ctrl+k",
	"sidebar_toggle":  "ctrl+b",
	"timeline":        "ctrl+t",
	"status":          "ctrl+g",
	"editor_open":     "<leader>e",
}

// Keys the composer, approvals and cancellation depend on cannot be rebound.
var reservedKeys = map[string]bool{
	"enter": true, "esc": true, "tab": true, "shift+tab": true, "backspace": true,
	"ctrl+c": true, "ctrl+d": true, "up": true, "down": true, "left": true, "right": true,
	"alt+enter": true, "ctrl+a": true, "ctrl+e": true, "ctrl+u": true, "ctrl+w": true, "ctrl+r": true,
}

var keyPattern = regexp.MustCompile(`^((ctrl|alt|shift)\+)*([a-z0-9]|f([1-9]|1[0-2]))$`)

// Keymap resolves a key press to an action. Leader holds the second key of a
// "<leader>x" binding.
type Keymap struct {
	Direct map[string]string // key -> action
	Leader map[string]string // key after leader -> action
	Hint   string            // short help shown after pressing the leader
}

// ResolveKeymap overlays user bindings on the defaults and rejects unknown
// actions, malformed or reserved keys, and keys bound to two actions.
func ResolveKeymap(user map[string]string) (Keymap, error) {
	merged := map[string]string{}
	for a, k := range DefaultKeybinds {
		merged[a] = k
	}
	for a, k := range user {
		if _, ok := DefaultKeybinds[a]; !ok {
			return Keymap{}, fmt.Errorf("keybinds: unknown action %q", a)
		}
		merged[a] = k
	}
	leader := strings.ToLower(strings.TrimSpace(merged["leader"]))
	if strings.Contains(leader, ",") || strings.HasPrefix(leader, "<leader>") || !keyPattern.MatchString(leader) || reservedKeys[leader] {
		return Keymap{}, fmt.Errorf("keybinds: leader must be one unreserved key, got %q", merged["leader"])
	}
	km := Keymap{Direct: map[string]string{leader: "leader"}, Leader: map[string]string{}}
	actions := make([]string, 0, len(merged))
	for a := range merged {
		actions = append(actions, a)
	}
	sort.Strings(actions)
	var hints []string
	for _, action := range actions {
		if action == "leader" || strings.TrimSpace(merged[action]) == "none" {
			continue
		}
		for _, raw := range strings.Split(merged[action], ",") {
			key := strings.ToLower(strings.TrimSpace(raw))
			table, isLeader := km.Direct, false
			if rest, ok := strings.CutPrefix(key, "<leader>"); ok {
				table, key, isLeader = km.Leader, rest, true
			}
			if !keyPattern.MatchString(key) || (!isLeader && reservedKeys[key]) {
				return Keymap{}, fmt.Errorf("keybinds: %s has invalid or reserved key %q", action, raw)
			}
			if other, taken := table[key]; taken {
				return Keymap{}, fmt.Errorf("keybinds: %q is bound to both %s and %s", raw, other, action)
			}
			table[key] = action
			if isLeader {
				hints = append(hints, key+" "+strings.ReplaceAll(action, "_", " "))
			}
		}
	}
	km.Hint = strings.ToUpper(leader[:1]) + leader[1:] + ": " + strings.Join(hints, " · ")
	return km, nil
}
