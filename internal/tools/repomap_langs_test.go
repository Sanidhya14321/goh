package tools

import (
	"testing"
)

func TestRegexSymbolsAcrossLanguages(t *testing.T) {
	cases := map[string]struct {
		src  string
		want []string
	}{
		"a.py":   {"class Parser:\n    def parse(self, s):\n        pass\nasync def main():\n    # def not_this(): pass\n    pass\n", []string{"class:Parser", "method:parse", "function:main"}},
		"a.ts":   {"export interface Opts { a: number }\nexport default class App {}\nexport async function run(x: Opts) {}\nconst add = (a: number, b: number): number => a + b\n// function nope() {}\n", []string{"type:Opts", "class:App", "function:run", "function:add"}},
		"a.rs":   {"pub struct Config {}\nimpl Config {\n    pub fn new() -> Self { Config{} }\n}\npub(crate) async fn serve() {}\n", []string{"type:Config", "impl:Config", "function:new", "function:serve"}},
		"A.java": {"public class Account {\n    private int n;\n    public static int total(int a) {\n        return a;\n    }\n}\n", []string{"type:Account", "method:total"}},
		"a.c":    {"struct node { int v; };\nstatic int add(int a, int b) {\n  if (a) { return a; }\n  return a + b;\n}\nint proto(int);\n", []string{"type:node", "function:add"}},
		"a.rb":   {"module Shop\n  class Cart\n    def total?\n    end\n    def self.build\n    end\n  end\nend\n", []string{"type:Shop", "type:Cart", "method:total?", "method:self.build"}},
	}
	for file, tc := range cases {
		got := []string{}
		for _, s := range regexSymbols(file, []byte(tc.src)).Symbols {
			got = append(got, s.Kind+":"+s.Name)
		}
		if len(got) != len(tc.want) {
			t.Fatalf("%s: got %v want %v", file, got, tc.want)
		}
		for i := range got {
			if got[i] != tc.want[i] {
				t.Fatalf("%s: got %v want %v", file, got, tc.want)
			}
		}
	}
}
