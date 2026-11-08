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

func TestReviewBuiltin(t *testing.T) {
	all := WithBuiltins(nil)
	got, name, err := Expand("/review feature-x", all)
	if err != nil || name != "review" || !strings.Contains(got, "What to review: feature-x") || !strings.Contains(got, "Do not edit") {
		t.Fatal(got, name, err)
	}
	if got, _, err := Expand("/review", all); err != nil || !strings.Contains(got, "uncommitted changes") {
		t.Fatal(got, err)
	}
}

func TestUnknownCommandSuggestsNearest(t *testing.T) {
	_, _, err := Expand("/reviw this", WithBuiltins(nil))
	if err == nil || !strings.Contains(err.Error(), "did you mean /review?") {
		t.Fatal(err)
	}
	_, _, err = Expand("/zzzzzz", WithBuiltins(nil))
	if err == nil || strings.Contains(err.Error(), "did you mean") || !strings.Contains(err.Error(), "/commands") {
		t.Fatal(err)
	}
}
