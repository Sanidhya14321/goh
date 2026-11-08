// Package tui renders the runtime's event stream; it owns no agent state.
package tui

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode"

	tea "charm.land/bubbletea/v2"
	"example.com/independent-harness/internal/core"
	"example.com/independent-harness/internal/provider"
	agent "example.com/independent-harness/internal/runtime"
	"github.com/charmbracelet/x/ansi"
)

type ChatNavigation struct {
	Action     string
	Session    string
	Project    string
	Config     *core.Config
	Backdrop   string
	Draft      string
	BeforeTurn int
	// RevertPrompt and RevertOccurrence select the turn for "revert"
	// (occurrence counts identical prompts from the latest, 0 = latest).
	RevertPrompt     string
	RevertOccurrence int
}

func (n *ChatNavigation) Error() string { return "chat navigation: " + n.Action }

type permissionMsg struct {
	request  core.Permission
	response chan core.Decision
}
type doneMsg struct {
	status string
	err    error
}
type model struct {
	a                    *agent.Agent
	events               chan tea.Msg
	input                string
	inputCursor          int
	promptHistory        []string
	historyPosition      int
	historyDraft         string
	log                  string
	question             *questionState
	pending              *permissionMsg
	active               bool
	cancel               context.CancelFunc
	reviewOffset         int
	durationMS           int64
	todos                []core.Todo
	usage                int
	width, height        int
	status               string
	closed               chan struct{}
	runDone              chan struct{}
	navigation           string
	revertPrompt         string
	revertOccurrence     int
	blocked              string
	scrollOffset         int
	lastTranscriptHeight int
	hideSidebar          bool
	palette              bool
	paletteQuery         string
	paletteIndex         int
	wrappedText          string
	wrappedWidth         int
	wrappedRows          []transcriptRow
	wrappedWindow        int
	wrappedReview        bool
	infoPanel            *statusPanel
	timeline             *timelinePanel
	fileCompletion       *fileCompletion
	fileIndex            *fileIndexMsg
	fileIndexStale       bool // a run finished since the snapshot; files may have changed
	fileRescanning       bool
	fork                 *optionPicker
	forkBefore           int
	leader               bool
	keys                 core.Keymap
	themePicker          *optionPicker
	copyPicker           *optionPicker
	timestamps           bool
	find                 *findState
	queue                []string // prompts typed during a run, sent in order
	export               *exportPanel
}

