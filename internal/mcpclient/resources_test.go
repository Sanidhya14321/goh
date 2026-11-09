package mcpclient

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

// C9: servers that offer resources expose list/read tools; others do not.
func TestResourcesListAndRead(t *testing.T) {
	plain := manager(t, nil, true)
	if err := plain.Connect(context.Background()); err != nil {
		t.Fatal(err)
	}
	if plain.Has(ResourceListTool) {
		t.Fatal("resource tools offered by a server without resources")
	}
	m := manager(t, map[string]string{"HARNESS_MCP_RESOURCES": "1"}, true)
	if err := m.Connect(context.Background()); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, d := range m.Definitions() {
		found = found || d.Function.Name == ResourceReadTool
	}
	if !found || !m.Has(ResourceListTool) {
		t.Fatal("resource tools missing")
	}
	list := m.Execute(context.Background(), ResourceListTool, json.RawMessage(`{}`))
	if list.Error || !strings.Contains(list.Content, "fixture://docs/api") {
		t.Fatalf("%+v", list)
	}
	read := m.Execute(context.Background(), ResourceReadTool, json.RawMessage(`{"server":"fixture","uri":"fixture://docs/api"}`))
	if read.Error || !strings.Contains(read.Content, "RESOURCE-TEXT-55") || !strings.Contains(read.Content, "Untrusted") {
		t.Fatalf("%+v", read)
	}
	if bad := m.Execute(context.Background(), ResourceReadTool, json.RawMessage(`{"server":"nope","uri":"x"}`)); !bad.Error {
		t.Fatal("unknown server accepted")
	}
}
