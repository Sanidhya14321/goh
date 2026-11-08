package tui

import (
	"example.com/independent-harness/internal/ignore"
	"os"
	"path"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"
)

const (
	fileScanLimit  = 12000
	fileScanDepth  = 24
	fileMatchLimit = 10
)

type projectFile struct {
	path, folded, basename string
}

type fileIndexMsg struct {
	files                  []projectFile
	ready, limited, failed bool
	scanned                time.Time
}

// fileIndexTTL bounds how long a snapshot is reused before a background
// rescan. Matches from the old snapshot stay visible meanwhile.
const fileIndexTTL = 15 * time.Second

type fileCompletion struct {
	start, selected int
	query           string
	matches         []string
}

// Scan names in batches so a single huge directory cannot allocate an
// unbounded listing. No file contents are opened.
func scanProjectFiles(project string, closed <-chan struct{}) fileIndexMsg {
	result := fileIndexMsg{ready: true}
	root, err := os.OpenRoot(project)
	if err != nil {
		result.failed = true
		return result
	}
	defer root.Close()
	remaining := fileScanLimit
	rules := &ignore.Matcher{}
	var scan func(string, int)
	scan = func(dir string, depth int) {
		// Read this directory's .gitignore before filtering its entries, so
		// its rules apply to the subtree as in git.
		if g, err := root.Open(path.Join(dir, ".gitignore")); err == nil {
			if info, statErr := g.Stat(); statErr == nil && info.Mode().IsRegular() {
				rules.Add(dir, g)
			}
			g.Close()
		}
		f, err := root.Open(dir)
		if err != nil {
			if dir == "." {
				result.failed = true
			}
			return
		}
		defer f.Close()
		for remaining > 0 {
			select {
			case <-closed:
				return
			default:
			}
			entries, readErr := f.ReadDir(min(128, remaining))
			for _, entry := range entries {
				if remaining == 0 {
					break
				}
				remaining--
				name := entry.Name()
				if strings.HasPrefix(name, ".") || !utf8.ValidString(name) || Safe(name) != name ||
					strings.ContainsAny(name, "\t\n") {
					continue
				}
				relative := path.Join(dir, name)
				if rules.Ignored(relative, entry.IsDir()) {
					continue
				}
				if entry.IsDir() {
					switch name {
					case "node_modules", "vendor":
						continue
					}
					if depth < fileScanDepth {
						scan(relative, depth+1)
					} else {
						result.limited = true
					}
				} else if entry.Type().IsRegular() {
					folded := strings.ToLower(relative)
					result.files = append(result.files, projectFile{relative, folded, path.Base(folded)})
				}
			}
			if readErr != nil {
				return
			}
		}
	}
	scan(".", 0)
	result.limited = result.limited || remaining == 0
	slices.SortFunc(result.files, func(a, b projectFile) int { return strings.Compare(a.path, b.path) })
	result.scanned = time.Now()
	return result
}

func fileMatches(files []projectFile, query string) []string {
	query = strings.ToLower(query)
	var buckets [4][]string
	for _, file := range files {
		var rank int
		switch {
		case strings.HasPrefix(file.basename, query):
			rank = 0
		case strings.Contains(file.basename, query):
			rank = 1
		case strings.Contains(file.folded, query):
			rank = 2
		case subsequence(file.folded, query):
			rank = 3
		default:
			continue
		}
		if len(buckets[rank]) < fileMatchLimit {
			buckets[rank] = append(buckets[rank], file.path)
		}
		if len(buckets[0]) == fileMatchLimit {
			break
		}
	}
	var matches []string
	for _, bucket := range buckets {
		matches = append(matches, bucket[:min(len(bucket), fileMatchLimit-len(matches))]...)
	}
	return matches
}

func subsequence(text, query string) bool {
	for _, r := range text {
		if query == "" {
			return true
		}
		next, size := utf8.DecodeRuneInString(query)
		if r == next {
			query = query[size:]
		}
	}
	return query == ""
}

