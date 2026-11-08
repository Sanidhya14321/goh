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
