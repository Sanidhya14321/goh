package commands

import (
	"github.com/Sanidhya14321/goh/internal/core"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExpandReferenceContract(t *testing.T) {
	definitions := map[string]core.CommandConfig{"positions": {Template: "first=<$1> second=<$2> all=<$ARGUMENTS>"}, "append": {Template: "Investigate the request."}}
	for _, test := range []struct{ input, want, name string }{
		{`/positions alpha "beta gamma" delta`, `first=<alpha> second=<beta gamma delta> all=<alpha "beta gamma" delta>`, "positions"},
		{`/positions only`, `first=<only> second=<> all=<only>`, "positions"},
		{`/append one two`, "Investigate the request.\n\none two", "append"},
		{"ordinary prompt", "ordinary prompt", ""},
		{"/compact", "/compact", ""},
	} {
		t.Run(test.input, func(t *testing.T) {
			got, name, err := Expand(test.input, definitions)
			if err != nil {
				t.Fatal(err)
			}
			if got != test.want || name != test.name {
				t.Fatalf("got %q/%q expected %q/%q", got, name, test.want, test.name)
			}
		})
	}
}
func TestQuoteParserAndNonrecursiveSubstitution(t *testing.T) {
	definitions := map[string]core.CommandConfig{"test": {Template: "<$1>|<$2>|<$3>"}, "raw": {Template: "$ARGUMENTS"}}
	for _, test := range []struct{ input, want string }{
		{`/test "" 'two words' last`, "<>|<two words>|<last>"},
		{`/test escaped\ space "double\"quote" 'single\literal'`, `<escaped space>|<double"quote>|<single\literal>`},
		{`/test $ARGUMENTS $1 tail`, `<$ARGUMENTS>|<$1>|<tail>`},
		{`/raw $1 $ARGUMENTS`, `$1 $ARGUMENTS`},
	} {
		got, _, err := Expand(test.input, definitions)
		if err != nil {
			t.Fatal(err)
		}
		if got != test.want {
			t.Fatalf("%s got %q expected %q", test.input, got, test.want)
		}
	}
}
func TestNoExecutableOrFileInterpolation(t *testing.T) {
	marker := filepath.Join(t.TempDir(), "marker")
	template := "Literal !`touch " + marker + "` @" + marker + " $(touch " + marker + ") $ARGUMENTS"
	got, _, err := Expand("/test !`touch "+marker+"`", map[string]core.CommandConfig{"test": {Template: template}})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "!`touch "+marker+"`") {
		t.Fatal("executable syntax was interpolated")
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatal("command performed a filesystem effect")
	}
}
func TestRejectMalformedAndOversized(t *testing.T) {
	for _, test := range []struct {
		input       string
		definitions map[string]core.CommandConfig
	}{
		{"/unknown", nil}, {"/test 'unfinished", map[string]core.CommandConfig{"test": {Template: "$1"}}}, {"/test trailing\\", map[string]core.CommandConfig{"test": {Template: "$1"}}}, {"/test", map[string]core.CommandConfig{"test": {Template: "$0"}}}, {"/test", map[string]core.CommandConfig{"test": {Template: "$99999999999999999999999"}}}, {"/test", map[string]core.CommandConfig{"test": {Template: strings.Repeat("x", MaxTemplate+1)}}}, {"/test", map[string]core.CommandConfig{"test": {Template: "ok"}, "compact": {Template: "override"}}}, {"/../test", nil},
	} {
		if _, _, err := Expand(test.input, test.definitions); err == nil {
			t.Fatalf("accepted %q %+v", test.input, test.definitions)
		}
	}
	raw := strings.Repeat("x", MaxPrompt/2)
	if _, _, err := Expand("/test "+raw, map[string]core.CommandConfig{"test": {Template: "$ARGUMENTS$ARGUMENTS$ARGUMENTS"}}); err == nil {
		t.Fatal("expanded output unbounded")
	}
	if _, _, err := Expand(strings.Repeat("x", MaxPrompt+1), nil); err == nil {
		t.Fatal("input unbounded")
	}
}
func FuzzExpansion(f *testing.F) {
	f.Add(`/test alpha "beta gamma"`)
	f.Add(`/test !` + "`echo hi`")
	f.Fuzz(func(t *testing.T, input string) {
		if len(input) > MaxPrompt {
			t.Skip()
		}
		got, _, err := Expand(input, map[string]core.CommandConfig{"test": {Template: "$1 $2 $ARGUMENTS"}})
		if err == nil && len(got) > MaxPrompt {
			t.Fatal("output bound violated")
		}
	})
}