func Safe(s string) string {
	return strings.Map(func(r rune) rune {
		if (unicode.IsControl(r) && r != '\n' && r != '\t') || r == 0x7f || r == 0x202e || r == 0x202d {
			return '�'
		}
		return r
	}, s)
}
func (m *model) Init() tea.Cmd { return nil }
func (m *model) wait() tea.Cmd {
	return func() tea.Msg {
		select {
		case v := <-m.events:
			return v
		case <-m.closed:
			return nil
		}
	}
}
func (m *model) Update(msg tea.Msg) (next tea.Model, command tea.Cmd) {
	hadSuggestions := m.hasSlashSuggestions()
	hadFiles := m.fileCompletion != nil
	defer func() {
		m.refreshFileCompletion()
		// The composer changes height when its completion panel opens or
		// closes. Invalidate the terminal diff to retire the old panel.
		if hadSuggestions != m.hasSlashSuggestions() || hadFiles != (m.fileCompletion != nil) {
			command = tea.Batch(command, tea.ClearScreen)
		}
	}()
	switch v := msg.(type) {
	case exportDoneMsg:
		if m.export != nil {
			m.export.saving = false
			m.export.saved = v.err == nil
			if v.err != nil {
				m.export.problem = v.err.Error()
			}
		}
	case forkOptionsMsg:
		if m.fork == v.picker {
			m.fork.options = v.options
			m.fork.footer = "  Enter fork  Esc back · copies conversation only"
		}
	case fileIndexMsg:
		m.fileIndex = &v
		m.fileRescanning = false
		if m.fileCompletion != nil {
			m.fileCompletion.matches = fileMatches(v.files, m.fileCompletion.query)
		}
	case tea.WindowSizeMsg:
		m.width = v.Width
		m.height = v.Height
	case core.Event:
		switch v.Type {
		case "subagent_event":
			b, _ := json.Marshal(v.Data)
			var d struct {
				TaskID string     `json:"task_id"`
				Event  core.Event `json:"event"`
			}
			json.Unmarshal(b, &d)
			if d.Event.Type == "assistant_updated" {
				b, _ := json.Marshal(d.Event.Data)
				var delta struct {
					Text string `json:"text"`
				}
				json.Unmarshal(b, &delta)
				m.log += Safe(delta.Text)
			} else if d.Event.Type == "tool_started" || d.Event.Type == "tool_finished" || d.Event.Type == "error" {
				m.log += "\n[child " + Safe(d.TaskID) + " " + Safe(d.Event.Type) + "]\n"
			}
		case "assistant_updated":
			b, _ := json.Marshal(v.Data)
			var d struct {
				Text string `json:"text"`
			}
			json.Unmarshal(b, &d)
			m.log += Safe(d.Text)
		case "tool_started":
			b, _ := json.Marshal(v.Data)
			var tool struct {
				Name      string          `json:"name"`
				Arguments json.RawMessage `json:"arguments"`
			}
			json.Unmarshal(b, &tool)
			args := Safe(toolArgumentText(tool.Arguments))
			if len(args) > 256 {
				args = strings.ToValidUTF8(args[:256], "�") + "…"
			}
			m.log += "\n▸ " + Safe(tool.Name) + "  " + args + "\n"
		case "tool_finished":
			b, _ := json.Marshal(v.Data)
			var tool struct {
				Result struct {
					Error   bool   `json:"error"`
					Content string `json:"content"`
				} `json:"result"`
			}
			json.Unmarshal(b, &tool)
			content := toolResultText(tool.Result.Content)
			if len(content) > 2048 {
				content = strings.ToValidUTF8(content[:2048], "�") + "…"
			}
			mark := "✓ "
			if tool.Result.Error {
				mark = "[error] "
			}
			m.log += mark + Safe(content) + "\n"
		case "subagent_started", "subagent_finished", "tool_updated", "error", "context_compacted", "skill_diagnostics", "command_expanded":
			b, _ := json.Marshal(v.Data)
			if len(b) > 2048 {
				b = b[:2048]
			}
			m.log += "\n[" + v.Type + "] " + Safe(string(b)) + "\n"
		case "provider_failover":
			m.log += failoverLine(v.Data)
		case "integration_degraded":
			m.log += degradedLine(v.Data)
		case "todos_updated":
			b, _ := json.Marshal(v.Data)
			var d struct {
				Todos []core.Todo `json:"todos"`
			}
			json.Unmarshal(b, &d)
			m.todos = append([]core.Todo{}, d.Todos...)
		case "assistant_completed":
			b, _ := json.Marshal(v.Data)
			var d struct {
				Usage *core.Usage `json:"usage"`
			}
			json.Unmarshal(b, &d)
			if d.Usage != nil {
				n := d.Usage.Total
				if n == 0 {
					n = d.Usage.Prompt + d.Usage.Completion
				}
				m.usage += n
			}
		case "run_finished":
			m.fileIndexStale = true
			m.log += "\n"
			b, _ := json.Marshal(v.Data)
			var d struct {
				Status   string `json:"status"`
				Duration int64  `json:"duration_ms"`
			}
			json.Unmarshal(b, &d)
			m.status = d.Status
			m.durationMS = d.Duration
		}
		if len(m.log) > 128<<10 {
			m.log = m.log[len(m.log)-(128<<10):]
		}
		return m, m.wait()
	case editorDoneMsg:
		m.applyEditor(v)
		return m, nil
	case questionMsg:
		m.question = &questionState{message: v, selected: map[int]bool{}}
		m.reviewOffset = 0
		return m, m.wait()
	case permissionMsg:
		m.pending = &v
		m.reviewOffset = 0
		return m, m.wait()
	case queuedSubmitMsg:
		return m.submitQueued()
	case doneMsg:
		m.active = false
		m.pending = nil
		m.question = nil
		m.status = v.status
		if v.err != nil {
			m.log += "\n" + Safe(v.err.Error()) + "\n"
			var httpErr *provider.HTTPError
			if errors.Is(v.err, core.ErrChatGPTLogin) {
				m.status = "ChatGPT sign-in required"
			} else if errors.As(v.err, &httpErr) {
				switch httpErr.Kind {
				case "authentication":
					m.blocked = "Provider rejected the API key. Open /provider to check the key and URL."
					if m.a != nil && m.a.Config.EffectiveProtocol() == core.ChatGPTProtocol {
						m.blocked = "ChatGPT authorization was rejected. Run goh auth login to reauthorize, or choose another provider with /connect."
					}
					m.status = "Provider authentication failed"
				case "request":
					m.log += "Check the API URL, protocol, and model ID with /provider.\n"
				case "rate_limit":
					m.log += "Provider rate limit reached. Wait and retry, or change the model with /provider.\n"
				}
			}
		}
		return m, m.afterRun(v)
	case tea.PasteMsg:
		if m.export != nil {
			return m.exportUpdate(v)
		}
		if m.fork != nil {
			return m.forkUpdate(v)
		}
		if m.themePicker != nil {
			return m.themeUpdate(v)
		}
		if m.copyPicker != nil {
			return m.copyUpdate(v)
		}
		if m.timeline != nil {
			return m.timelineUpdate(v)
		}
		if m.infoPanel != nil {
			return m, nil
		}
		if m.question != nil && m.question.custom {
			if len(m.question.text)+len(v.Content) <= 4096 {
				m.question.text += Safe(v.Content)
			}
		} else if m.palette {
			m.paletteQuery += Safe(v.Content)
			m.paletteIndex = 0
		} else if !m.active {
			m.insertInput(Safe(v.Content))
		}
	case tea.KeyPressMsg:
		key := v.String()
		if key == "ctrl+l" {
			return m, tea.ClearScreen
		}
		if m.export != nil {
			return m.exportUpdate(v)
		}
		if m.fork != nil {
			return m.forkUpdate(v)
		}
		if m.themePicker != nil {
			return m.themeUpdate(v)
		}
		if m.copyPicker != nil {
			return m.copyUpdate(v)
		}
		if m.timeline != nil {
			return m.timelineUpdate(v)
		}
		if m.infoPanel != nil {
			return m.statusKey(key)
		}
		if m.find != nil && m.pending == nil && m.question == nil {
			return m.findKey(key)
		}
		if key == "ctrl+c" {
			if m.active {
				m.cancel()
				return m, nil
			}
			m.log += "\nNo active run. Ctrl+D exits.\n"
			return m, nil
		}
		if key == "ctrl+d" {
			if m.active {
				m.cancel()
				m.log += "\nCancelling; Ctrl+D exits after cleanup.\n"
				return m, nil
			}
			return m, tea.Quit
		}
		if m.question != nil {
			m.questionKey(v)
			return m, nil
		}
		if m.pending != nil {
			switch key {
			case "pgdown", "down":
				m.reviewOffset += 5
			case "pgup", "up":
				m.reviewOffset -= 5
				if m.reviewOffset < 0 {
					m.reviewOffset = 0
				}
			case "y":
				m.pending.response <- core.Decision{Allow: true}
				m.pending = nil
			case "s":
				m.pending.response <- core.Decision{Allow: true, Session: true}
				m.pending = nil
			case "n", "esc":
				m.pending.response <- core.Decision{}
				m.pending = nil
			}
			return m, nil
		}
		switch key {
		case "pgup":
			m.scrollOffset = min(len(m.log), m.scrollOffset+max(1, m.height/2))
			return m, nil
		case "pgdown":
			m.scrollOffset = max(0, m.scrollOffset-max(1, m.height/2))
			return m, nil
		}
		if m.active {
			return m.activeKey(v)
		}
		if m.palette {
			return m.paletteKey(v)
		}
		if m.keys.Direct == nil {
			var c *core.Config
			if m.a != nil {
				c = &m.a.Config
			}
			m.keys = keymapFor(c)
		}
		if m.leader {
			// Leader key, as in OpenCode: the next key selects an action.
			m.leader = false
			m.status = ""
			if action, ok := m.keys.Leader[key]; ok {
				return m.runKeyAction(action)
			}
			return m, nil
		}
		if action, ok := m.keys.Direct[key]; ok {
			return m.runKeyAction(action)
		}
		if m.fileCompletion != nil && m.fileCompletionKey(key) {
			return m, nil
		}
		if key == "esc" {
			m.palette = false
			return m, nil
		}
		if key == "f2" {
			m.navigation = "provider"
			return m, tea.Quit
		}
		switch key {
		case "enter":
			if strings.TrimSpace(m.input) == "" {
				return m, nil
			}
			prompt := m.input
			if strings.TrimSpace(prompt) == "/export" {
				m.input, m.inputCursor = "", 0
				m.openExport()
				return m, nil
			}
			if strings.TrimSpace(prompt) == "/fork" {
				m.input, m.inputCursor = "", 0
				return m, m.openFork()
			}
			if trimmed := strings.TrimSpace(prompt); trimmed == "/find" || strings.HasPrefix(trimmed, "/find ") {
				m.input, m.inputCursor = "", 0
				m.startFind(strings.TrimPrefix(trimmed, "/find"))
				return m, nil
			}
			if strings.TrimSpace(prompt) == "/timestamps" {
				m.input, m.inputCursor = "", 0
				m.toggleTimestamps()
				return m, nil
			}
			if strings.TrimSpace(prompt) == "/copy" {
				m.input, m.inputCursor = "", 0
				m.openCopyPicker()
				return m, nil
			}
			if strings.TrimSpace(prompt) == "/theme" {
				m.input, m.inputCursor = "", 0
				m.openThemePicker()
				return m, nil
			}
			if strings.TrimSpace(prompt) == "/editor" {
				m.input, m.inputCursor = "", 0
				return m, m.openEditor()
			}
			if p := strings.TrimSpace(prompt); p == "/undo" || p == "/redo" {
				m.input, m.inputCursor = "", 0
				return m, m.requestRewind(p[1:])
			}
			if strings.TrimSpace(prompt) == "/timeline" {
				m.input, m.inputCursor = "", 0
				m.openTimeline()
				return m, nil
			}
			if strings.TrimSpace(prompt) == "/status" {
				m.input, m.inputCursor = "", 0
				m.openStatus()
				return m, nil
			}
			if strings.TrimSpace(prompt) == "/sidebar" {
				m.input, m.inputCursor = "", 0
				m.hideSidebar = !m.hideSidebar
				return m, nil
			}
			if strings.TrimSpace(prompt) == "/help" || strings.TrimSpace(prompt) == "/palette" {
				m.input = ""
				m.palette, m.paletteQuery, m.paletteIndex = true, "", 0
				return m, nil
			}
			switch strings.TrimSpace(prompt) {
			case "/connect", "/provider", "/settings", "/workspace", "/sessions", "/models", "/agents", "/new":
				m.navigation = strings.TrimSpace(prompt)[1:]
				m.input, m.inputCursor = "", 0
				return m, tea.Quit
			}
			if m.blocked != "" {
				m.log = m.blocked
				return m, nil
			}
			if strings.TrimSpace(prompt) == "/commands" {
				m.input = ""
				m.log += "\nCommands: /provider, /models, /agents, /new, /settings, /workspace, /sessions, /compact, /todos, /skills, /commands\n"
				names := make([]string, 0, len(m.a.Config.Commands))
				for name := range m.a.Config.Commands {
					names = append(names, name)
				}
				sort.Strings(names)
				for _, name := range names {
					m.log += "/" + name + " — " + Safe(m.a.Config.Commands[name].Description) + "\n"
				}
				if len(m.log) > 128<<10 {
					m.log = m.log[len(m.log)-(128<<10):]
				}
				return m, nil
			}
			if strings.TrimSpace(prompt) == "/skills" {
				m.input = ""
				m.log += "\nAvailable local skills:\n"
				for _, info := range m.a.Skills.List() {
					m.log += Safe(info.Name) + " — " + Safe(info.Description) + "\n"
				}
				if len(m.log) > 128<<10 {
					m.log = m.log[len(m.log)-(128<<10):]
				}
				return m, nil
			}
			if strings.TrimSpace(prompt) == "/todos" || strings.TrimSpace(prompt) == "/tasks" {
				m.input = ""
				m.log += "\nSession tasks:\n" + Safe(agent.TodoDisplay(m.todos))
				if len(m.todos) == 0 {
					m.log += "(empty)\n"
				}
				if len(m.log) > 128<<10 {
					m.log = m.log[len(m.log)-(128<<10):]
				}
				return m, nil
			}
			m.rememberPrompt(prompt)
			m.input, m.inputCursor = "", 0
			m.scrollOffset = 0
			m.active = true
			m.status = "running"
			m.log += stampLine(time.Now()) + "\nYou: " + prompt + "\n"
			ctx, cancel := context.WithCancel(context.Background())
			m.cancel = cancel
			runDone := make(chan struct{})
			m.runDone = runDone
			go func(done chan struct{}) {
				defer close(done)
				var s string
				var e error
				if strings.TrimSpace(prompt) == "/compact" {
					s, e = m.a.Compact(ctx)
				} else {
					s, e = m.a.Run(ctx, prompt)
				}
				select {
				case m.events <- doneMsg{s, e}:
				case <-m.closed:
				}
			}(runDone)
			return m, m.wait()
		case "alt+enter", "shift+enter", "ctrl+j":
			m.insertInput("\n")
		case "left":
			m.inputCursor = max(0, min(m.inputCursor, len([]rune(m.input)))-1)
		case "right":
			m.inputCursor = min(len([]rune(m.input)), m.inputCursor+1)
		case "home", "ctrl+a":
			m.inputLineEnd(false)
		case "end", "ctrl+e":
			m.inputLineEnd(true)
		case "up":
			m.inputUpDown(false)
		case "down":
			m.inputUpDown(true)
		case "ctrl+u":
			m.input, m.inputCursor = "", 0
		case "ctrl+w":
			m.deleteWord()
		case "tab":
			if matches := m.slashMatches(); len(matches) > 0 {
				m.input = matches[0].command
				m.inputCursor = len([]rune(m.input))
			} else if m.input == "" && m.a != nil {
				m.navigation = "cycle_agent"
				return m, tea.Quit
			}
		case "shift+tab":
			if m.input == "" && m.a != nil {
				m.navigation = "previous_agent"
				return m, tea.Quit
			}
		case "backspace":
			m.deleteInput(true)
		case "delete":
			m.deleteInput(false)
		default:
			if v.Text != "" {
				m.insertInput(Safe(v.Text))
				if v.Text == "@" && m.fileCompletion == nil {
					return m, m.openFileCompletion()
				}
			}
		}
	}
	return m, nil
}
func terminalRows(text string, width int) []string {
	text = strings.ReplaceAll(Safe(text), "\t", "    ")
	rows := strings.Split(ansi.Hardwrap(text, width, true), "\n")
	for i := range rows {
		rows[i] = ansi.Truncate(rows[i], width, "")
	}
	return rows
}

