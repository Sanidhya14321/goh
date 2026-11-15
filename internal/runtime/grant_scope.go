package runtime

import (
	"encoding/json"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/Sanidhya14321/goh/internal/core"
)

// codeRunners take their program as an argument (sh -c, node -e, python
// script.py), so their first two words say nothing about what will run.
var codeRunners = map[string]bool{
	"sh": true, "bash": true, "zsh": true, "dash": true, "ksh": true, "fish": true, "csh": true, "tcsh": true,
	"python": true, "python2": true, "python3": true, "node": true, "deno": true, "bun": true, "ruby": true,
	"perl": true, "php": true, "lua": true, "osascript": true, "pwsh": true, "powershell": true, "cmd": true,
	"eval": true, "exec": true, "env": true, "xargs": true, "sudo": true, "doas": true, "nohup": true,
	"timeout": true, "nice": true, "watch": true, "time": true, "command": true, "builtin": true, "awk": true,
}

// runsArbitrary reports a command whose session approval must cover only
// itself: quoted text, a code runner, or a flag that executes or deletes
// (find -exec, go test -exec, find -delete).
func runsArbitrary(cmd string, words []string) bool {
	if strings.ContainsAny(cmd, `'"\$`) || len(words) == 0 || codeRunners[filepath.Base(words[0])] {
		return true
	}
	for _, w := range words[1:] {
		switch strings.SplitN(w, "=", 2)[0] {
		case "-exec", "-execdir", "-ok", "-okdir", "-delete", "--exec", "-toolexec", "--upload-pack", "-c":
			return true
		}
	}
	return false
}

// GrantScope returns what a "for this session" approval of p covers. Empty
// means the whole tool (edits inside the project, a named MCP tool). For
// shell commands and background jobs it is the first two words of the
// command, or exactly the command when it is compound or could run arbitrary
// code (see runsArbitrary); for webfetch it is the
// URL's origin.
func GrantScope(p core.Permission) string {
	name := strings.TrimPrefix(p.Tool, "child:")
	switch name {
	case "shell", "job_start":
		var a struct {
			Command string `json:"command"`
		}
		if json.Unmarshal(p.Arguments, &a) != nil {
			return "invalid"
		}
		cmd := strings.TrimSpace(a.Command)
		words := strings.Fields(cmd)
		if core.HasShellOperator(cmd) || runsArbitrary(cmd, words) {
			return "exact:" + cmd
		}
		if len(words) > 2 {
			words = words[:2]
		}
		return "prefix:" + strings.Join(words, " ")
	case "webfetch":
		var a struct {
			URL string `json:"url"`
		}
		if json.Unmarshal(p.Arguments, &a) != nil {
			return "invalid"
		}
		u, err := url.Parse(a.URL)
		if err != nil || u.Host == "" {
			return "invalid"
		}
		return "origin:" + strings.ToLower(u.Scheme+"://"+u.Host)
	}
	return ""
}
