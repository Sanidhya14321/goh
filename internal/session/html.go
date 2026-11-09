package session

import (
	"bufio"
	"encoding/json"
	"fmt"
	"html"
	"io"

	"github.com/Sanidhya14321/goh/internal/core"
)

const htmlHead = `<!doctype html>
<html lang="en"><head><meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<meta http-equiv="Content-Security-Policy" content="default-src 'none'; style-src 'unsafe-inline'">
<title>GoHarness session</title>
<style>
:root{--bg:#fff;--fg:#1d1d1f;--muted:#6e6e73;--user:#eef3ff;--tool:#f5f5f7;--line:#d2d2d7}
@media (prefers-color-scheme:dark){:root{--bg:#111;--fg:#eee;--muted:#999;--user:#1b2433;--tool:#1c1c1e;--line:#333}}
body{background:var(--bg);color:var(--fg);font:15px/1.55 -apple-system,system-ui,sans-serif;max-width:860px;margin:0 auto;padding:24px 16px}
h1{font-size:20px}.meta{color:var(--muted);font-size:13px}
section{border-top:1px solid var(--line);padding:12px 0}
section.user{background:var(--user);padding:12px;border-radius:8px;border:0;margin:12px 0}
h2{font-size:13px;text-transform:uppercase;letter-spacing:.04em;color:var(--muted);margin:0 0 6px}
.text{white-space:pre-wrap;overflow-wrap:anywhere}
.tool{background:var(--tool);font:13px/1.45 ui-monospace,Menlo,monospace;padding:8px;border-radius:6px;white-space:pre-wrap;overflow-wrap:anywhere}
</style></head><body>
`

// WriteHTMLTranscript writes a self-contained, script-free HTML page of the
// conversation. All text is HTML-escaped and recognisable credentials are
// masked, as in the Markdown transcript. Nothing is uploaded.
func (s *Store) WriteHTMLTranscript(out io.Writer, includeTools bool) error {
	s.mu.Lock()
	records := append([]Record(nil), s.records...)
	s.mu.Unlock()
	w := bufio.NewWriter(out)
	size := 0
	put := func(parts ...string) error {
		for _, p := range parts {
			if size += len(p); size > maxExportBytes {
				return fmt.Errorf("HTML transcript exceeds 64 MiB limit")
			}
			w.WriteString(p)
		}
		return nil
	}
	text := func(s string) string { return html.EscapeString(RedactSecrets(s)) }
	if err := put(htmlHead, "<h1>GoHarness session</h1><p class=\"meta\">Session ", text(s.ID), " · project ", text(s.Project), "</p>\n"); err != nil {
		return err
	}
	for i, record := range records {
		if record.Type != "message" {
			continue
		}
		var m core.Message
		if err := json.Unmarshal(record.Data, &m); err != nil {
			return fmt.Errorf("transcript message %d: %w", i+1, err)
		}
		var err error
		switch m.Role {
		case "user":
			who := "You"
			if m.Automatic {
				who = "GoHarness (automatic)"
			}
			err = put("<section class=\"user\"><h2>", who, "</h2><div class=\"text\">", text(m.Content), "</div></section>\n")
		case "assistant":
			if m.Content == "" && (!includeTools || len(m.ToolCalls) == 0) {
				continue
			}
			err = put("<section><h2>Assistant</h2>")
			if err == nil && m.Content != "" {
				err = put("<div class=\"text\">", text(m.Content), "</div>")
			}
			for _, c := range m.ToolCalls {
				if includeTools && err == nil {
					err = put("<div class=\"tool\">", text(c.Function.Name+" "+c.Function.Arguments), "</div>")
				}
			}
			if err == nil {
				err = put("</section>\n")
			}
		case "tool":
			if includeTools {
				err = put("<section><h2>Tool result</h2><div class=\"tool\">", text(m.Content), "</div></section>\n")
			}
		}
		if err != nil {
			return err
		}
	}
	if err := put("</body></html>\n"); err != nil {
		return err
	}
	return w.Flush()
}
