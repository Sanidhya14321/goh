package tui

import (
	"encoding/json"
	"fmt"
	"image/color"
	"os"
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/x/ansi"
)

// Inline Markdown styling travels through wrapping as zero-width format
// characters (they survive Safe and measure zero cells) and becomes trusted
// color sequences only when a row is cached for display. Model text cannot
// forge them: transcriptRows removes them from content before rendering.
const (
	codeOn  = "\u2061"
	codeOff = "\u2062"
	boldOn  = "\u2063"
	boldOff = "\u2064"
	// diffMark starts a diff line written by editDiffLines: "+", "-" or " "
	// follows it.
	diffMark = verbatimMark + verbatimMark
)

var styleMarks = strings.NewReplacer(codeOn, "", codeOff, "", boldOn, "", boldOff, "")

// hasStyleMarks reports a style mark (U+2061–U+2064, encoded E2 81 A1–A4)
// with a byte search; the word joiner (E2 81 A0) is on most log lines.
func hasStyleMarks(s string) bool {
	for {
		i := strings.Index(s, "\xe2\x81")
		if i < 0 || i+2 >= len(s) {
			return false
		}
		if c := s[i+2]; c >= 0xa1 && c <= 0xa4 {
			return true
		}
		s = s[i+2:]
	}
}

// styledRow turns marked text into trusted SGR for a row drawn in fg. Spans
// left open at the end of a row are reopened on the next by the caller
// passing the returned state back in.
func styledRow(text, fg string, code, bold bool) (styled string, codeOpen, boldOpen bool) {
	noColor := os.Getenv("NO_COLOR") != ""
	restore := "\x1b[39m"
	if fg != "" {
		restore = "\x1b[38;2;" + fg + "m"
	}
	var b strings.Builder
	if !noColor {
		if code {
			b.WriteString("\x1b[38;2;" + accent + "m")
		}
		if bold {
			b.WriteString("\x1b[1m")
		}
	}
	for _, r := range text {
		switch string(r) {
		case codeOn:
			code = true
			if !noColor {
				b.WriteString("\x1b[38;2;" + accent + "m")
			}
		case codeOff:
			code = false
			if !noColor {
				b.WriteString(restore)
			}
		case boldOn:
			bold = true
			if !noColor {
				b.WriteString("\x1b[1m")
			}
		case boldOff:
			bold = false
			if !noColor {
				b.WriteString("\x1b[22m")
			}
		default:
			b.WriteRune(r)
		}
	}
	if !noColor && (code || bold) {
		b.WriteString(restore + "\x1b[22m")
	}
	return b.String(), code, bold
}

// toolVerbs name tools by what they do; unknown tools keep their own name.
var toolVerbs = map[string]string{
	"read_file": "Read", "write_file": "Write", "create_file": "Create", "edit_file": "Edit",
	"apply_patch": "Patch", "shell": "Shell", "grep": "Search", "glob": "Find", "search": "Search",
	"list_dir": "List", "list_files": "List", "webfetch": "Fetch", "websearch": "Web search",
	"todo_write": "Tasks", "todowrite": "Tasks", "task": "Agent", "question": "Question",
	"skill": "Skill", "read_artifact": "Read output", "job_output": "Job output", "job_kill": "Stop job",
	"lsp": "Code intel", "repo_map": "Map",
}

func toolVerb(name string) string {
	if verb, ok := toolVerbs[name]; ok {
		return verb
	}
	return name
}

const diffRowLimit = 12

