package tui

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Sanidhya14321/goh/internal/core"
	agent "github.com/Sanidhya14321/goh/internal/runtime"
	"github.com/Sanidhya14321/goh/internal/session"
	"github.com/Sanidhya14321/goh/internal/testutil"
	"github.com/Sanidhya14321/goh/internal/webfetch"
	"github.com/charmbracelet/x/ansi"
)

// The answer opens with a Markdown link, as replies about a repository do.
const repoAnswer = "[Fixture](https://github.com/example/fixture) is a **terminal coding assistant** written in Go. " +
	"It edits files with exact tools, runs `go test` through an approval step, and keeps durable sessions that can be resumed, forked and exported.\n\n" +
	"Its interface is a full-screen terminal application; a headless mode prints JSON or text for scripts, and an editor protocol lets other programs drive it."

// scriptedFetch fetches a page, then answers.
type scriptedFetch struct {
	url   string
	turns int
}

func (s *scriptedFetch) Complete(_ context.Context, _ []core.Message, _ []core.Tool, _ func(string)) (core.Reply, error) {
	s.turns++
	if s.turns > 1 {
		return core.Reply{Message: core.Message{Role: "assistant", Content: repoAnswer}, Finish: "stop"}, nil
	}
	c := core.Call{ID: "f1", Type: "function"}
	args, _ := json.Marshal(map[string]string{"url": s.url})
	c.Function.Name, c.Function.Arguments = "webfetch", string(args)
	return core.Reply{Message: core.Message{Role: "assistant", ToolCalls: []core.Call{c}}, Finish: "tool_calls"}, nil
}

func renderedText(log string, width int) ([]transcriptRow, string) {
	rows := transcriptRows(log, width, "", false)
	var text []string
	for _, row := range rows {
		text = append(text, row.text)
	}
	return rows, strings.Join(text, "\n")
}

// checkFetchTranscript checks a transcript holding the fetch and the answer
// in each /details mode.
func checkFetchTranscript(t *testing.T, m *model, host string) {
	t.Helper()
	_, shown := renderedText(m.displayLog(), 100)
	summary := "✓ http://" + host + "/example/fixture · HTTP 200 · " // https:// is the part dropped
	if !strings.Contains(shown, "▸ webfetch  http://"+host+"/example/fixture") || !strings.Contains(shown, summary) || !strings.Contains(shown, " markdown") {
		t.Fatalf("compact fetch result missing:\n%s", shown)
	}
	for _, page := range []string{"Title: GitHub", "# Fixture", "Exact file tools", "go install"} {
		if strings.Contains(shown, page) {
			t.Fatalf("page text %q shown by default:\n%s", page, shown)
		}
	}
	m.toggleDetails()
	if m.details != detailsPreviews || !strings.Contains(m.status, "previews") {
		t.Fatal("first /details should reveal page previews", m.status)
	}
	_, previews := renderedText(m.displayLog(), 100)
	for _, want := range []string{summary, "  Title: GitHub - example/fixture", "  # Fixture", "  [Fixture](https://github.com/example/fixture) is a **terminal", "  … 4 more lines"} {
		if !strings.Contains(previews, want) {
			t.Fatalf("preview lost %q:\n%s", want, previews)
		}
	}
	if strings.Contains(previews, "Pricing") || strings.Contains(previews, "Sign in") {
		t.Fatalf("navigation in preview:\n%s", previews)
	}
	m.toggleDetails()
	_, hidden := renderedText(m.displayLog(), 100)
	if m.details != detailsHidden || !strings.Contains(hidden, summary) || strings.Contains(hidden, "# Fixture") {
		t.Fatalf("hidden mode:\n%s", hidden)
	}
	m.toggleDetails()
	if m.details != detailsShown {
		t.Fatal("details should cycle back to shown")
	}

	for _, width := range []int{20, 32} {
		narrow, text := renderedText(m.displayLog(), width)
		for _, row := range narrow {
			if ansi.StringWidth(row.text) > width {
				t.Fatalf("width %d overflow: %q", width, row.text)
			}
		}
		flat := strings.Join(strings.Fields(text), "")
		if !strings.Contains(flat, strings.ReplaceAll(summary, " ", "")) || !strings.Contains(flat, "Fixture(https://github.com/example/fixture)isaterminal") || strings.Contains(text, "# Fixture") {
			t.Fatalf("width %d transcript:\n%s", width, text)
		}
	}

	rows, _ := renderedText(m.displayLog(), 100)
	var first *transcriptRow
	for i := range rows {
		if strings.HasPrefix(rows[i].text, "Fixture") {
			first = &rows[i]
			break
		}
	}
	if first == nil || first.fg != foreground || !strings.HasPrefix(first.text, "Fixture (https://github.com/example/fixture) is a terminal coding assistant") {
		t.Fatalf("leading-link paragraph not rendered as prose: %+v", first)
	}
	for _, row := range rows {
		if strings.Contains(row.text, "**") || strings.Contains(row.text, "](") || strings.Contains(row.text, "`go test`") {
			t.Fatalf("raw Markdown in the answer: %q", row.text)
		}
	}
}