func (m *model) runControls() string {
	if m.pending != nil || m.question != nil {
		return "Ctrl+C cancel  ·  Ctrl+D exits after cleanup"
	}
	controls := "Esc / Ctrl+C cancel  ·  Enter queues a prompt  ·  Ctrl+D exits after cleanup"
	if n := len(m.queue); n > 0 {
		controls = fmt.Sprintf("%d queued  ·  ", n) + controls
	}
	return controls
}

func (m *model) compactView() tea.View {
	width, height := m.width, m.height
	if width < 1 {
		width = 80
	}
	if height < 1 {
		height = 24
	}
	header := []string{"GoHarness • Chat", m.status}
	if m.a != nil {
		header = []string{
			fmt.Sprintf("GoHarness • %s • session %s", m.a.Config.Model, m.a.Store.ID),
			m.status,
			"Agent: " + m.a.Config.Agent + " • Mode: " + m.a.Config.AgentMode + " • Profile: " + m.a.Config.Profile,
			fmt.Sprintf("Reported tokens: %d • Last run: %d ms", m.usage, m.durationMS),
		}
	}
	body, review := m.log, ""
	if m.pending != nil {
		review = "APPROVAL: " + m.pending.request.Tool + " • PgUp/PgDn review • y once / s session / n or Esc deny"
		body = m.pending.request.Preview
	} else if m.question != nil {
		review = "INTERACTIVE QUESTION • answers grant no permissions • PgUp/PgDn review"
		body = m.question.view()
	} else if len(m.todos) > 0 {
		remaining := 0
		for _, todo := range m.todos {
			if todo.Status != "completed" {
				remaining++
			}
		}
		header = append(header, fmt.Sprintf("Tasks: %d remaining / %d total • /todos lists all", remaining, len(m.todos)))
		for _, todo := range m.todos[:min(4, len(m.todos))] {
			header = append(header, "["+todo.Status+"] "+todo.Content)
		}
	}
	if height <= 3 {
		title := header[0]
		if review != "" {
			title = review
		}
		rows := []string{ansi.Truncate(Safe(title), width, "…")}
		if height > 1 {
			rows = append(rows, ansi.Truncate("Ctrl+C cancel • Ctrl+D exit", width, "…"))
		}
		view := tea.NewView(strings.Join(rows, "\n"))
		view.AltScreen = true
		return view
	}

	controls := "Enter send • Alt+Enter newline • Ctrl+C cancel • Ctrl+D exit • /settings"
	if m.active {
		controls = m.runControls()
	} else if m.blocked != "" {
		controls = "Ctrl+P provider • /workspace • /settings • Ctrl+D exit"
	}
	footer := terminalRows(controls, width)
	if len(footer) > max(1, height/3) {
		footer = terminalRows("^C cancel • ^D exit", width)
		footer = footer[:min(len(footer), height-2)]
	}
	draft := m.input
	if draft == "" && m.blocked != "" {
		draft = "/provider or /settings"
	}
	if len(draft) > 4096 {
		draft = "… " + strings.ToValidUTF8(draft[len(draft)-4096:], "�")
	}
	input := terminalRows("> "+draft, width)
	if m.input != "" {
		input = m.inputRows(width)
	}
	draftRows := min(3, max(1, height-len(footer)-2))
	input = input[max(0, len(input)-draftRows):]
	available := height - len(footer) - len(input)
	files := m.fileCompletionRows(width, min(5, max(0, available-1)))
	available -= len(files)
	reviewRows := []string{}
	if review != "" {
		reviewRows = terminalRows(review, width)
		if len(reviewRows) >= available {
			reviewRows = []string{ansi.Truncate(Safe(review), width, "…")}
		}
	}
	header = header[:min(len(header), max(0, available-len(reviewRows)-2))]
	for i, row := range header {
		row = strings.ReplaceAll(Safe(row), "\t", "    ")
		header[i] = ansi.Truncate(row, width, "…")
	}
	rows := append(header, reviewRows...)
	visible := max(0, available-len(rows))
	lines := terminalRows(body, width)
	start := max(0, len(lines)-visible)
	if review != "" {
		start = min(m.reviewOffset, max(0, len(lines)-visible))
	}
	rows = append(rows, lines[start:min(start+visible, len(lines))]...)
	rows = append(rows, files...)
	rows = append(rows, input...)
	rows = append(rows, footer...)
	view := tea.NewView(strings.Join(rows, "\n"))
	view.AltScreen = true
	return view
}
func Run(a *agent.Agent, notice, draft string) error {
	applyTheme(a.Config.Theme)
	m := &model{a: a, events: make(chan tea.Msg, 128), width: 80, height: 24, status: "ready", closed: make(chan struct{}), todos: append([]core.Todo{}, a.Todos...), keys: keymapFor(&a.Config)}
	m.insertInput(Safe(draft))
	var promptTimes map[string][]time.Time
	if a.Store != nil {
		promptTimes = userPromptTimes(a.Store.Records())
	}
	for _, message := range a.History {
		switch message.Role {
		case "user":
			m.rememberPrompt(message.Content)
			m.log += stampLine(popTime(promptTimes, message.Content)) + "\nYou: " + Safe(message.Content) + "\n"
		case "assistant":
			m.log += Safe(message.Content)
			for _, call := range message.ToolCalls {
				m.log += "\n[tool] " + Safe(call.Function.Name) + "\n"
			}
		case "tool":
			text := historyToolText(message.Content)
			if len(text) > 2048 {
				text = text[:2048] + "…"
			}
			m.log += "\n" + Safe(text) + "\n"
		}
	}
	if len(m.log) > 128<<10 {
		m.log = m.log[len(m.log)-(128<<10):]
	}
	if notice != "" {
		m.log += "\n[notice] " + Safe(notice) + "\n"
		m.status = "Action needs attention"
	}
	oldEmit, oldApprove, oldAsk := a.Emit, a.Approve, a.Ask
	a.Emit = func(e core.Event) {
		select {
		case m.events <- e:
		case <-m.closed:
		}
	}
	a.Approve = func(ctx context.Context, p core.Permission) (core.Decision, error) {
		response := make(chan core.Decision, 1)
		select {
		case m.events <- permissionMsg{p, response}:
		case <-ctx.Done():
			return core.Decision{}, ctx.Err()
		case <-m.closed:
			return core.Decision{}, context.Canceled
		}
		select {
		case d := <-response:
			return d, nil
		case <-ctx.Done():
			return core.Decision{}, ctx.Err()
		case <-m.closed:
			return core.Decision{}, context.Canceled
		}
	}
	a.Ask = func(ctx context.Context, request core.QuestionRequest) (core.QuestionReply, error) {
		response := make(chan questionResponse, 1)
		select {
		case m.events <- questionMsg{request, response}:
		case <-ctx.Done():
			return core.QuestionReply{}, ctx.Err()
		case <-m.closed:
			return core.QuestionReply{}, context.Canceled
		}
		select {
		case result := <-response:
			return result.reply, result.err
		case <-ctx.Done():
			return core.QuestionReply{}, ctx.Err()
		case <-m.closed:
			return core.QuestionReply{}, context.Canceled
		}
	}
	_, e := tea.NewProgram(m).Run()
	close(m.closed)
	if m.cancel != nil {
		m.cancel()
	}
	if m.runDone != nil {
		select {
		case <-m.runDone:
		case <-time.After(12 * time.Second):
			return fmt.Errorf("terminal closed; runtime cleanup did not settle within twelve seconds")
		}
	}
	a.Emit = oldEmit
	a.Approve = oldApprove
	a.Ask = oldAsk
	if e == nil && m.navigation != "" {
		return &ChatNavigation{Action: m.navigation, Session: a.Store.ID, Project: a.Store.Project, Config: &a.Config, Backdrop: m.View().Content, Draft: m.input, BeforeTurn: m.forkBefore, RevertPrompt: m.revertPrompt, RevertOccurrence: m.revertOccurrence}
	}
	return e
}

