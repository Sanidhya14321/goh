package tui

import (
	"fmt"
	"sort"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/Sanidhya14321/goh/internal/session"
)

type previewBatch struct {
	results map[string]session.PreviewInfo
	errors  map[string]string
	next    int
}

type chooser struct {
	sessions      []session.Info
	summaries     map[string]session.PreviewInfo
	problems      map[string]string
	dir, project  string
	current       string
	query         string
	allProjects   bool
	loading       bool
	index         int
	chosen        bool
	selected      string
	previewOffset int
	previewOnly   bool
	width, height int
	// Session management, as in OpenCode's session list (Ctrl+R / Ctrl+D).
	renaming      bool
	renameText    string
	confirmDelete string
	notice        string
}

func (m *chooser) loadBatch(start int) tea.Cmd {
	dir := m.dir
	items := append([]session.Info(nil), m.sessions[start:min(start+32, len(m.sessions))]...)
	return func() tea.Msg {
		batch := previewBatch{results: map[string]session.PreviewInfo{}, errors: map[string]string{}, next: start + len(items)}
		for _, info := range items {
			summary, err := session.Preview(dir, info.ID)
			if err != nil {
				batch.errors[info.ID] = err.Error()
			} else {
				if meta, metaErr := session.ReadMeta(dir, info.ID); metaErr == nil && meta.Title != "" {
					summary.Title = meta.Title // a user-chosen title wins
				}
				batch.results[info.ID] = summary
			}
		}
		return batch
	}
}

func (m *chooser) Init() tea.Cmd {
	if m.dir == "" || len(m.sessions) == 0 {
		return nil
	}
	return m.loadBatch(0)
}

func (m *chooser) matches() []session.Info {
	var matches []session.Info
	query := strings.ToLower(strings.TrimSpace(m.query))
	if query == "" {
		matches = append(matches, session.Info{})
	}
	for _, info := range m.sessions {
		if !m.allProjects && m.project != "" && info.Project != m.project && info.Damaged == "" {
			continue
		}
		title := m.summaries[info.ID].Title
		if query == "" || strings.Contains(strings.ToLower(title+" "+info.ID+" "+info.Project), query) {
			matches = append(matches, info)
		}
	}
	return matches
}

func (m *chooser) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok, handled := wheelAsKey(msg); handled {
		if !ok {
			return m, nil
		}
		msg = key
	}
	switch v := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = v.Width, v.Height
	case previewBatch:
		selected := ""
		old := m.matches()
		if m.index < len(old) {
			selected = old[m.index].ID
		}
		if m.summaries == nil {
			m.summaries, m.problems = map[string]session.PreviewInfo{}, map[string]string{}
		}
		for id, summary := range v.results {
			m.summaries[id] = summary
		}
		for id, problem := range v.errors {
			m.problems[id] = problem
		}
		matches := m.matches()
		m.index = min(m.index, max(0, len(matches)-1))
		for i, info := range matches {
			if info.ID == selected {
				m.index = i
				break
			}
		}
		m.loading = v.next < len(m.sessions)
		if m.loading {
			return m, m.loadBatch(v.next)
		}
	case tea.PasteMsg:
		if m.renaming {
			if len(m.renameText)+len(v.Content) <= 256 {
				m.renameText += oneLineQuery(v.Content)
			}
			return m, nil
		}
		if len(m.query)+len(v.Content) <= 256 {
			m.query += oneLineQuery(v.Content)
			m.index, m.previewOffset = 0, 0
		}
	case tea.KeyPressMsg:
		matches := m.matches()
		if m.renaming {
			return m, m.renameKey(v, matches)
		}
		if v.String() != "ctrl+d" {
			m.confirmDelete = ""
		}
		switch v.String() {
		case "ctrl+r":
			if info, ok := m.selectedSession(matches); ok {
				m.renaming, m.renameText, m.notice = true, m.summaries[info.ID].Title, ""
			}
			return m, nil
		case "ctrl+d":
			m.deleteKey(matches)
			return m, nil
		case "up":
			m.index, m.previewOffset = max(0, m.index-1), 0
		case "down":
			m.index, m.previewOffset = min(max(0, len(matches)-1), m.index+1), 0
		case "home":
			m.index, m.previewOffset = 0, 0
		case "end":
			m.index, m.previewOffset = max(0, len(matches)-1), 0
		case "pgup":
			m.previewOffset = max(0, m.previewOffset-5)
		case "pgdown":
			m.previewOffset += 5
		case "tab":
			m.previewOnly = !m.previewOnly
		case "ctrl+w":
			m.allProjects = !m.allProjects
			m.index, m.previewOffset = 0, 0
		case "ctrl+u":
			m.query, m.index, m.previewOffset = "", 0, 0
		case "backspace":
			value := []rune(m.query)
			if len(value) > 0 {
				m.query = string(value[:len(value)-1])
			}
			m.index, m.previewOffset = 0, 0
		case "enter":
			if len(matches) > 0 {
				m.selected, m.chosen = matches[min(m.index, len(matches)-1)].ID, true
				return m, tea.Quit
			}
		case "esc", "ctrl+c":
			return m, tea.Quit
		default:
			if v.Text != "" && len(m.query)+len(v.Text) <= 256 {
				m.query += Safe(v.Text)
				m.index, m.previewOffset = 0, 0
			}
		}
	}
	return m, nil
}

