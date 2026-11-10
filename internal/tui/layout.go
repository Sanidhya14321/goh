package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

// Only these trusted theme sequences reach the terminal. User/model content is
// sanitized before styling, including text received inside code fences.
// The values come from the fixed theme table in theme.go.
var (
	background = "10;10;10"
	panel      = "24;24;24"
	foreground = "230;230;230"
	muted      = "132;132;132"
	accent     = "250;178;131"
	blue       = "92;156;245"
	green      = "127;216;143"
	red        = "224;108;117"
)

func paint(text, fg, bg string) string {
	if os.Getenv("NO_COLOR") != "" {
		return text
	}
	return "\x1b[38;2;" + fg + "m\x1b[48;2;" + bg + "m" + text + "\x1b[0m"
}

func cell(text string, width int, fg, bg string) string {
	text = strings.ReplaceAll(strings.ReplaceAll(Safe(text), "\t", "    "), "\n", " ")
	text = ansi.Truncate(text, max(0, width), "")
	return paint(text+strings.Repeat(" ", max(0, width-ansi.StringWidth(text))), fg, bg)
}

func canvas(rows []string, width, height int) tea.View {
	rows = rows[:min(len(rows), height)]
	for len(rows) < height {
		rows = append(rows, "")
	}
	for i, row := range rows {
		// Rows already contain our own styling. Padding is always a trusted
		// sequence and never reinterprets untrusted escape codes.
		row = ansi.Truncate(row, width, "")
		rows[i] = row + paint(strings.Repeat(" ", max(0, width-ansi.StringWidth(row))), foreground, background)
	}
	view := tea.NewView(strings.Join(rows, "\n"))
	view.AltScreen = true
	return view
}

func (m *model) View() tea.View {
	width, height := m.width, m.height
	if width < 1 {
		width = 80
	}
	if height < 1 {
		height = 24
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
		project, sessionID = m.a.Store.Project, m.a.Store.ID
	}
	if project == "" {
		project, _ = os.Getwd()
	}
	right := "goh"
	if version := Version; version != "" {
		right += " " + version
	}
	right = Safe(modelName) + "  ·  " + right + " "
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
	if m.input == "" {
		draft := "Ask anything about your project…"
		switch {
		case m.pending != nil:
			draft = "Waiting for your approval: y allow once · s allow for session · n deny"
		case m.question != nil:
			draft = "Answer the question above"
		case m.active:
			draft = "Working… type to queue a follow-up"
		}
		if m.blocked != "" {
			draft = "Connect a provider to start chatting"
		}
		draftRows = terminalRows("▏ "+draft, promptWidth-5)
	}
	draftRows = draftRows[max(0, len(draftRows)-3):]
	promptRows := m.fileCompletionRows(promptWidth, min(6, max(1, height-16)))
	if matches := m.slashMatches(); len(matches) > 0 && m.input != matches[0].command {
		for _, match := range matches[:min(3, len(matches))] {
			promptRows = append(promptRows, cell("  "+match.command+"  "+match.label, promptWidth, muted, panel))
		}
		promptRows = append(promptRows, cell("  Tab complete", promptWidth, accent, panel))
	}
	promptRows = append(promptRows, cell("▎", 1, blue, background)+cell("", promptWidth-1, foreground, panel))
	for _, row := range draftRows {
		promptRows = append(promptRows, cell("▎", 1, blue, background)+cell("  "+row, promptWidth-1, foreground, panel))
	}
	promptRows = append(promptRows,
		cell("▎", 1, blue, background)+cell("", promptWidth-1, foreground, panel),
		cell("▎", 1, blue, background)+cell("  Agent: "+agentName+"  ·  "+modelName, promptWidth-1, blue, panel))
	controls := "Enter send  Alt+Enter newline  "
	if key := m.keyLabel("command_palette"); key != "" {
		controls += key + " commands  "
	}
	controls += "Ctrl+D exit  Tab agents"
	if m.active {
		controls = m.runControls()
	}
	if m.blocked != "" {
		controls = "Ctrl+P provider commands  /provider setup  Ctrl+D exit"
	}
	promptRows = append(promptRows, cell("   "+controls, promptWidth, muted, background))

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
		rows = append(rows,
			cell("  GoHarness  /  "+filepath.Base(project), columns+4, foreground, panel),
			cell("  "+m.statusLine(), columns+4, statusColor(m), background),
			"")
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
		review = "Allow " + m.pending.request.Tool + "?   y once  ·  s " + sessionScopeLabel(m.pending.request) + "  ·  n / Esc deny  ·  PgUp/PgDn scroll"
	} else if m.question != nil {
		body, review = m.question.view(), "INTERACTIVE QUESTION · answers grant no permissions"
	} else if home {
		body = ""
		if m.blocked != "" {
			body = "Chat setup required. /provider configures the URL, model and key."
			reason := strings.Split(m.blocked, "\n")
			if len(reason) > 1 {
				body += "\n" + reason[1]
			} else {
				body += "\n" + m.blocked
			}
		}
	}
	if review != "" {
		reviewRows := terminalRows(review, columns)
		reviewRows = reviewRows[:min(len(reviewRows), max(1, contentHeight-1))]
		for _, row := range reviewRows {
			rows = append(rows, cell("  "+row, columns+4, accent, panel))
		}
		contentHeight = max(0, contentHeight-len(reviewRows))
	}
	lines := m.visibleRows(body, columns-2, contentHeight, review != "")
	needle := ""
	if m.find != nil && review == "" {
		needle = strings.ToLower(m.find.query)
	}
	for _, row := range lines {
		if needle != "" && strings.Contains(strings.ToLower(row.text), needle) {
			// The marker keeps matches visible when NO_COLOR disables colors.
			rows = append(rows, cell("▸ "+row.text, columns+4, background, accent))
			continue
		}
		rows = append(rows, transcriptCell(row, columns+4))
	}
	for count := len(lines); count < contentHeight; count++ {
		rows = append(rows, "")
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
			row := ansi.Truncate(rows[i], width-sideWidth-1, "")
			rows[i] = row + paint(strings.Repeat(" ", max(0, width-sideWidth-1-ansi.StringWidth(row))), foreground, background) +
				cell("│", 1, muted, background) + cell("  "+side, sideWidth, sidebarColor(side), panel)
		}
	}
	for len(rows) < height-1 {
		rows = append(rows, "")
	}
	rows = append(rows, footer)
	return canvas(rows, width, height)
}