// RunBlocked renders the chat surface before a usable agent can be created.
func RunBlocked(status, reason, draft string) error {
	m := &model{width: 80, height: 24, status: status, blocked: reason, log: reason, keys: keymapFor(nil)}
	m.insertInput(Safe(draft))
	_, err := tea.NewProgram(m).Run()
	if err == nil && m.navigation != "" {
		return &ChatNavigation{Action: m.navigation, Backdrop: m.View().Content, Draft: m.input}
	}
	return err
}

// failoverLine describes a routing change in plain language. The data carries
// profile names and an error class only, never provider bodies or keys.
func failoverLine(data any) string {
	b, _ := json.Marshal(data)
	var d struct {
		Purpose, From, To, Reason, State, Detail string
	}
	json.Unmarshal(b, &d)
	what := "this turn"
	if d.Purpose == "compaction" {
		what = "context compaction"
	}
	if d.State == "skipped" {
		return "\n[provider] skipped fallback " + Safe(d.To) + ": " + Safe(d.Detail) + "\n"
	}
	return "\n[provider] " + Safe(d.From) + " failed (" + Safe(d.Reason) + ") before responding; retrying " + what + " with fallback profile " + Safe(d.To) + "\n"
}

// degradedLine explains that an optional integration is off for this run while
// chat and built-in tools continue.
func degradedLine(data any) string {
	b, _ := json.Marshal(data)
	var d struct{ Kind, Name, Error string }
	json.Unmarshal(b, &d)
	label := map[string]string{"mcp": "MCP server", "lsp": "language server"}[d.Kind]
	if label == "" {
		label = "integration"
	}
	return "\n[integration] " + label + " " + Safe(d.Name) + " unavailable and stopped: " + Safe(d.Error) + "\nChat and built-in tools continue. Fix its config and start a new run to retry.\n"
}