func (m *chooser) View() tea.View {
	width, height := m.width, m.height
	if width < 1 || height < 1 {
		return blankView()
	}
	previewWidth := 0
	if width >= 100 && height >= 18 && !m.previewOnly {
		previewWidth = min(48, width/3)
	}
	listWidth := width - previewWidth
	scope := "Current workspace"
	if m.allProjects || m.project == "" {
		scope = "All workspaces"
	}
	matches := m.matches()
	count := len(matches)
	if count > 0 && matches[0].ID == "" {
		count--
	}
	rows := []string{
		cell("  GoHarness  /  Sessions", width, foreground, panel),
		cell("  Search  "+m.query+" ▏", width, accent, panel),
		cell(fmt.Sprintf("  %s · %d sessions · Ctrl+W switch scope", scope, count), width, muted, background),
		"",
	}
	visible := max(1, height-6)
	start := max(0, m.index-visible+1)
	for i := start; i < min(len(matches), start+visible); i++ {
		info := matches[i]
		title := "Start a new session"
		if info.Damaged != "" {
			title = "⚠ damaged " + Safe(info.ID) + " · Enter to recover"
		} else if info.ID != "" {
			title = m.summaries[info.ID].Title
			if title == "" {
				title = info.ID
			}
			if info.ID == m.current {
				title += " ● current"
			}
		}
		fg, bg, mark := muted, background, "  "
		if i == m.index {
			fg, bg, mark = background, accent, "› "
		}
		rows = append(rows, cell(mark+title, listWidth, fg, bg))
	}
	if len(matches) == 0 {
		rows = append(rows, cell("  No matching sessions", listWidth, muted, background))
	}
	for len(rows) < height-2 {
		rows = append(rows, cell("", listWidth, foreground, background))
	}
	detailWidth := previewWidth
	if m.previewOnly {
		detailWidth = width
	}
	if detailWidth > 0 {
		preview := []string{"PREVIEW", "", "Choose a session to inspect its saved conversation."}
		if len(matches) > 0 {
			info := matches[min(m.index, len(matches)-1)]
			if info.ID != "" {
				summary, ok := m.summaries[info.ID]
				preview = []string{"PREVIEW", "", info.ID, info.Project}
				if ok {
					if summary.Limited {
						preview = append(preview, "Limited preview · first 64 KiB")
					}
					preview = append(preview, summary.Modified.Local().Format("02 Jan 15:04"), summary.Status, "", "YOU", summary.User, "", "ASSISTANT", summary.Assistant)
				} else if problem := m.problems[info.ID]; problem != "" {
					preview = append(preview, "", "Preview unavailable", problem)
				} else {
					preview = append(preview, "", "Loading preview…")
				}
			}
		}
		lines := terminalRows(strings.Join(preview, "\n"), max(1, detailWidth-3))
		start := min(m.previewOffset, max(0, len(lines)-(height-6)))
		for y := 4; y < len(rows); y++ {
			text := ""
			if at := start + y - 4; at < len(lines) {
				text = lines[at]
			}
			if m.previewOnly {
				rows[y] = cell("  "+text, width, muted, panel)
			} else {
				rows[y] = rows[y] + cell("│ "+text, previewWidth, muted, panel)
			}
		}
	}
	status := "↑/↓ select  Enter open  Ctrl+R rename  Ctrl+D delete  Esc back"
	if m.previewOnly {
		status = "Tab list  PgUp/PgDn scroll  Enter open  Esc back"
	} else if previewWidth == 0 {
		status = "Tab preview  Enter open  Esc back  ↑/↓ select"
	}
	if m.loading {
		status += " · Loading summaries…"
	}
	if m.renaming {
		status = "Rename: " + m.renameText + "▏  Enter save · Esc cancel · empty restores the default"
	} else if m.notice != "" {
		status = m.notice
	}
	rows = append(rows, cell(status, width, muted, panel))
	return canvas(rows, width, height)
}