// Reflow only the visible transcript tail. Cached rows are reused during typing
// and permission decisions. Scrolling expands the window into older content.
func (m *model) visibleRows(text string, width, height int, review bool) []transcriptRow {
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
		start = min(m.reviewOffset, max(0, len(lines)-height))
		end = min(len(lines), start+height)
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
	rows = append(rows, sessionID, "", "MODEL", modelName)
	if t := humanTokens(m.usage); t != "" {
		rows = append(rows, t)
	}
	if m.priced {
		rows = append(rows, fmt.Sprintf("~$%.4f estimated", m.costUSD))
	}
	if line := m.contextLine(); line != "" {
		rows = append(rows, line)
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
		for _, todo := range m.todos[:min(len(m.todos), 8)] {
			mark := "○ "
			switch todo.Status {
			case "completed":
				mark = "✓ "
			case "in_progress":
				mark = "◐ "
			}
			rows = append(rows, mark+todo.Content)
		}
		if len(m.todos) > 8 {
			rows = append(rows, fmt.Sprintf("  +%d more · /todos", len(m.todos)-8))
		}
	}
	rows = append(rows, "")
	if key := m.keyLabel("sidebar_toggle"); key != "" {
		rows = append(rows, key+" toggle sidebar")
	}
	rows = append(rows, "PgUp/PgDn history")
	if key := m.keyLabel("command_palette"); key != "" {
		rows = append(rows, key+" commands")
	}
	return rows
}

// contextLine reports how full the context window was at the latest reply,
// like OpenCode's context meter. Empty until a provider reports usage.
func (m *model) contextLine() string {
	if m.a == nil || m.contextTokens <= 0 || m.a.Config.ContextWindow <= 0 {
		return ""
	}
	percent := m.contextTokens * 100 / m.a.Config.ContextWindow
	return fmt.Sprintf("Context: %d%% of %dk", min(percent, 100), m.a.Config.ContextWindow/1000)
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
	case m.status == "failed" || m.status == "cancelled":
		return red
	case m.status == "completed":
		return green
	}
	return muted
}