// requestRewind hands /undo or /redo to the launcher, which owns session
// switching. File restoration happens there after the run has settled.
func (m *model) requestRewind(action string) tea.Cmd {
	if m.a == nil {
		m.status = "Configure a provider to " + action + " a turn"
		return nil
	}
	if m.status == "running" {
		m.status = "Wait for the run to finish, or press Esc to cancel, before /" + action
		return nil
	}
	m.navigation = action
	return tea.Quit
}

// runKeyAction performs a configurable chat action.
func (m *model) runKeyAction(action string) (tea.Model, tea.Cmd) {
	switch action {
	case "leader":
		m.leader = true
		m.status = m.keys.Hint
	case "command_palette":
		m.palette, m.paletteQuery, m.paletteIndex = true, "", 0
	case "sidebar_toggle":
		m.hideSidebar = !m.hideSidebar
	case "timeline":
		m.openTimeline()
	case "status":
		m.openStatus()
	case "editor_open":
		if m.status != "running" {
			return m, m.openEditor()
		}
	case "theme_list":
		m.openThemePicker()
	case "messages_first":
		m.scrollOffset = len(m.log) // clamped to the oldest rendered row
	case "messages_last":
		m.scrollOffset = 0
	case "messages_half_page_up":
		m.scrollOffset += max(1, m.height/2)
	case "messages_half_page_down":
		m.scrollOffset = max(0, m.scrollOffset-max(1, m.height/2))
	case "messages_previous", "messages_next":
		m.jumpUserTurn(action == "messages_previous")
	case "messages_copy":
		return m, m.copyLastReply()
	case "messages_undo", "messages_redo":
		return m, m.requestRewind(strings.TrimPrefix(action, "messages_"))
	}
	return m, nil
}

