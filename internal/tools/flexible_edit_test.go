package tools

import (
	"strings"
	"testing"
)

func TestFlexibleReplace(t *testing.T) {
	file := "func a() {\n\tif x {\n\t\treturn 1\n\t}\n}\n\nfunc b() {\n\tif y {\n\t\treturn 2\n\t}\n}\n"
	// The model dropped the tabs: still a unique match, re-indented.
	got, ok, why := flexibleReplace(file, "if x {\n\treturn 1\n}", "if x {\n\treturn 10\n}")
	if !ok || !strings.Contains(got, "\tif x {\n\t\treturn 10\n\t}\n") || !strings.Contains(got, "return 2") {
		t.Fatalf("%v %s\n%s", ok, why, got)
	}
	// Ambiguous when whitespace is ignored: refused, with line numbers.
	amb := "x := 1\n  x := 1\n"
	if _, ok, why := flexibleReplace(amb, "x := 1", "y"); ok || !strings.Contains(why, "lines 1, 2") {
		t.Fatal(ok, why)
	}
	// Not found: points at where the first line appears.
	if _, ok, why := flexibleReplace(file, "if x {\n\treturn 99\n}", "z"); ok || !strings.Contains(why, "line(s) 2") {
		t.Fatal(ok, why)
	}
	// No trailing newline preserved.
	if got, ok, _ := flexibleReplace("a\n  b", "b", "c"); !ok || got != "a\n  c" {
		t.Fatalf("%q", got)
	}
}