func (m *model) openFileCompletion() tea.Cmd {
	draft := []rune(m.input)
	start := m.inputCursor - 1
	if (start > 0 && !unicode.IsSpace(draft[start-1])) ||
		(m.inputCursor < len(draft) && !unicode.IsSpace(draft[m.inputCursor])) {
		return nil
	}
	m.fileCompletion = &fileCompletion{start: start}
	if m.fileIndex != nil {
		m.fileCompletion.matches = m.completionMatches(m.fileIndex.files, "")
		stale := m.fileIndex.ready && (m.fileIndexStale || time.Since(m.fileIndex.scanned) > fileIndexTTL)
		if !stale || m.fileRescanning {
			return nil
		}
		m.fileRescanning = true // keep showing the old snapshot until this lands
	} else {
		m.fileIndex = &fileIndexMsg{}
	}
	m.fileIndexStale = false
	project := ""
	if m.a != nil {
		project = m.a.Store.Project
	}
	if project == "" {
		project, _ = os.Getwd()
	}
	closed := m.closed
	return func() tea.Msg { return scanProjectFiles(project, closed) }
}

func (m *model) refreshFileCompletion() {
	picker := m.fileCompletion
	if picker == nil {
		return
	}
	if m.active || m.palette || m.timeline != nil || m.infoPanel != nil || m.fork != nil || m.export != nil || m.themePicker != nil || m.copyPicker != nil || m.jobsPicker != nil || m.changesPicker != nil {
		m.fileCompletion = nil
		return
	}
	draft := []rune(m.input)
	cursor := min(m.inputCursor, len(draft))
	if cursor <= picker.start || draft[picker.start] != '@' ||
		(cursor < len(draft) && !unicode.IsSpace(draft[cursor])) {
		m.fileCompletion = nil
		return
	}
	query := string(draft[picker.start+1 : cursor])
	if len(query) > 512 || strings.ContainsFunc(query, unicode.IsSpace) {
		m.fileCompletion = nil
		return
	}
	if query != picker.query {
		picker.query, picker.selected = query, 0
		picker.matches = m.completionMatches(m.fileIndex.files, query)
	}
}

func (m *model) fileCompletionKey(key string) bool {
	picker := m.fileCompletion
	switch key {
	case "esc":
		m.fileCompletion = nil
	case "up":
		picker.selected = max(0, picker.selected-1)
	case "down":
		picker.selected = min(max(0, len(picker.matches)-1), picker.selected+1)
	case "enter", "tab":
		if len(picker.matches) > 0 {
			chosen := picker.matches[picker.selected]
			if strings.ContainsFunc(chosen, unicode.IsSpace) || strings.ContainsAny(chosen, "\\\"") {
				chosen = strconv.Quote(chosen)
			}
			draft := []rune(m.input)
			suffix := string(draft[m.inputCursor:])
			reference := "@" + chosen
			if suffix == "" {
				reference += " "
			}
			m.input = string(draft[:picker.start]) + reference + suffix
			m.inputCursor = picker.start + utf8.RuneCountInString(reference)
			m.fileCompletion = nil
		}
	default:
		return false
	}
	return true
}

func (m *model) fileCompletionRows(width, limit int) []string {
	picker := m.fileCompletion
	if picker == nil || limit <= 0 {
		return nil
	}
	index := m.fileIndex
	title := "  Project files"
	if index.limited {
		title += " · partial index"
	}
	if len(picker.matches) == 0 {
		status := "No matching files"
		switch {
		case !index.ready:
			status = "Scanning project…"
		case index.failed:
			status = "Project files unavailable"
		}
		return []string{cell("  "+status+" · Esc dismiss", width, muted, panel)}
	}
	var rows []string
	visible := min(limit, len(picker.matches))
	if limit >= 3 {
		rows = append(rows, cell(title+" · ↑/↓ select · Tab/Enter insert · Esc dismiss", width, muted, panel))
		visible = min(limit-1, len(picker.matches))
	}
	start := max(0, picker.selected-visible+1)
	for i := start; i < min(len(picker.matches), start+visible); i++ {
		if i == picker.selected {
			rows = append(rows, cell("› "+picker.matches[i], width, background, accent))
		} else {
			rows = append(rows, cell("  "+picker.matches[i], width, foreground, panel))
		}
	}
	return rows
}

// completionMatches lists subagents whose names start with the query (so
// "@gen" offers "general" for delegation) ahead of project files.
func (m *model) completionMatches(files []projectFile, query string) []string {
	var agents []string
	if m.a != nil && m.a.Store != nil {
		names := []string{"explore", "general"}
		for name := range m.a.Config.Agents {
			if name != "explore" && name != "general" {
				names = append(names, name)
			}
		}
		sort.Strings(names)
		for _, name := range names {
			if strings.HasPrefix(name, strings.ToLower(query)) && len(agents) < 4 {
				agents = append(agents, Safe(name))
			}
		}
	}
	return append(agents, fileMatches(files, query)...)
}