func SelectSession(infos []session.Info, dir, current, project string) (string, bool, error) {
	items := append([]session.Info(nil), infos...)
	sort.Slice(items, func(i, j int) bool { return items[i].ID > items[j].ID })
	m := &chooser{sessions: items, dir: dir, current: current, project: project, loading: dir != "" && len(items) > 0}
	for i, info := range m.matches() {
		if info.ID == current && current != "" {
			m.index = i
			break
		}
	}
	_, err := tea.NewProgram(m).Run()
	return m.selected, m.chosen, err
}

func (m *chooser) selectedSession(matches []session.Info) (session.Info, bool) {
	if len(matches) == 0 {
		return session.Info{}, false
	}
	info := matches[min(m.index, len(matches)-1)]
	return info, info.ID != ""
}

func (m *chooser) renameKey(v tea.KeyPressMsg, matches []session.Info) tea.Cmd {
	switch v.String() {
	case "esc", "ctrl+c":
		m.renaming = false
	case "enter":
		m.renaming = false
		info, ok := m.selectedSession(matches)
		if !ok {
			return nil
		}
		if err := session.SetTitle(m.dir, info.ID, m.renameText); err != nil {
			m.notice = "Rename failed: " + err.Error()
			return nil
		}
		title := session.CleanTitle(m.renameText)
		summary := m.summaries[info.ID]
		if title == "" {
			m.notice = "Title cleared; reopen sessions to see the default"
		} else {
			summary.Title = title
			m.notice = "Renamed to " + title
		}
		if m.summaries == nil {
			m.summaries = map[string]session.PreviewInfo{}
		}
		m.summaries[info.ID] = summary
	case "backspace":
		if r := []rune(m.renameText); len(r) > 0 {
			m.renameText = string(r[:len(r)-1])
		}
	case "ctrl+u":
		m.renameText = ""
	default:
		if v.Text != "" && len([]rune(m.renameText)) < 120 {
			m.renameText += Safe(v.Text)
		}
	}
	return nil
}

// deleteKey moves the selected session to the trash on a second Ctrl+D.
func (m *chooser) deleteKey(matches []session.Info) {
	info, ok := m.selectedSession(matches)
	if !ok {
		return
	}
	if info.ID == m.current {
		m.notice = "The current session cannot be deleted; open another session first"
		return
	}
	if m.confirmDelete != info.ID {
		m.confirmDelete = info.ID
		m.notice = "Press Ctrl+D again to move this session to the trash (goh sessions restore " + info.ID + " brings it back)"
		return
	}
	m.confirmDelete = ""
	if err := session.Trash(m.dir, info.ID); err != nil {
		m.notice = "Delete failed: " + err.Error()
		return
	}
	for i, s := range m.sessions {
		if s.ID == info.ID {
			m.sessions = append(m.sessions[:i], m.sessions[i+1:]...)
			break
		}
	}
	m.index = min(m.index, max(0, len(m.matches())-1))
	m.notice = "Moved " + info.ID + " to the trash"
}