// transcriptHeight is the row count the transcript area used in the last
// render; jumps place a target turn at the top of that area.
func (m *model) transcriptHeight() int {
	if m.lastTranscriptHeight > 0 {
		return m.lastTranscriptHeight
	}
	return max(1, m.height-8)
}

// jumpUserTurn moves the view to the previous or next "You:" turn relative to
// the turn currently at the top. Older turns outside the wrapped window are
// reached by scrolling a page, which widens the window on the next render.
func (m *model) jumpUserTurn(previous bool) {
	rows := m.wrappedRows
	if len(rows) == 0 {
		return
	}
	h := m.transcriptHeight()
	top := max(0, len(rows)-m.scrollOffset-h)
	target := -1
	if previous {
		for i := top - 1; i >= 0; i-- {
			if strings.HasPrefix(strings.TrimSpace(rows[i].text), "You:") {
				target = i
				break
			}
		}
		if target < 0 {
			m.scrollOffset += h
			m.status = "Earlier turns: scrolled up a page"
			return
		}
	} else {
		for i := top + 1; i < len(rows); i++ {
			if strings.HasPrefix(strings.TrimSpace(rows[i].text), "You:") {
				target = i
				break
			}
		}
		if target < 0 {
			m.scrollOffset = 0
			return
		}
	}
	m.scrollOffset = max(0, len(rows)-target-h)
}

