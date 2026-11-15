package tui

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	tea "charm.land/bubbletea/v2"
)

// WorkspaceChooser configures the project folder browser.
type WorkspaceChooser struct {
	Start   string                  // folder shown first, usually where goh was started
	Recent  []string                // recently opened projects, newest first
	Check   func(path string) error // refuses folders that cannot be projects
	Trusted func(path string) bool  // already trusted: open without asking
	Kind    func(path string) string
}

// ChooseWorkspace shows where goh was started, recent projects and the
// folders there; the user opens one (trusting it the first time) or browses
// into another. It returns the chosen folder, or ok=false when cancelled.
func ChooseWorkspace(c WorkspaceChooser) (string, bool, error) {
	m := &workspacePicker{c: c, width: 80, height: 24}
	m.browse(c.Start)
	err := runProgram(m)
	return m.chosen, m.chosen != "", err
}

const (
	rowOpen = iota
	rowParent
	rowRecent
	rowFolder
	rowJump
)

type workspaceRow struct {
	kind                int
	path, label, detail string
}

type workspacePicker struct {
	c             WorkspaceChooser
	dir, query    string
	folders       []string // subfolder names of dir
	index         int
	notice        string
	confirm       string // a folder waiting for the trust answer
	confirmIndex  int
	chosen        string
	width, height int
}

// browse shows dir's subfolders. Hidden folders are listed only when the
// filter starts with a dot.
func (m *workspacePicker) browse(dir string) {
	if abs, err := filepath.Abs(dir); err == nil {
		dir = abs
	}
	m.dir, m.query, m.index, m.notice, m.folders = dir, "", 0, "", nil
	entries, err := os.ReadDir(dir)
	if err != nil {
		m.notice = "Can't list this folder: " + err.Error()
	}
	for _, e := range entries {
		if len(m.folders) >= 2000 {
			break
		}
		if e.IsDir() || (e.Type()&os.ModeSymlink != 0 && isDir(filepath.Join(dir, e.Name()))) {
			m.folders = append(m.folders, e.Name())
		}
	}
	sort.Slice(m.folders, func(i, j int) bool { return strings.ToLower(m.folders[i]) < strings.ToLower(m.folders[j]) })
	if m.c.Check != nil && m.c.Check(dir) != nil {
		m.index = m.firstSelectable()
	}
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func (m *workspacePicker) firstSelectable() int {
	rows := m.rows()
	for i, r := range rows {
		if r.kind != rowParent {
			return i
		}
	}
	return 0
}

func (m *workspacePicker) allowed(path string) bool {
	return m.c.Check == nil || m.c.Check(path) == nil
}

func (m *workspacePicker) kind(path string) string {
	if m.c.Kind == nil {
		return ""
	}
	return m.c.Kind(path)
}

// rows lists what can be selected: open this folder, its parent, recent
// projects and subfolders, narrowed by the filter. A filter that looks
// like a path offers to go there.
func (m *workspacePicker) rows() []workspaceRow {
	if target, ok := typedPath(m.query, m.dir); ok {
		if isDir(target) && !strings.HasSuffix(m.query, string(filepath.Separator)+".") {
			return []workspaceRow{{kind: rowJump, path: target, label: shortPath(target, 60), detail: m.kind(target)}}
		}
		// A partial name completes against the folders in its parent.
		var rows []workspaceRow
		parent, prefix := filepath.Dir(target), strings.ToLower(filepath.Base(target))
		entries, _ := os.ReadDir(parent)
		for _, e := range entries {
			path := filepath.Join(parent, e.Name())
			if strings.HasPrefix(strings.ToLower(e.Name()), prefix) && isDir(path) && len(rows) < 50 {
				rows = append(rows, workspaceRow{kind: rowJump, path: path, label: shortPath(path, 60), detail: m.kind(path)})
			}
		}
		return rows
	}
	var rows []workspaceRow
	if m.query == "" {
		if m.allowed(m.dir) {
			detail := m.kind(m.dir)
			if m.c.Trusted != nil && m.c.Trusted(m.dir) {
				detail = strings.TrimPrefix(detail+" · trusted", " · ")
			}
			rows = append(rows, workspaceRow{kind: rowOpen, path: m.dir, label: "Open " + shortPath(m.dir, 50), detail: detail})
		}
		if parent := filepath.Dir(m.dir); parent != m.dir {
			rows = append(rows, workspaceRow{kind: rowParent, path: parent, label: "‹ Parent folder", detail: shortPath(parent, 40)})
		}
	}
	var candidates []workspaceRow
	for _, path := range m.c.Recent {
		if path != m.dir && isDir(path) {
			candidates = append(candidates, workspaceRow{kind: rowRecent, path: path, label: filepath.Base(path), detail: shortPath(path, 40)})
		}
	}
	hidden := strings.HasPrefix(m.query, ".")
	for _, name := range m.folders {
		if strings.HasPrefix(name, ".") && !hidden {
			continue
		}
		path := filepath.Join(m.dir, name)
		candidates = append(candidates, workspaceRow{kind: rowFolder, path: path, label: name + string(filepath.Separator)})
	}
	if m.query == "" {
		return append(rows, candidates...)
	}
	texts := make([]string, len(candidates))
	for i, r := range candidates {
		texts[i] = r.label + " " + r.detail
	}
	for _, i := range rankFuzzy(m.query, texts) {
		rows = append(rows, candidates[i])
	}
	return rows
}

// typedPath reads a filter that starts like a path (/, ~, ./ or ../).
func typedPath(query, dir string) (string, bool) {
	switch {
	case query == "~" || strings.HasPrefix(query, "~/"):
		home, err := os.UserHomeDir()
		if err != nil {
			return "", false
		}
		return filepath.Join(home, query[1:]), true
	case filepath.IsAbs(query):
		return filepath.Clean(query), true
	case query == "." || query == ".." || strings.HasPrefix(query, "./") || strings.HasPrefix(query, "../"):
		return filepath.Join(dir, query), true
	}
	return "", false
}

func (m *workspacePicker) Init() tea.Cmd { return nil }

func (m *workspacePicker) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok, handled := wheelAsKey(msg); handled {
		if !ok {
			return m, nil
		}
		msg = key
	}
	switch v := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = v.Width, v.Height
	case tea.PasteMsg:
		if m.confirm == "" && len(m.query)+len(v.Content) <= 1024 {
			m.query += oneLineQuery(v.Content)
			m.index = 0
		}
	case tea.KeyPressMsg:
		if m.confirm != "" {
			return m.confirmKey(v)
		}
		return m.browseKey(v)
	}
	return m, nil
}

