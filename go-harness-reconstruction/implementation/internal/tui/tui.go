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
	a               *agent.Agent
	events          chan tea.Msg
	input           string
	inputCursor     int
	promptHistory   []string
	historyPosition int
	historyDraft    string
	log             string
	question        *questionState
	pending         *permissionMsg
	active          bool
	cancel          context.CancelFunc
	reviewOffset    int
	durationMS      int64
	todos           []core.Todo
	usage           int
	width, height   int
	status          string
	closed          chan struct{}
	runDone         chan struct{}
	navigation      string
	blocked         string
	scrollOffset    int
	hideSidebar     bool
	palette         bool
	paletteQuery    string
	paletteIndex    int
	wrappedText     string
	wrappedWidth    int
	wrappedRows     []transcriptRow
	wrappedWindow   int
	wrappedReview   bool
	infoPanel       *statusPanel
	timeline        *timelinePanel
	fileCompletion  *fileCompletion
	fileIndex       *fileIndexMsg
	fork            *optionPicker
	forkBefore      int
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
	case forkOptionsMsg:
		if m.fork == v.picker {
			m.fork.options = v.options
			m.fork.footer = "  Enter fork  Esc back · copies conversation only"
		}
	case fileIndexMsg:
		m.fileIndex = &v
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
	case questionMsg:
		m.question = &questionState{message: v, selected: map[int]bool{}}
		m.reviewOffset = 0
		return m, m.wait()
	case permissionMsg:
		m.pending = &v
		m.reviewOffset = 0
		return m, m.wait()
	case doneMsg:
		m.active = false
		m.pending = nil
		m.question = nil
		m.status = v.status
		if v.err != nil {
			m.log += "\n" + Safe(v.err.Error()) + "\n"
			var httpErr *provider.HTTPError
			if errors.As(v.err, &httpErr) {
				switch httpErr.Kind {
				case "authentication":
					m.blocked = "Provider rejected the API key. Open /provider to check the key and URL."
					m.status = "Provider authentication failed"
				case "request":
					m.log += "Check the API URL, protocol, and model ID with /provider.\n"
				case "rate_limit":
					m.log += "Provider rate limit reached. Wait and retry, or change the model with /provider.\n"
				}
			}
		}
		return m, nil
	case tea.PasteMsg:
		if m.fork != nil {
			return m.forkUpdate(v)
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
		if m.fork != nil {
			return m.forkUpdate(v)
		}
		if m.timeline != nil {
			return m.timelineUpdate(v)
		}
		if m.infoPanel != nil {
			return m.statusKey(key)
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
			return m, nil
		}
		if m.palette {
			return m.paletteKey(v)
		}
		switch key {
		case "ctrl+p", "ctrl+k":
			m.palette, m.paletteQuery, m.paletteIndex = true, "", 0
			return m, nil
		case "ctrl+b":
			m.hideSidebar = !m.hideSidebar
			return m, nil
		case "ctrl+t":
			m.openTimeline()
			return m, nil
		case "ctrl+g":
			m.openStatus()
			return m, nil
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
			if strings.TrimSpace(prompt) == "/fork" {
				m.input, m.inputCursor = "", 0
				return m, m.openFork()
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
			case "/provider", "/settings", "/workspace", "/sessions", "/models", "/agents", "/new":
				m.navigation = strings.TrimSpace(prompt)[1:]
				m.input, m.inputCursor = "", 0
				return m, tea.Quit
			}
			if m.blocked != "" {
				m.input = ""
				m.inputCursor = 0
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
			m.log += "\nYou: " + prompt + "\n"
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
		review = "APPROVAL: " + m.pending.request.Tool + " • PgUp/PgDn review • y once / s session / n deny"
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
	if m.blocked != "" {
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
	m := &model{a: a, events: make(chan tea.Msg, 128), width: 80, height: 24, status: "ready", closed: make(chan struct{}), todos: append([]core.Todo{}, a.Todos...)}
	m.insertInput(Safe(draft))
	for _, message := range a.History {
		switch message.Role {
		case "user":
			m.rememberPrompt(message.Content)
			m.log += "\nYou: " + Safe(message.Content) + "\n"
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
		return &ChatNavigation{Action: m.navigation, Session: a.Store.ID, Project: a.Store.Project, Config: &a.Config, Backdrop: m.View().Content, Draft: m.input, BeforeTurn: m.forkBefore}
	}
	return e
}

// RunBlocked renders the chat surface before a usable agent can be created.
func RunBlocked(status, reason string) error {
	m := &model{width: 80, height: 24, status: status, blocked: reason, log: reason}
	_, err := tea.NewProgram(m).Run()
	if err == nil && m.navigation != "" {
		return &ChatNavigation{Action: m.navigation, Backdrop: m.View().Content}
	}
	return err
}