func TestFetchedPageIsCompactLiveAndResumed(t *testing.T) {
	page, err := os.ReadFile("../webfetch/testdata/github_repo.html")
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(page)
	}))
	defer server.Close()
	host := strings.TrimPrefix(server.URL, "http://")

	cfg := core.Defaults()
	cfg.SessionDir, cfg.Model, cfg.ContextWindow = t.TempDir(), "fixture", 1<<20
	store, err := session.New(cfg.SessionDir, testutil.TempDir(t))
	if err != nil {
		t.Fatal(err)
	}
	a, err := agent.New(cfg, store)
	if err != nil {
		t.Fatal(err)
	}
	a.Web = webfetch.NewAllowingPrivate() // the fixture is a loopback server
	a.Provider = &scriptedFetch{url: server.URL + "/example/fixture"}
	approvals := 0
	a.Approve = func(_ context.Context, p core.Permission) (core.Decision, error) {
		if p.Tool == "webfetch" {
			approvals++
		}
		return core.Decision{Allow: true}, nil
	}
	m := &model{a: a, width: 100, height: 40, status: "ready"}
	a.Emit = func(e core.Event) { m.Update(e) }
	m.log += "\nYou: explain example/fixture\n"
	if status, err := a.Run(context.Background(), "explain example/fixture"); status != "completed" || err != nil {
		t.Fatal(status, err)
	}
	if approvals != 1 {
		t.Fatal("the fetch must still ask for approval", approvals)
	}
	t.Run("live", func(t *testing.T) { checkFetchTranscript(t, m, host) })

	id := store.ID
	store.Close()
	reopened, err := session.Open(cfg.SessionDir, id)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	resumed, err := agent.New(cfg, reopened)
	if err != nil {
		t.Fatal(err)
	}
	r := &model{a: resumed, width: 100, height: 40, status: "ready"}
	r.loadHistory(resumed.History, nil)
	t.Run("resumed", func(t *testing.T) { checkFetchTranscript(t, r, host) })
}

// A failed fetch stays a visible error line in every mode.
func TestFetchErrorsStayVisible(t *testing.T) {
	log := toolCallLine("webfetch", json.RawMessage(`{"url":"https://example.com/missing"}`)) + toolResultLog("[error] ", "webfetch HTTP 404 Not Found")
	for _, mode := range []detailsMode{detailsShown, detailsPreviews, detailsHidden} {
		rows, text := renderedText(withDetails(log, mode), 80)
		if !strings.Contains(text, "[error] webfetch HTTP 404 Not Found") || rows[len(rows)-2].fg != red {
			t.Fatalf("mode %d: %q", mode, text)
		}
	}
	// Shell output stays literal and fully shown by default.
	shell := toolResultLog("✓ ", `{"exit_code":0,"output":"# heading\n**kwargs\nhttps://x.test/a"}`)
	if _, text := renderedText(withDetails(shell, detailsShown), 80); !strings.Contains(text, "exit 0\n# heading\n**kwargs\nhttps://x.test/a") {
		t.Fatalf("shell output changed: %q", text)
	}
}