func (m *workspacePicker) confirmKey(v tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch v.String() {
	case "up", "down", "tab", "shift+tab":
		m.confirmIndex = 1 - m.confirmIndex
	case "y":
		m.confirmIndex = 0
		fallthrough
	case "enter":
		if m.confirmIndex == 0 {
			m.chosen, m.confirm = m.confirm, ""
			return m, tea.Quit
		}
		m.confirm = ""
	case "n", "esc":
		m.confirm = ""
	case "ctrl+c", "ctrl+d":
		return m, tea.Quit
	}
	return m, nil
}

func (m *workspacePicker) browseKey(v tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	rows := m.rows()
	current := func() (workspaceRow, bool) {
		if len(rows) == 0 {
			return workspaceRow{}, false
		}
		return rows[min(m.index, len(rows)-1)], true
	}
	m.notice = ""
	switch v.String() {
	case "ctrl+c", "ctrl+d":
		return m, tea.Quit
	case "esc":
		if m.query != "" {
			m.query, m.index = "", 0
			return m, nil
		}
		return m, tea.Quit
	case "up":
		m.index = max(0, m.index-1)
	case "down":
		m.index = min(max(0, len(rows)-1), m.index+1)
	case "pgup":
		m.index = max(0, m.index-10)
	case "pgdown":
		m.index = min(max(0, len(rows)-1), m.index+10)
	case "home":
		m.index = 0
	case "end":
		m.index = max(0, len(rows)-1)
	case "left":
		if m.query == "" {
			m.up()
		}
	case "right", "tab":
		if r, ok := current(); ok && r.kind != rowOpen {
			m.browse(r.path)
		}
	case "enter":
		r, ok := current()
		if !ok {
			return m, nil
		}
		switch r.kind {
		case rowParent:
			m.browse(r.path)
		default:
			return m.open(r.path)
		}
	case "ctrl+u":
		m.query, m.index = "", 0
	case "backspace":
		if m.query == "" {
			m.up()
			return m, nil
		}
		value := []rune(m.query)
		m.query, m.index = string(value[:len(value)-1]), 0
	default:
		if v.Text != "" && len(m.query)+len(v.Text) <= 1024 {
			m.query += Safe(v.Text)
			m.index = 0
		}
	}
	return m, nil
}

// up browses the parent folder with the folder just left selected.
func (m *workspacePicker) up() {
	parent := filepath.Dir(m.dir)
	if parent == m.dir {
		return
	}
	left := m.dir
	m.browse(parent)
	for i, r := range m.rows() {
		if r.path == left {
			m.index = i
		}
	}
}

