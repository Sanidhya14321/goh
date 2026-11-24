package tui

import (
	"fmt"
	"os"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

// Only these trusted theme sequences reach the terminal. User/model content is
// sanitized before styling, including text received inside code fences.
// The values come from the fixed theme table in theme.go; an empty value
// leaves the terminal's own color. Defaults are the "terminal" theme.
var (
	background = ""
	panel      = ""
	foreground = ""
	muted      = "128;128;128"
	accent     = "214;120;60"
	blue       = "60;130;230"
	green      = "40;160;80"
	red        = "210;70;80"
)

func paint(text, fg, bg string) string {
	if os.Getenv("NO_COLOR") != "" || (fg == "" && bg == "") {
		return text
	}
	style := ""
	if fg != "" {
		style += "\x1b[38;2;" + fg + "m"
	}
	if bg != "" {
		style += "\x1b[48;2;" + bg + "m"
	}
	return style + text + "\x1b[0m"
}

func cell(text string, width int, fg, bg string) string {
	text = strings.ReplaceAll(strings.ReplaceAll(Safe(text), "\t", "    "), "\n", " ")
	w := textWidth(text)
	if w > width {
		text = ansi.Truncate(text, max(0, width), "…") // a cut shows, not a word fragment
		w = ansi.StringWidth(text)
	}
	return paint(text+strings.Repeat(" ", max(0, width-w)), fg, bg)
}

// fitHints shortens a row of key hints to width by dropping whole hints
// from the end, so a narrow terminal never shows half a key ("Ctrl+").
// Hints are separated by "  ·  " or, failing that, two spaces.
func fitHints(text string, width int) string {
	if ansi.StringWidth(text) <= width {
		return text
	}
	sep := "  ·  "
	if !strings.Contains(text, sep) {
		sep = "  "
	}
	items := strings.Split(text, sep)
	for len(items) > 1 {
		items = items[:len(items)-1]
		if joined := strings.Join(items, sep); ansi.StringWidth(joined) <= width {
			return joined
		}
	}
	return text // a single hint: cell truncates it with an ellipsis
}

func canvas(rows []string, width, height int) tea.View {
	rows = rows[:min(len(rows), height)]
	for len(rows) < height {
		rows = append(rows, "")
	}
	for i, row := range rows {
		// Rows already contain our own styling. Padding is always a trusted
		// sequence and never reinterprets untrusted escape codes.
		w := textWidth(row)
		if w > width {
			row = ansi.Truncate(row, width, "")
			w = ansi.StringWidth(row)
		}
		rows[i] = row + paint(strings.Repeat(" ", max(0, width-w)), foreground, background)
	}
	view := tea.NewView(strings.Join(rows, "\n"))
	view.AltScreen = true
	view.MouseMode = mouseMode()
	return view
}

func (m *model) View() tea.View {
	width, height := m.width, m.height
	if width < 1 || height < 1 {
		return blankView()
	}
	if m.export != nil {
		return m.exportView(width, height)
	}
	if m.fork != nil {
		m.fork.width, m.fork.height = width, height
		return m.fork.View()
	}
	if m.changesPicker != nil {
		m.changesPicker.width, m.changesPicker.height = width, height
		return m.changesPicker.View()
	}
	if m.jobsPicker != nil {
		m.jobsPicker.width, m.jobsPicker.height = width, height
		return m.jobsPicker.View()
	}
	if m.copyPicker != nil {
		m.copyPicker.width, m.copyPicker.height = width, height
		return m.copyPicker.View()
	}
	if m.themePicker != nil {
		m.themePicker.width, m.themePicker.height = width, height
		return m.themePicker.View()
	}
	if m.timeline != nil {
		return m.timelineView(width, height)
	}
	if m.infoPanel != nil {
		return m.statusView(width, height)
	}
	if m.palette {
		return m.paletteView(width, height)
	}
	if width < 60 || height < 18 {
		return m.compactView()
	}

	modelName, agentName, project, sessionID := "No model connected", "build", "", ""
	if m.a != nil {
		modelName, agentName = m.a.Config.Model, m.a.Config.Agent
		if m.servedModel != "" && m.servedModel != modelName {
			modelName += " (served by " + m.servedModel + ")"
		}
		project, sessionID = m.a.Store.Project, m.a.Store.ID
	}
	if m.fix == FixWorkspace {
		modelName = "No project folder open"
	}
	if m.fix == FixSessions {
		modelName = "Session unavailable"
	}
	if project == "" {
		project, _ = os.Getwd()
	}
	right := "goh"
	if version := Version; version != "" {
		right += " " + version
	}
	if line := m.contextLine(); line != "" {
		right = strings.TrimPrefix(line, "Context: ") + " context  ·  " + right
	}
	right += " "
	rightWidth := min(ansi.StringWidth(right), max(0, width/2))
	footer := cell(" "+Safe(shortPath(project, width-rightWidth-3)), width-rightWidth, muted, background) +
		cell(right, rightWidth, muted, background)
	home := m.pending == nil && m.question == nil && !m.active &&
		len(m.todos) == 0 && (m.a == nil || len(m.a.History) <= 1) && (m.log == "" || m.blocked != "")

	columns := width - 4
	sideWidth := 0
	if !home && width >= 112 && !m.hideSidebar {
		sideWidth = 30
		columns -= sideWidth + 3
	}
	promptWidth := columns
	if home {
		promptWidth = min(76, columns)
	}
	left := 2
	if home {
		left = (width - promptWidth) / 2
	}

	draftRows := m.inputRows(promptWidth - 5)
	placeholder := false
	if m.input == "" {
		draft := "Ask anything about your project…"
		switch {
		case m.pending != nil:
			draft = "Waiting for your approval (keys above)"
		case m.question != nil:
			draft = "Answer the question above"
		case m.active:
			draft = "Working… type to queue a follow-up"
		case len(m.queue) > 0 && m.queueHeld != "":
			draft = fmt.Sprintf("%d queued prompt(s) held: Enter sends the next · ↑ edits the newest", len(m.queue))
		}
		if m.blocked != "" {
			_, step := m.setupStep()
			draft = "Press Enter to " + step
		}
		draftRows = terminalRows("▏ "+draft, promptWidth-5)
		placeholder = true
	}
	draftRows = draftRows[max(0, len(draftRows)-3):]
	promptRows := m.fileCompletionRows(promptWidth, min(6, max(1, height-16)))
	if matches := m.slashMatches(); m.hasSlashSuggestions() {
		current := 0
		for i, match := range matches {
			if match.command == m.input {
				current = i
			}
		}
		first := max(0, min(current-1, len(matches)-3))
		for i := first; i < min(len(matches), first+3); i++ {
			if matches[i].command == m.input {
				promptRows = append(promptRows, cell("› "+matches[i].command+"  "+matches[i].label, promptWidth, onAccent, accent))
				continue
			}
			promptRows = append(promptRows, cell("  "+matches[i].command+"  "+matches[i].label, promptWidth, muted, panel))
		}
		exact := false
		for _, match := range matches {
			exact = exact || match.command == m.input
		}
		hint := "  Tab complete"
		if len(matches) > 1 {
			hint = fmt.Sprintf("  Tab next (%d matches)", len(matches))
		}
		if exact {
			hint += " · Enter run" // only a complete command runs
		}
		promptRows = append(promptRows, cell(hint, promptWidth, accent, panel))
	}
	if !home {
		// What is happening now sits where the eye is: just above the input.
		promptRows = append(promptRows, cell(m.statusLine(), promptWidth, statusColor(m), background))
	}
	draftColor := foreground
	if placeholder {
		draftColor = muted
	}
	promptRows = append(promptRows, cell("▌", 1, blue, panel)+cell("", promptWidth-1, foreground, panel))
	for _, row := range draftRows {
		promptRows = append(promptRows, cell("▌", 1, blue, panel)+cell("  "+row, promptWidth-1, draftColor, panel))
	}
	promptRows = append(promptRows,
		cell("▌", 1, blue, panel)+cell("", promptWidth-1, foreground, panel),
		cell("▌", 1, blue, panel)+agentLine(agentName, modelName, m.accessLabel(), promptWidth-1))
	controls := "Enter send  Alt+Enter newline  "
	if key := m.keyLabel("command_palette"); key != "" {
		controls += key + " commands  "
	}
	controls += "Ctrl+D exit  Tab agents  Shift+Tab access"
	if m.active {
		controls = m.runControls()
	}
	if m.blocked != "" {
		_, step := m.setupStep()
		palette := "/ commands"
		if key := m.keyLabel("command_palette"); key != "" {
			palette = key + " commands"
		}
		controls = "Enter " + step + "  " + palette + "  /providers  /settings  Ctrl+D exit"
	}
	if m.scrollOffset > 0 && m.pending == nil && m.question == nil {
		// Without this a reader cannot tell that newer output is below.
		controls = "↑ Scrolled up · PgDn or wheel for newer"
		if !m.active {
			controls += " · Esc jump to latest"
		}
	}
	if !m.lastInterrupt.IsZero() && time.Since(m.lastInterrupt) < 2*time.Second {
		controls = "Press Ctrl+C again to exit"
	}
	promptRows = append(promptRows, cell("   ", 3, muted, background)+keyHints(controls, promptWidth-3))

	var rows []string
	if home {
		logo := []string{
			" ██████╗  ██████╗  ██╗  ██╗",
			"██╔════╝ ██╔═══██╗ ██║  ██║",
			"██║  ███╗██║   ██║ ███████║",
			"██║   ██║██║   ██║ ██╔══██║",
			"╚██████╔╝╚██████╔╝ ██║  ██║",
			" ╚═════╝  ╚═════╝  ╚═╝  ╚═╝",
		}
		if height < 24 {
			logo = []string{"g o h"}
		}
		top := max(1, (height-len(promptRows)-len(logo)-5)/2)
		for range top {
			rows = append(rows, "")
		}
		for _, line := range logo {
			padding := strings.Repeat(" ", max(0, (width-ansi.StringWidth(line))/2))
			rows = append(rows, paint(padding+line, foreground, background))
		}
		rows = append(rows, cell(strings.Repeat(" ", max(0, (width-9)/2))+"GoHarness", width, muted, background), "")
	} else {
		rows = append(rows, "")
		if len(m.todos) > 0 && sideWidth == 0 {
			remaining := 0
			for _, todo := range m.todos {
				if todo.Status != "completed" {
					remaining++
				}
			}
			rows = append(rows, cell(fmt.Sprintf("  Tasks: %d remaining / %d total", remaining, len(m.todos)), columns+4, muted, background))
		}
	}

	contentHeight := max(0, height-len(rows)-len(promptRows)-2)
	if home {
		contentHeight = min(3, contentHeight)
	}
	body, review := m.displayLog(), ""
	if m.pending != nil {
		body = relativizePreview(m.pending.request.Preview, project)
		review = "Allow " + approvalTitle(m.pending.request.Tool, body) + "?  ·  " + m.approvalChoices("  ·  ") + "  ·  PgUp/PgDn scroll"
	} else if m.question != nil {
		body, review = m.question.view(), "INTERACTIVE QUESTION · answers grant no permissions"
	} else if home {
		body = ""
		if m.blocked != "" {
			// The first paragraph: what is wrong. The hints say what fixes it.
			body, _, _ = strings.Cut(m.blocked, "\n\n")
			var lines []string
			for _, line := range strings.Split(body, "\n") {
				lines = append(lines, wordWrap(line, max(1, columns-2))...)
			}
			body = strings.Join(lines, "\n")
		}
	}
	if review != "" {
		// The conversation stays visible: the approval or question docks
		// as a panel just above the composer, where the keys are typed.
		header := keyRows(review, "  ·  ", columns-2)
		header = header[:min(len(header), max(1, contentHeight-1))]
		space := max(1, contentHeight-len(header)-1) // one blank row above the composer
		// The decision comes first: the whole preview when it fits, with the
		// conversation in the rows left over (at least three, or none).
		need := len(m.wrapReview(body, columns-2))
		limit := space
		if need <= space-4 {
			limit = need
		}
		preview := m.reviewRows(body, columns-2, limit)
		transcript := space - len(preview) - 1
		if transcript < 3 {
			transcript = 0
		}
		m.reviewHidden = len(preview) == 0
		if transcript > 0 {
			lines := m.visibleRows(m.displayLog(), columns-2, transcript, false)
			for count := len(lines); count < transcript; count++ {
				rows = append(rows, "")
			}
			for _, row := range lines {
				rows = append(rows, transcriptCell(row, columns+4))
			}
			rows = append(rows, "")
		}
		for i, row := range header {
			lead := paint("  ", foreground, background) + cell("▌", 1, accent, panel)
			if i == 0 {
				// The question in accent, its keys styled like every hint row.
				question, keys, _ := strings.Cut(row, "?  ·  ")
				if keys == "" || os.Getenv("NO_COLOR") != "" {
					rows = append(rows, lead+cell(" "+row, columns+1, accent, panel))
					continue
				}
				head := " " + question + "?  "
				rows = append(rows, lead+paint(head, accent, panel)+panelHints(keys, columns+1-ansi.StringWidth(head)))
				continue
			}
			rows = append(rows, lead+paint(" ", foreground, panel)+panelHints(row, columns))
		}
		for _, row := range preview {
			row.gutter, row.bar = "▌", accent
			if row.bg == background {
				row.bg = panel
			}
			rows = append(rows, transcriptCell(row, columns+4))
		}
		used := len(header) + len(preview)
		if transcript > 0 {
			used += transcript + 1
		}
		for count := used; count < contentHeight; count++ {
			rows = append(rows, "")
		}
	} else {
		lines := m.visibleRows(body, columns-2, contentHeight, false)
		m.reviewHidden = false
		needle := ""
		if m.find != nil {
			needle = strings.ToLower(m.find.query)
		}
		for _, row := range lines {
			if needle != "" && strings.Contains(strings.ToLower(row.text), needle) {
				// The marker keeps matches visible when NO_COLOR disables colors.
				rows = append(rows, cell("▸ "+row.text, columns+4, onAccent, accent))
				continue
			}
			rows = append(rows, transcriptCell(row, columns+4))
		}
		for count := len(lines); count < contentHeight; count++ {
			rows = append(rows, "")
		}
	}
	for _, row := range promptRows {
		rows = append(rows, paint(strings.Repeat(" ", left), foreground, background)+row)
	}
	if sideWidth > 0 {
		sidebar := m.sidebar(sessionID, modelName)
		for i := range rows {
			side := ""
			if i < len(sidebar) {
				side = sidebar[i]
			}
			row, w := rows[i], textWidth(rows[i])
			if w > width-sideWidth-1 {
				row = ansi.Truncate(row, width-sideWidth-1, "")
				w = ansi.StringWidth(row)
			}
			rows[i] = row + paint(strings.Repeat(" ", max(0, width-sideWidth-1-w)), foreground, background) +
				cell("│", 1, muted, background) + cell("  "+side, sideWidth, sidebarColor(side), background)
		}
	}
	for len(rows) < height-1 {
		rows = append(rows, "")
	}
	rows = append(rows, footer)
	return canvas(rows, width, height)
}

// wrapReview wraps an approval preview or question once per text and width.
func (m *model) wrapReview(text string, width int) []transcriptRow {
	if m.previewText != text || m.previewWidth != width || m.previewRows == nil {
		m.previewText, m.previewWidth = text, width
		m.previewRows = transcriptRows(text, width, "", true)
	}
	return m.previewRows
}

// noteReviewWidth resets review progress when the review is wrapped at a
// different width (a resize, or a switch between the wide and compact
// layouts). reviewSeen counts rows, and the same rows cover different text
// at another width, so progress cannot carry over; the review starts again
// from the top. A review that fits on screen is seen again at once.
func (m *model) noteReviewWidth(width int) {
	if m.reviewWidth != width {
		if m.reviewWidth != 0 {
			m.reviewSeen, m.reviewOffset = 0, 0
		}
		m.reviewWidth = width
	}
}

// reviewRows returns the visible part of an approval preview or question:
// at most height rows, scrolled by reviewOffset, sized to its content so a
// short preview leaves room for the conversation above it.
func (m *model) reviewRows(text string, width, height int) []transcriptRow {
	m.noteReviewWidth(width)
	lines := m.wrapReview(text, width)
	height = min(height, len(lines))
	m.reviewHeight = height
	start := min(m.reviewOffset, max(0, len(lines)-height))
	m.reviewOffset = start // overshooting PgDn must not make PgUp feel dead
	end := min(len(lines), start+height)
	if start <= m.reviewSeen && height > 0 {
		m.reviewSeen = max(m.reviewSeen, end)
	}
	m.reviewTotal = len(lines)
	return lines[start:end]
}

// Reflow only the visible transcript tail. Cached rows are reused during typing
// and permission decisions. Scrolling expands the window into older content.
func (m *model) visibleRows(text string, width, height int, review bool) []transcriptRow {
	if !review {
		m.anchorScroll(text, width)
	}
	offset := m.scrollOffset
	limit := max(4096, (height+offset+4)*width*4)
	if review {
		offset, limit = 0, 0
	}
	if m.wrappedText != text || m.wrappedWidth != width || m.wrappedRows == nil || m.wrappedWindow != limit || m.wrappedReview != review {
		m.wrappedText, m.wrappedWidth = text, width
		m.wrappedWindow, m.wrappedReview = limit, review
		fence := ""
		if !review && len(text) > limit {
			start := len(text) - limit
			// Keep the whole logical line and the preceding fence state.
			start = strings.LastIndexByte(text[:start], '\n') + 1
			fence, opening := fenceContext(text[:start])
			if fence != "" {
				start = opening
			} else {
				for start > 0 {
					previous := strings.LastIndexByte(text[:start-1], '\n') + 1
					if !strings.Contains(text[previous:start], "|") {
						break
					}
					start = previous
				}
			}
			text = strings.ToValidUTF8(text[start:], "�")
		}
		m.wrappedRows = transcriptRows(text, width, fence, review)
	}
	lines := m.wrappedRows
	if !review {
		m.lastTranscriptHeight = height
		offset = min(offset, max(0, len(lines)-height))
		m.scrollOffset = offset
	}
	end := max(0, len(lines)-offset)
	start := max(0, end-height)
	if review {
		m.noteReviewWidth(width)
		m.reviewHeight = height
		start = min(m.reviewOffset, max(0, len(lines)-height))
		m.reviewOffset = start // overshooting PgDn must not make PgUp feel dead
		end = min(len(lines), start+height)
		if start <= m.reviewSeen && height > 0 {
			m.reviewSeen = max(m.reviewSeen, end)
		}
		m.reviewTotal = len(lines)
	}
	return lines[start:end]
}

func (m *model) sidebar(sessionID, modelName string) []string {
	rows := []string{"SESSION"}
	if m.sessionTitle != "" {
		rows = append(rows, m.sessionTitle)
	}
	if len(sessionID) > 15 {
		sessionID = sessionID[len(sessionID)-12:]
	}
	rows = append(rows, Safe(sessionID), "", "MODEL", Safe(modelName))
	if t := humanTokens(m.usage); t != "" {
		rows = append(rows, t)
		if m.usageIn+m.usageOut > 0 {
			// Input is mostly instructions and tool definitions sent with
			// every request; the split shows where the tokens went.
			rows = append(rows, "  "+strings.TrimSuffix(humanTokens(m.usageIn), " tokens")+" in · "+strings.TrimSuffix(humanTokens(m.usageOut), " tokens")+" out")
		}
	}
	if m.priced {
		rows = append(rows, fmt.Sprintf("~$%.4f estimated", m.costUSD))
	}
	if line := m.contextLine(); line != "" {
		rows = append(rows, line)
		if bar := m.contextBar(20); bar != "" {
			rows = append(rows, bar)
		}
	}
	if m.a != nil {
		var tools []string
		if n := len(m.a.Config.MCP); n > 0 {
			tools = append(tools, fmt.Sprintf("%d MCP server(s)", n))
		}
		if n := len(m.a.Config.LSP); n > 0 {
			tools = append(tools, fmt.Sprintf("%d LSP server(s)", n))
		}
		if running := runningJobs(m.a); running > 0 {
			tools = append(tools, fmt.Sprintf("%d background job(s) · /jobs", running))
		}
		if len(tools) > 0 {
			rows = append(rows, "", "TOOLS")
			rows = append(rows, tools...)
		}
	}
	if len(m.todos) > 0 {
		done := 0
		for _, todo := range m.todos {
			if todo.Status == "completed" {
				done++
			}
		}
		rows = append(rows, "", fmt.Sprintf("TASKS  %d/%d", done, len(m.todos)))
		// Show the work that is still open: start just before the first
		// unfinished task so a long finished prefix cannot hide it.
		first := 0
		for i, todo := range m.todos {
			if todo.Status != "completed" {
				first = max(0, min(i-1, len(m.todos)-8))
				break
			}
		}
		if first > 0 {
			rows = append(rows, fmt.Sprintf("  ✓ %d earlier done", first))
		}
		for _, todo := range m.todos[first:min(len(m.todos), first+8)] {
			mark := "○ "
			switch todo.Status {
			case "completed":
				mark = "✓ "
			case "in_progress":
				mark = "◐ "
			}
			rows = append(rows, mark+todo.Content)
		}
		if rest := len(m.todos) - first - 8; rest > 0 {
			rows = append(rows, fmt.Sprintf("  +%d more · /todos", rest))
		}
	}
	if key := m.keyLabel("sidebar_toggle"); key != "" {
		rows = append(rows, "", key+" hides this")
	}
	return rows
}

// contextLine reports how full the context window was at the latest reply,
// like OpenCode's context meter. Empty until a provider reports usage.
func (m *model) contextLine() string {
	if m.a == nil || m.contextTokens <= 0 || m.a.Config.ContextWindow <= 0 {
		return ""
	}
	if m.contextFallback != "" {
		// The primary's window is not the denominator for a fallback's
		// usage, and the fallback's own window is not known here.
		return fmt.Sprintf("Context: %dk (fallback %s)", (m.contextTokens+999)/1000, Safe(m.contextFallback))
	}
	percent := m.contextTokens * 100 / m.a.Config.ContextWindow
	return fmt.Sprintf("Context: %d%% of %dk", min(percent, 100), m.a.Config.ContextWindow/1000)
}

// contextBar draws how full the context window is, width cells wide; empty
// when the window is unknown (no usage yet, or a fallback served the reply).
func (m *model) contextBar(width int) string {
	if m.a == nil || m.contextTokens <= 0 || m.a.Config.ContextWindow <= 0 || m.contextFallback != "" {
		return ""
	}
	filled := min(width, (m.contextTokens*width+m.a.Config.ContextWindow-1)/m.a.Config.ContextWindow)
	return strings.Repeat("▰", filled) + strings.Repeat("▱", width-filled)
}

// sidebarColor highlights section headings (all-caps rows).
func sidebarColor(row string) string {
	if row != "" && strings.ToUpper(row) == row && strings.ContainsAny(row, "ABCDEFGHIJKLMNOPQRSTUVWXYZ") {
		return accent
	}
	return muted
}

func statusColor(m *model) string {
	switch {
	case m.pending != nil || m.question != nil:
		return accent
	case m.active:
		return blue
	case m.status == "failed" || m.status == "cancelled" || m.lastRun.failedVerification():
		return red
	case m.lastRun != nil && m.lastRun.Outcome == "unverified" && strings.HasPrefix(m.status, "completed"):
		// Changes that no check confirmed are not a green result.
		return accent
	case m.status == "completed" || strings.HasPrefix(m.status, "completed ·"):
		return green
	}
	return muted
}

// anchorScroll keeps a scrolled-up reader on the same rows while output keeps
// arriving: the offset counts rows from the bottom, so it grows by however
// many rows the appended text added.
func (m *model) anchorScroll(text string, width int) {
	previous := m.anchorText
	m.anchorText = text
	if m.scrollOffset == 0 || previous == "" || width != m.anchorWidth {
		m.anchorWidth = width
		return
	}
	if len(text) <= len(previous) || !strings.HasPrefix(text, previous) {
		return
	}
	lineStart := strings.LastIndexByte(previous, '\n') + 1
	before := len(transcriptRows(previous[lineStart:], width, "", false))
	after := len(transcriptRows(text[lineStart:], width, "", false))
	if after > before {
		m.scrollOffset += after - before
	}
}

// keyRows wraps a line of key hints ("y once  ·  s session  ·  …") between
// its groups, so a key and its label never split across rows ("PgUp/P",
// "gDn"); a group wider than the row is wrapped on its own.
func keyRows(text, sep string, width int) []string {
	width = max(1, width)
	var rows []string
	line := ""
	for _, group := range strings.Split(text, sep) {
		switch {
		case line == "":
			line = group
		case ansi.StringWidth(line+sep+group) <= width:
			line += sep + group
		default:
			rows = append(rows, line)
			line = group
		}
		if ansi.StringWidth(line) > width {
			wrapped := terminalRows(line, width)
			rows = append(rows, wrapped[:len(wrapped)-1]...)
			line = wrapped[len(wrapped)-1]
		}
	}
	return append(rows, line)
}

// agentLine is the composer's last row: the agent in blue, the model and
// access muted ("Agent: build  ·  gpt-6.1-sol  ·  Access: Ask").
func agentLine(agent, model, access string, width int) string {
	text := "  Agent: " + agent + "  ·  " + model + "  ·  " + access
	if os.Getenv("NO_COLOR") != "" || ansi.StringWidth(text) > width {
		return cell(text, width, blue, panel)
	}
	return paint("  Agent: "+Safe(agent), blue, panel) +
		cell("  ·  "+model+"  ·  "+access, width-ansi.StringWidth("  Agent: "+agent), muted, panel)
}

// approvalTitle names what is asked in words: "Edit note.txt", "Shell go
// test ./...". The file comes from the preview's "+++ " header.
func approvalTitle(tool, preview string) string {
	if rest, ok := strings.CutPrefix(tool, "child:"); ok {
		return "subagent: " + approvalTitle(rest, preview)
	}
	verb := toolVerb(tool)
	for _, line := range strings.SplitN(preview, "\n", 8) {
		if path, ok := strings.CutPrefix(line, "+++ "); ok && path != "/dev/null" {
			return verb + " " + oneLine(Safe(strings.TrimPrefix(path, "b/")), 60)
		}
	}
	return verb
}

// panelHints is keyHints drawn on the panel color.
func panelHints(text string, width int) string { return keyHintsOn(text, width, panel) }