func TestFetchSummaryNamesTruncationAndArtifact(t *testing.T) {
	body := strings.Repeat("line of page text\n", 40)
	raw, _ := json.Marshal(map[string]any{"url": "https://www.example.com/docs/", "status_code": 200, "format": "text", "content": body, "truncated": true, "artifact_id": "fetch-1.log", "artifact_bytes": 719})
	summary, preview, ok := fetchedPage(string(raw))
	if !ok || summary != "example.com/docs · HTTP 200 · 719 bytes text · truncated · artifact fetch-1.log" {
		t.Fatalf("%q", summary)
	}
	lines := strings.Split(preview, "\n")
	if len(lines) != fetchPreviewLines+1 || lines[len(lines)-1] != "… 28 more lines (719 bytes of page text in artifact fetch-1.log)" {
		t.Fatalf("%q", preview)
	}
	if _, _, ok := fetchedPage(`{"path":"a.txt","content":"x"}`); ok {
		t.Fatal("a file read is not a fetched page")
	}
}

func TestInlineMarkdown(t *testing.T) {
	for in, want := range map[string]string{
		"[goh](https://github.com/x/goh) is **bold** and *it* and _it_":  "goh (https://github.com/x/goh) is bold and it and it",
		"[https://a.test](https://a.test) and <https://b.test>":          "https://a.test and https://b.test",
		"see [the **docs**](https://d.test \"Docs\") and [top](#top)":    "see the docs (https://d.test) and top",
		"![logo](https://i.test/l.png) `a*b*c` ``x ` y``":                "logo a*b*c x ` y",
		"def f(*args, **kwargs): 2 * 3 * 4 snake_case_name ~~old~~ \\*x": "def f(*args, **kwargs): 2 * 3 * 4 snake_case_name old *x",
		"[unclosed](link and [x] plain":                                  "[unclosed](link and [x] plain",
		"***both*** __strong__":                                          "both strong",
	} {
		if got := renderInline(in); got != want {
			t.Errorf("%q\n got %q\nwant %q", in, got, want)
		}
	}
	for line, tag := range map[string]bool{"[error] x": true, "[tool] read": true, "[GoHarness] note": true, "[goh](https://x) is": false, "[a][b] ref": false, "plain": false} {
		if bracketTag(line) != tag {
			t.Errorf("bracketTag(%q) = %v", line, !tag)
		}
	}
}

// Prose wraps between words at any width; only a token wider than a row is
// broken, and wide characters count as two cells.
func TestProseWrapsAtWordBoundaries(t *testing.T) {
	words := strings.Fields(renderInline(strings.ReplaceAll(repoAnswer, "\n\n", " ")))
	for _, width := range []int{24, 40, 60, 100, 160} {
		rows := transcriptRows(repoAnswer, width, "", false)
		var got []string
		all := ""
		for _, row := range rows {
			if w := ansi.StringWidth(row.text); w > width {
				t.Fatalf("width %d: row of %d cells: %q", width, w, row.text)
			}
			got = append(got, strings.Fields(row.text)...)
			all += strings.ReplaceAll(row.text, " ", "")
		}
		if all != strings.Join(words, "") {
			t.Fatalf("width %d lost text", width)
		}
		// Every word that fits a row is on one row.
		for _, word := range words {
			if ansi.StringWidth(word) <= width && !slices.Contains(got, word) {
				t.Fatalf("width %d: %q split across rows: %q", width, word, got)
			}
		}
		if width >= 40 && strings.Join(got, " ") != strings.Join(words, " ") {
			t.Fatalf("width %d: %q", width, got)
		}
	}
	long := "see https://example.com/" + strings.Repeat("a", 50) + " end"
	rows := proseRows(long, 20, "")
	if rows[0] != "see" || len(rows) != 5 || strings.Join(rows[1:], "") != "https://example.com/"+strings.Repeat("a", 50)+" end" {
		t.Fatalf("overlong token: %q", rows)
	}
	if rows := proseRows("界界界 界界界 界界界", 7, ""); len(rows) != 3 || rows[0] != "界界界" {
		t.Fatalf("display width: %q", rows)
	}
	item := transcriptRows("- "+strings.Repeat("word ", 12), 24, "", false)
	if !strings.HasPrefix(item[0].text, "• word") || !strings.HasPrefix(item[1].text, "  word") {
		t.Fatalf("list continuation not indented: %+v", item)
	}
	// Fenced code and approval diffs keep their exact characters.
	fence := strings.Repeat("\x60", 3)
	code := transcriptRows(fence+"go\nx := \"a b c d e f g h i j k l m n\"\n"+fence, 16, "", false)
	var joined string
	for _, row := range code[1 : len(code)-1] {
		joined += row.text
	}
	if joined != "x := \"a b c d e f g h i j k l m n\"" {
		t.Fatalf("code changed: %q", joined)
	}
	if diff := transcriptRows("+ **literal** [x](y)", 80, "", true); diff[0].text != "+ **literal** [x](y)" || diff[0].fg != green {
		t.Fatalf("approval diff changed: %+v", diff[0])
	}
}

