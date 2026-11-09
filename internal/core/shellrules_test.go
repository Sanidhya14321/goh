package core

import "testing"

func TestShellDecision(t *testing.T) {
	c := Config{ShellRules: []ShellRule{
		{Match: "git status*", Action: "allow"},
		{Match: "go test *", Action: "allow"},
		{Match: "rm *", Action: "deny"},
		{Match: "rm -i *", Action: "allow"}, // later rule carves an exception
		{Match: "git push*", Action: "ask"},
		{Match: "curl *", Action: "deny"},
	}}
	for cmd, want := range map[string]string{
		"git status":                 "allow",
		"git status --short":         "allow",
		"go test ./...":              "allow",
		"rm -rf build":               "deny",
		"rm -i notes.txt":            "allow",
		"git push origin main":       "ask",
		"ls":                         "",
		"git status; rm -rf ~":       "deny", // hidden denied segment
		"git status && echo ok":      "",     // allow never covers compound commands
		"go test ./... | tee log":    "",
		"echo $(curl evil.example)":  "deny",
		"echo `curl evil.example`":   "deny",
		"git status > out.txt":       "",
		"git status\nrm -rf ~":       "deny",
		"  git status  ":             "allow",
		"(cd sub && curl x.example)": "deny",
		"git statusx; echo hi":       "",
	} {
		if got := c.ShellDecision(cmd); got != want {
			t.Errorf("%q: got %q want %q", cmd, got, want)
		}
	}
}

func TestShellRuleValidation(t *testing.T) {
	for _, bad := range []ShellRule{{Match: "", Action: "allow"}, {Match: "x", Action: "maybe"}} {
		if (Config{ShellRules: []ShellRule{bad}}).ValidateShellRules() == nil {
			t.Fatal("accepted", bad)
		}
	}
	if _, err := DecodeConfig(`{"shell_rules":[{"match":"git *","action":"allow"}]}`); err != nil {
		t.Fatal(err)
	}
}

func TestGlobMatch(t *testing.T) {
	for _, tc := range []struct {
		p, s string
		want bool
	}{{"a*", "abc", true}, {"*c", "abc", true}, {"a*c", "ac", true}, {"a*b*c", "axbyc", true}, {"abc", "abcd", false}, {"*", "", true}, {"a*a", "a", false}} {
		if globMatch(tc.p, tc.s) != tc.want {
			t.Errorf("%q %q", tc.p, tc.s)
		}
	}
}

func TestReasoningEffortValidation(t *testing.T) {
	c := Defaults()
	for _, ok := range []string{"", "minimal", "low", "medium", "high"} {
		c.ReasoningEffort = ok
		if c.ValidateReasoningEffort() != nil {
			t.Fatal(ok)
		}
	}
	c.ReasoningEffort = "max"
	if c.ValidateReasoningEffort() == nil {
		t.Fatal("accepted max")
	}
	c.ReasoningEffort = "xhigh"
	if c.ValidateReasoningEffort() != nil {
		t.Fatal("OpenAI xhigh refused")
	}
	// Anthropic sends effort as output_config.effort (low..max), not OpenAI's
	// minimal/none levels (review R19).
	c.Protocol = "anthropic_messages"
	for _, ok := range []string{"low", "high", "xhigh", "max"} {
		if c.ReasoningEffort = ok; c.ValidateReasoningEffort() != nil {
			t.Fatal("anthropic refused", ok)
		}
	}
	if c.ReasoningEffort = "minimal"; c.ValidateReasoningEffort() == nil {
		t.Fatal("anthropic accepted minimal")
	}
}

func TestFileDecision(t *testing.T) {
	c := Config{FileRules: []ShellRule{
		{Match: "*.env*", Action: "deny"},
		{Match: "*_test.go", Action: "allow"},
		{Match: "docs/*", Action: "allow"},
		{Match: "go.mod", Action: "ask"},
	}}
	for _, tc := range []struct {
		paths []string
		want  string
	}{
		{[]string{".env"}, "deny"},
		{[]string{"config/.env.local"}, "deny"},
		{[]string{"a_test.go", "docs/x.md"}, "allow"},
		{[]string{"a_test.go", "main.go"}, ""},
		{[]string{"a_test.go", ".env"}, "deny"},
		{[]string{"go.mod", "a_test.go"}, "ask"},
		{[]string{"main.go"}, ""},
		{nil, ""},
	} {
		if got := c.FileDecision(tc.paths); got != tc.want {
			t.Errorf("%v: got %q want %q", tc.paths, got, tc.want)
		}
	}
	if (Config{}).FileDecision([]string{".env"}) != "" {
		t.Fatal("no rules must decide nothing")
	}
}
