package core

import (
	"strings"
	"testing"
)

func TestKeymapDefaultsAndOverrides(t *testing.T) {
	km, err := ResolveKeymap(nil)
	if err != nil || km.Direct["ctrl+p"] != "command_palette" || km.Direct["ctrl+k"] != "command_palette" || km.Direct["ctrl+x"] != "leader" || km.Leader["e"] != "editor_open" {
		t.Fatal(km, err)
	}
	km, err = ResolveKeymap(map[string]string{"leader": "ctrl+o", "editor_open": "<leader>v,f4", "sidebar_toggle": "none"})
	if err != nil || km.Direct["ctrl+o"] != "leader" || km.Leader["v"] != "editor_open" || km.Direct["f4"] != "editor_open" {
		t.Fatal(km, err)
	}
	if _, bound := km.Direct["ctrl+b"]; bound {
		t.Fatal("none should unbind the default")
	}
	if !strings.HasPrefix(km.Hint, "Ctrl+o:") || !strings.Contains(km.Hint, "v editor open") {
		t.Fatal(km.Hint)
	}
}

func TestKeymapRejectsBadBindings(t *testing.T) {
	for name, user := range map[string]map[string]string{
		"unknown action":  {"launch_rockets": "ctrl+r"},
		"reserved key":    {"timeline": "ctrl+c"},
		"redraw key":      {"command_palette": "ctrl+l"},
		"malformed":       {"status": "ctrl+"},
		"conflict":        {"status": "ctrl+p"},
		"leader list":     {"leader": "ctrl+x,ctrl+o"},
		"reserved leader": {"leader": "enter"},
	} {
		if _, err := ResolveKeymap(user); err == nil {
			t.Errorf("%s accepted: %v", name, user)
		}
	}
	c := Defaults()
	c.Model, c.Endpoint, c.Keybinds = "m", "http://127.0.0.1:1/v1/chat/completions", map[string]string{"status": "ctrl+p"}
	if err := c.Validate(); err == nil || !strings.Contains(err.Error(), "bound to both") {
		t.Fatal("config validation must reject conflicting keybinds:", err)
	}
}
