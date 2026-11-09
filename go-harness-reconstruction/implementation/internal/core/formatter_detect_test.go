package core_test

import (
	"errors"
	"testing"

	. "example.com/independent-harness/internal/core"

	"example.com/independent-harness/internal/formatters"
)

func TestWithDetectedFormatters(t *testing.T) {
	onPath := map[string]bool{"gofmt": true, "prettier": true, "ruff": true, "shfmt": true}
	look := func(cmd string) (string, error) {
		if onPath[cmd] {
			return "/usr/bin/" + cmd, nil
		}
		return "", errors.New("not found")
	}
	var c Config
	if got := c.WithDetectedFormatters(look); len(got.Formatters) != 0 {
		t.Fatal("added formatters while formatter_auto is off", got.Formatters)
	}
	c.FormatterAuto = true
	c.Formatters = map[string]FormatterConfig{
		"mypy":  {Command: "black", Args: []string{"$FILE"}, Extensions: []string{".py"}},
		"shfmt": {Disabled: true, Extensions: []string{".sh"}},
	}
	got := c.WithDetectedFormatters(look)
	if _, ok := got.Formatters["gofmt"]; !ok {
		t.Fatal("gofmt on PATH not added", got.Formatters)
	}
	if _, ok := got.Formatters["prettier"]; !ok {
		t.Fatal("prettier on PATH not added")
	}
	if _, ok := got.Formatters["ruff"]; ok {
		t.Fatal("ruff added although a configured formatter owns .py")
	}
	if !got.Formatters["shfmt"].Disabled {
		t.Fatal("disabled configured formatter replaced")
	}
	if _, ok := got.Formatters["rustfmt"]; ok {
		t.Fatal("formatter not on PATH added")
	}
	if _, err := formatters.Discover(got.Formatters, t.TempDir()); err != nil {
		t.Fatal(err)
	}
	if len(c.Formatters) != 2 {
		t.Fatal("input config mutated")
	}
}
