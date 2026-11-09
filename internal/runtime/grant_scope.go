package runtime

import (
	"encoding/json"
	"net/url"
	"strings"

	"github.com/Sanidhya14321/goh/internal/core"
)

// compoundMarks make a shell command compound; such a command is only ever
// covered by an approval of exactly itself.
var compoundMarks = []string{";", "&", "|", "`", "$(", ">", "<", "\n", "\r"}

// GrantScope returns what a "for this session" approval of p covers. Empty
// means the whole tool (edits inside the project, a named MCP tool). For
// shell commands and background jobs it is the first two words of the
// command (exactly the command when it is compound); for webfetch it is the
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
		for _, m := range compoundMarks {
			if strings.Contains(cmd, m) {
				return "exact:" + cmd
			}
		}
		words := strings.Fields(cmd)
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