// editDiffLines shows what a successful file change did, from the call's own
// arguments: removed and added lines for an edit, the first lines of a
// written file, the changed lines of a patch. Empty for other tools.
func editDiffLines(name string, arguments json.RawMessage) string {
	var args struct {
		OldText   string `json:"old_text"`
		NewText   string `json:"new_text"`
		Content   string `json:"content"`
		PatchText string `json:"patchText"`
		Patch     string `json:"patch"`
		StartLine int    `json:"start_line"`
	}
	if json.Unmarshal(arguments, &args) != nil {
		return ""
	}
	var lines []string
	switch name {
	case "edit_file":
		old, added := splitLines(args.OldText), splitLines(args.NewText)
		// Lines both sides share at the ends are context, not change.
		for len(old) > 0 && len(added) > 0 && old[0] == added[0] {
			old, added = old[1:], added[1:]
		}
		for len(old) > 0 && len(added) > 0 && old[len(old)-1] == added[len(added)-1] {
			old, added = old[:len(old)-1], added[:len(added)-1]
		}
		for _, line := range old {
			lines = append(lines, "-"+line)
		}
		for _, line := range added {
			lines = append(lines, "+"+line)
		}
	case "write_file", "create_file":
		for _, line := range splitLines(args.Content) {
			lines = append(lines, "+"+line)
		}
	case "apply_patch":
		patch := args.PatchText
		if patch == "" {
			patch = args.Patch
		}
		for _, line := range strings.Split(patch, "\n") {
			if strings.HasPrefix(line, "+") || strings.HasPrefix(line, "-") {
				lines = append(lines, line)
			}
		}
	default:
		return ""
	}
	if len(lines) == 0 {
		return ""
	}
	var b strings.Builder
	for i, line := range lines {
		if i == diffRowLimit {
			fmt.Fprintf(&b, "%s %s\n", diffMark, plural(len(lines)-i, "more changed line", "more changed lines"))
			break
		}
		b.WriteString(diffMark + Safe(strings.ReplaceAll(oneLineRaw(line), "\t", "    ")) + "\n")
	}
	return b.String()
}

func splitLines(s string) []string {
	s = strings.TrimSuffix(s, "\n")
	if s == "" {
		return nil
	}
	return strings.Split(s, "\n")
}

func oneLineRaw(s string) string { return strings.TrimRight(s, "\r") }

// diffCounts is "+3 −1" for a diff written by editDiffLines.
func diffCounts(diff string) string {
	added, removed := 0, 0
	for _, line := range strings.Split(diff, "\n") {
		switch {
		case strings.HasPrefix(line, diffMark+"+"):
			added++
		case strings.HasPrefix(line, diffMark+"-"):
			removed++
		}
	}
	if strings.Contains(diff, "more changed line") {
		return ""
	}
	return fmt.Sprintf("+%d −%d", added, removed)
}

// Tints for the default theme, derived from the terminal's own background
// once it answers; empty until then (and in terminals that do not answer),
// which draws no fill at all.
var (
	diffAddBg = ""
	diffDelBg = ""
)

func rgb(c color.Color) (r, g, b int) {
	r16, g16, b16, _ := c.RGBA()
	return int(r16 >> 8), int(g16 >> 8), int(b16 >> 8)
}

func parseRGB(s string) (r, g, b int, ok bool) {
	_, err := fmt.Sscanf(s, "%d;%d;%d", &r, &g, &b)
	return r, g, b, err == nil
}

// mix moves base toward target by weight (0..1).
func mix(r, g, b int, target string, weight float64) string {
	tr, tg, tb, ok := parseRGB(target)
	if !ok {
		return ""
	}
	f := func(a, t int) int { return int(float64(a) + (float64(t)-float64(a))*weight + 0.5) }
	return fmt.Sprintf("%d;%d;%d", f(r, tr), f(g, tg), f(b, tb))
}

// deriveTints sets the panel and diff fills from a background color: a
// slightly lighter (or, on light backgrounds, darker) panel, and diff rows
// washed toward green and red.
func deriveTints(r, g, b int, setPanel bool) {
	toward := "255;255;255"
	weight := 0.07
	if (r*299+g*587+b*114)/1000 > 128 {
		toward, weight = "0;0;0", 0.05
	}
	if setPanel {
		panel = mix(r, g, b, toward, weight)
	}
	diffAddBg = mix(r, g, b, green, 0.18)
	diffDelBg = mix(r, g, b, red, 0.18)
}

