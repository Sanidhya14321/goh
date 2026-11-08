package commands

import (
	"fmt"
	"strings"
	"testing"

	"example.com/independent-harness/internal/core"
)

func TestInitBuiltin(t *testing.T) {
	all := WithBuiltins(nil)
	if Validate(all) != nil {
		t.Fatal("builtins invalid")
	}
	got, name, err := Expand("/init focus on tests", all)
	if err != nil || name != "init" || !strings.Contains(got, "AGENTS.md") || !strings.HasSuffix(got, "focus on tests") {
		t.Fatal(got, name, err)
	}
	mine := WithBuiltins(map[string]core.CommandConfig{"init": {Template: "my init"}})
	if mine["init"].Template != "my init" {
		t.Fatal("user command must replace the builtin")
	}
	full := map[string]core.CommandConfig{}
	for i := 0; i < 256; i++ {
		full[fmt.Sprintf("c%d", i)] = core.CommandConfig{Template: "x"}
	}
	if got := WithBuiltins(full); len(got) != 256 || Validate(got) != nil {
		t.Fatal("builtins pushed past the limit", len(got))
	}
}