// copyLastReply copies the latest assistant message (Markdown as written by
// the model) through the terminal clipboard (OSC 52), which also works over SSH
// in terminals that allow it.
func (m *model) copyLastReply() tea.Cmd {
	if m.a == nil {
		return nil
	}
	for i := len(m.a.History) - 1; i >= 0; i-- {
		if msg := m.a.History[i]; msg.Role == "assistant" && strings.TrimSpace(msg.Content) != "" {
			m.status = fmt.Sprintf("Copied last reply (%d characters)", len([]rune(msg.Content)))
			return tea.SetClipboard(msg.Content)
		}
	}
	m.status = "No assistant reply to copy yet"
	return nil
}

// keymapFor resolves the configured bindings, falling back to defaults (the
// configuration was already validated when it loaded).
func keymapFor(c *core.Config) core.Keymap {
	var user map[string]string
	if c != nil {
		user = c.Keybinds
	}
	km, err := core.ResolveKeymap(user)
	if err != nil {
		km, _ = core.ResolveKeymap(nil)
	}
	return km
}

// keyLabel formats the first key bound to action for on-screen hints, such as
// "Ctrl+B" or "Ctrl+X B" for a leader binding. It returns "" when unbound.
func (m *model) keyLabel(action string) string {
	if m.keys.Direct == nil {
		var c *core.Config
		if m.a != nil {
			c = &m.a.Config
		}
		m.keys = keymapFor(c)
	}
	format := func(k string) string {
		parts := strings.Split(k, "+")
		for i, p := range parts {
			if len(p) > 0 {
				parts[i] = strings.ToUpper(p[:1]) + p[1:]
			}
		}
		return strings.Join(parts, "+")
	}
	var direct, leaderKey []string
	for k, a := range m.keys.Direct {
		if a == action {
			direct = append(direct, k)
		}
		if a == "leader" {
			leaderKey = append(leaderKey, k)
		}
	}
	if len(direct) > 0 {
		sort.Strings(direct)
		// Prefer the conventional ctrl+ form when several keys are bound.
		sort.SliceStable(direct, func(i, j int) bool { return len(direct[i]) < len(direct[j]) })
		best := direct[0]
		for _, k := range direct {
			if strings.HasPrefix(k, "ctrl+") && !strings.HasPrefix(best, "ctrl+") {
				best = k
			}
		}
		if action == "command_palette" {
			for _, k := range direct {
				if k == "ctrl+p" {
					best = k
				}
			}
		}
		return format(best)
	}
	for k, a := range m.keys.Leader {
		if a == action && len(leaderKey) > 0 {
			return format(leaderKey[0]) + " " + strings.ToUpper(k)
		}
	}
	return ""
}