func TestFetchPreviewShortensLinesBeforeTheBudget(t *testing.T) {
	raw, _ := json.Marshal(map[string]any{"url": "https://example.com/min", "status_code": 200, "format": "text", "content": strings.Repeat("x", 5000) + "\nsecond line"})
	_, preview, ok := fetchedPage(string(raw))
	lines := strings.Split(preview, "\n")
	if !ok || len(lines) != 2 || !strings.HasSuffix(lines[0], "…") || len([]rune(lines[0])) != fetchLineRunes+1 || lines[1] != "second line" {
		t.Fatalf("%q", preview)
	}
}

// Trimming a long transcript cuts at a line start, so a collapsed preview or
// detail never loses its marker and shows, and a code block keeps its opening.
func TestTrimmedTranscriptKeepsPreviewsHiddenAndFences(t *testing.T) {
	useDarkTheme(t) // code panels then have their own background
	m := &model{}
	m.log = fetchedPageLines("✓ ", "example.com · HTTP 200", strings.Repeat("hiddenpreview ", 12000)) + "\nanswer\n"
	m.trimLog()
	if text := withDetails(m.log, detailsShown); strings.Contains(text, "hiddenpreview") || strings.Contains(text, "\x00") || !strings.Contains(text, "answer") {
		t.Fatalf("trimmed preview leaked: %q", text[:min(len(text), 200)])
	}
	m = &model{}
	fence := strings.Repeat("\x60", 4)
	m.log = "intro\n" + fence + "go\n" + strings.Repeat("code := 1\n", 14000) + fence + "\nafter the block\n"
	m.trimLog()
	rows := transcriptRows(m.log, 60, "", false)
	if rows[0].bg != panel || rows[len(rows)-2].text != "after the block" || rows[len(rows)-2].bg == panel {
		t.Fatalf("fence lost after trim: first %+v, last %+v", rows[0], rows[len(rows)-2])
	}
}

// A block a message leaves open ends with that message, live and resumed;
// a fence closes only on a run of its own character at least as long.
func TestFencesStayInsideTheirMessage(t *testing.T) {
	useDarkTheme(t)
	fence := strings.Repeat("\x60", 3)
	reply := "Here is a start:\n" + fence + "go\nfunc partial() {"
	call := core.Call{ID: "c1", Type: "function"}
	call.Function.Name, call.Function.Arguments = "read_file", `{"path":"a.go"}`
	check := func(t *testing.T, log string) {
		t.Helper()
		for _, row := range transcriptRows(withDetails(log, detailsShown), 60, "", false) {
			// A swallowed line is painted as code: panel background, code text.
			if strings.Contains(row.text, "read_file") && row.bg == panel || strings.Contains(row.text, "You: next") && row.fg != blue {
				t.Fatalf("row swallowed by an unclosed fence: %+v", row)
			}
		}
	}
	live := &model{width: 80, height: 24}
	live.log = "\nYou: show code\n"
	live.Update(core.Event{Type: "assistant_updated", Data: map[string]any{"text": reply}})
	live.Update(core.Event{Type: "tool_started", Data: map[string]any{"name": "read_file", "arguments": json.RawMessage(call.Function.Arguments)}})
	live.Update(core.Event{Type: "tool_finished", Data: map[string]any{"result": map[string]any{"content": `{"path":"a.go","content":"package a"}`}}})
	live.log += "\nYou: next\n"
	t.Run("live", func(t *testing.T) { check(t, live.log) })
	resumed := &model{width: 80, height: 24}
	resumed.loadHistory([]core.Message{
		{Role: "user", Content: "show code\n" + fence},
		{Role: "assistant", Content: reply, ToolCalls: []core.Call{call}},
		{Role: "tool", Content: `{"Content":"{\"path\":\"a.go\",\"content\":\"package a\"}","Error":false}`},
		{Role: "user", Content: "next"},
	}, nil)
	t.Run("resumed", func(t *testing.T) { check(t, resumed.log) })

	outer := strings.Repeat("\x60", 4)
	rows := transcriptRows(outer+"md\n"+fence+"go\ninner\n"+fence+"\n~~~\n"+outer+"\nprose", 60, "", false)
	var code []string
	for _, row := range rows {
		if row.bg == panel {
			code = append(code, row.text)
		}
	}
	if strings.Join(code, "|") != "┌─ md|"+fence+"go|inner|"+fence+"|~~~|└─" || rows[len(rows)-1].text != "prose" {
		t.Fatalf("delimiter length not respected: %q", code)
	}
}