// open chooses path: at once when it is trusted, else after the trust
// question. A folder that cannot be a project says why and stays.
func (m *workspacePicker) open(path string) (tea.Model, tea.Cmd) {
	if m.c.Check != nil {
		if err := m.c.Check(path); err != nil {
			m.notice = err.Error()
			return m, nil
		}
	}
	if m.c.Trusted != nil && m.c.Trusted(path) {
		m.chosen = path
		return m, tea.Quit
	}
	m.confirm, m.confirmIndex = path, 0
	return m, nil
}

func (m *workspacePicker) View() tea.View {
	width, height := m.width, m.height
	if width < 1 || height < 1 {
		return blankView()
	}
	boxWidth := min(96, max(1, width-4))
	pad := strings.Repeat(" ", max(0, (width-boxWidth)/2))
	line := func(text, fg string) string { return pad + cell(text, boxWidth, fg, panel) }
	var rows []string
	if m.confirm != "" {
		rows = append(rows, line("  GoHarness  /  Trust this folder?", foreground), "",
			line("  "+shortPath(m.confirm, boxWidth-6), accent))
		if kind := m.kind(m.confirm); kind != "" {
			rows = append(rows, line("  "+kind, muted))
		}
		rows = append(rows, "")
		for _, text := range wordWrap("goh will read the files in this folder and its subfolders, and can propose edits and run commands there. Edits and commands still ask for your approval unless you allow them. Only trust folders whose contents you trust.", boxWidth-4) {
			rows = append(rows, line("  "+text, foreground))
		}
		rows = append(rows, "")
		for i, choice := range []string{"Yes, trust and open", "No, choose another folder"} {
			if i == m.confirmIndex {
				rows = append(rows, pad+cell("› "+choice, boxWidth, onAccent, accent))
			} else {
				rows = append(rows, line("  "+choice, foreground))
			}
		}
		rows = append(rows, "", line("  "+fitHints("↑/↓ select  ·  Enter confirm  ·  y yes  ·  n or Esc back", boxWidth-2), muted))
		top := max(0, (height-len(rows))/3)
		return canvas(append(make([]string, top), rows...), width, height)
	}

	rows = append(rows, line("  GoHarness  /  Open a project", foreground),
		line("  In "+shortPath(m.dir, boxWidth-8), muted))
	if m.c.Check != nil && m.c.Check(m.dir) != nil && m.query == "" {
		rows = append(rows, line("  This folder can't be a project (goh's saved keys live in it). Pick a folder below.", muted))
	}
	rows = append(rows, line("  Search  "+m.query+"▏", accent), "")
	list := m.rows()
	labelWidth := 0
	for _, r := range list {
		labelWidth = max(labelWidth, min(40, len([]rune(r.label))))
	}
	// Lines with section headers; selectedLine keeps the selection visible.
	type listLine struct {
		text, fg, bg string
	}
	var lines []listLine
	selectedLine, section := 0, -1
	for i, r := range list {
		group := r.kind
		if group == rowOpen || group == rowParent || group == rowJump {
			group = rowOpen
		}
		if group != section {
			section = group
			switch group {
			case rowRecent:
				lines = append(lines, listLine{"  Recent projects", muted, panel})
			case rowFolder:
				lines = append(lines, listLine{"  Folders in " + shortPath(m.dir, 40), muted, panel})
			}
		}
		detail := r.detail
		if r.kind == rowFolder {
			detail = m.kind(r.path)
		}
		text := r.label
		if detail != "" {
			text += strings.Repeat(" ", max(1, labelWidth-len([]rune(r.label))+2)) + detail
		}
		if i == min(m.index, len(list)-1) {
			selectedLine = len(lines)
			lines = append(lines, listLine{"› " + text, onAccent, accent})
		} else {
			fg := foreground
			if r.kind == rowParent {
				fg = muted
			}
			lines = append(lines, listLine{"  " + text, fg, panel})
		}
	}
	if len(list) == 0 {
		msg := "  No folders match"
		if _, ok := typedPath(m.query, m.dir); ok {
			msg = "  No such folder"
		}
		lines = append(lines, listLine{msg, muted, panel})
	}
	var footer []string
	if m.notice != "" {
		for _, text := range wordWrap(m.notice, boxWidth-4) {
			footer = append(footer, line("  "+text, red))
		}
	}
	footer = append(footer, "", line("  "+fitHints("Enter open  ·  → or Tab look inside  ·  ← back  ·  type to filter, or a path like ~/code  ·  Esc quit", boxWidth-2), muted))
	visible := max(1, height-len(rows)-len(footer))
	start := max(0, min(selectedLine-visible/2, len(lines)-visible))
	for _, l := range lines[start:min(len(lines), start+visible)] {
		rows = append(rows, pad+cell(l.text, boxWidth, l.fg, l.bg))
	}
	return canvas(append(rows, footer...), width, height)
}