// setTerminalBackground applies the terminal's reported background to the
// default theme. Other themes carry their own colors.
func setTerminalBackground(c color.Color) bool {
	if currentTheme != "terminal" || c == nil {
		return false
	}
	r, g, b := rgb(c)
	terminalBackground = fmt.Sprintf("%d;%d;%d", r, g, b)
	deriveTints(r, g, b, true)
	return true
}

// terminalBackground is the reported background ("" when unknown), kept so
// switching back to the default theme restores its tints.
var terminalBackground = ""

// keyHints draws "Enter send  ·  Tab agents" with each key bright and its
// label muted, so the eye finds the keys. Hints are trusted, fixed text.
func keyHints(text string, width int) string { return keyHintsOn(text, width, background) }

func keyHintsOn(text string, width int, background string) string {
	text = fitHints(text, width)
	if os.Getenv("NO_COLOR") != "" {
		return cell(text, width, "", "")
	}
	sep := "  ·  "
	if !strings.Contains(text, sep) {
		sep = "  "
	}
	var b strings.Builder
	for i, group := range strings.Split(text, sep) {
		if i > 0 {
			b.WriteString(paint(sep, muted, background))
		}
		key, label, ok := strings.Cut(group, " ")
		if !ok || !looksLikeKey(key) {
			b.WriteString(paint(group, muted, background))
			continue
		}
		b.WriteString(paint(key, foreground, background) + paint(" "+label, muted, background))
	}
	used := ansi.StringWidth(text)
	return b.String() + paint(strings.Repeat(" ", max(0, width-used)), foreground, background)
}

func looksLikeKey(s string) bool {
	if s == "/" || strings.HasPrefix(s, "/") {
		return true
	}
	for _, prefix := range []string{"Ctrl+", "Alt+", "Shift+", "Enter", "Esc", "Tab", "PgUp", "PgDn", "↑", "↓", "y", "n", "s", "a", "Space"} {
		if strings.HasPrefix(s, prefix) {
			return true
		}
	}
	return len(s) == 1
}

// textWidth is ansi.StringWidth with a fast path: text made only of ASCII,
// color sequences and the narrow symbols the transcript draws (✓ ▸ • · … box drawing,
// arrows, Latin letters) is one cell per rune, so it skips grapheme
// segmentation, which otherwise dominates rendering time. Anything else
// (wide, combining or emoji characters, escapes) takes the exact path.
func textWidth(s string) int {
	n := 0
	for i := 0; i < len(s); {
		c := s[i]
		if c >= 0x20 && c < 0x7f {
			n++
			i++
			continue
		}
		if c == 0x1b && i+1 < len(s) && s[i+1] == '[' {
			// A CSI sequence (our own colors) takes no cells: skip to its
			// final byte.
			j := i + 2
			for j < len(s) && s[j] >= 0x20 && s[j] < 0x40 {
				j++
			}
			if j < len(s) && s[j] >= 0x40 && s[j] <= 0x7e {
				i = j + 1
				continue
			}
			return ansi.StringWidth(s)
		}
		r, size := utf8.DecodeRuneInString(s[i:])
		switch {
		case r >= 0x2060 && r <= 0x2064: // word joiner and style marks
		case narrowRune(r):
			n++
		default:
			return ansi.StringWidth(s)
		}
		i += size
	}
	return n
}

// narrowRune reports runes that are always one cell and never combine with
// a neighbor; TestTextWidthMatchesAnsi checks every one against ansi.
func narrowRune(r rune) bool {
	switch {
	case r >= 0xa1 && r <= 0x24f && r != 0xad: // Latin-1 and Latin Extended, not the soft hyphen
		return true
	case r >= 0x2010 && r <= 0x2027, r >= 0x2030 && r <= 0x205e: // dashes, quotes, bullets, ellipsis
		return true
	case r >= 0x2190 && r <= 0x21ff: // arrows
		return true
	case r >= 0x2500 && r <= 0x25ff && r != 0x25fd && r != 0x25fe: // box drawing, blocks, shapes (two are emoji)
		return true
	case r == 0x2713 || r == 0x2717 || r == 0x2212: // ✓ ✗ −
		return true
	}
	return false
}