// A very long token is broken in one pass and the screenshot flow fits a
// narrow terminal, live and resumed.
func TestLongTokensAndNarrowTerminals(t *testing.T) {
	token := strings.Repeat("a界", 100000)
	start := time.Now()
	rows := proseRows("x "+token, 40, "")
	if time.Since(start) > 2*time.Second {
		t.Fatalf("long token reflow took %s", time.Since(start))
	}
	if rows[0] != "x" || strings.Join(rows[1:], "") != token {
		t.Fatal("long token changed")
	}
	for _, row := range rows {
		if ansi.StringWidth(row) > 40 {
			t.Fatalf("row too wide: %d", ansi.StringWidth(row))
		}
	}
}

func useDarkTheme(t *testing.T) {
	t.Helper()
	applyTheme("dark")
	t.Cleanup(func() { applyTheme("") })
	if panel == background {
		t.Fatal("dark theme should paint code panels")
	}
}

// answerer replies to prompts and summarizes when compaction asks.
type answerer struct{ n int }

func (s *answerer) Complete(_ context.Context, _ []core.Message, tools []core.Tool, _ func(string)) (core.Reply, error) {
	s.n++
	text := fmt.Sprintf("answer number %d: %s", s.n, strings.Repeat("detail ", 80))
	if len(tools) == 0 {
		text = "Summary: the earlier work is retained."
	}
	return core.Reply{Message: core.Message{Role: "assistant", Content: text}, Finish: "stop"}, nil
}

// A reopened chat shows the journaled conversation, not the compacted model
// context that replaced it.
func TestReopenedChatAfterCompactionShowsJournal(t *testing.T) {
	cfg := core.Defaults()
	cfg.SessionDir, cfg.Model, cfg.ContextWindow = t.TempDir(), "fixture", 131072
	store, err := session.New(cfg.SessionDir, testutil.TempDir(t))
	if err != nil {
		t.Fatal(err)
	}
	a, err := agent.New(cfg, store)
	if err != nil {
		t.Fatal(err)
	}
	a.Provider = &answerer{}
	for _, prompt := range []string{"first question about parsing", "second question about tests", "third question"} {
		if status, err := a.Run(context.Background(), prompt); status != "completed" || err != nil {
			t.Fatal(status, err)
		}
	}
	if status, err := a.Compact(context.Background()); status != "completed" || err != nil {
		t.Fatal("compaction", status, err)
	}
	id := store.ID
	store.Close()
	reopened, err := session.Open(cfg.SessionDir, id)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	resumed, err := agent.New(cfg, reopened)
	if err != nil {
		t.Fatal(err)
	}
	inContext := ""
	for _, m := range resumed.History {
		inContext += m.Content
	}
	if strings.Contains(inContext, "answer number 1") {
		t.Fatal("precondition: compaction should have removed the first answer from the model context")
	}
	m := &model{a: resumed, width: 100, height: 40}
	m.loadHistory(screenHistory(resumed), nil)
	for _, want := range []string{"You: first question about parsing", "answer number 1", "You: second question about tests", "You: third question", "answer number 3"} {
		if !strings.Contains(m.log, want) {
			t.Fatalf("reopened chat lost %q:\n%s", want, m.log)
		}
	}
	// Bounded: a long journal renders only its recent end.
	for i := range 400 {
		if err := reopened.Append("message", core.Message{Role: "user", Content: fmt.Sprintf("prompt %d %s", i, strings.Repeat("x", 2000))}); err != nil {
			t.Fatal(err)
		}
	}
	recent := screenHistory(resumed)
	if len(recent) >= 400 || len(recent) < 50 || !strings.HasPrefix(recent[len(recent)-1].Content, "prompt 399 ") {
		t.Fatalf("history not bounded to its recent end: %d messages", len(recent))
	}
}
