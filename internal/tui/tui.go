// Package tui renders the runtime's event stream; it owns no agent state.
package tui

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode"

	tea "charm.land/bubbletea/v2"
	"github.com/Sanidhya14321/goh/internal/commands"
	"github.com/Sanidhya14321/goh/internal/core"
	"github.com/Sanidhya14321/goh/internal/provider"
	agent "github.com/Sanidhya14321/goh/internal/runtime"
	"github.com/Sanidhya14321/goh/internal/session"
	"github.com/Sanidhya14321/goh/internal/tools"
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

// mcpPromptsMsg carries the result of /mcp.
type mcpPromptsMsg struct {
	list []core.MCPPrompt
	err  error
}

type doneMsg struct {
	status string
	err    error
}
type model struct {
	// reviewHidden is set by the view when a pending approval or question
	// has none of its content on screen; it cannot be answered unseen.
	reviewHidden bool
	// reviewHeight is how many rows the last render gave a review body.
	reviewHeight int
	// reviewSeen is how many rows of the review body, from the top, have
	// been on screen; reviewTotal is how many it has.
	reviewSeen, reviewTotal int
	// stashedDraft is a draft set aside while a palette command uses the
	// composer; it returns when that command has run.
	stashedDraft string
	// partialReply is the assistant text streamed since the last other
	// event, removed from the transcript if the stream is retried.
	partialReply string
	partialStart int // offset in log where partialReply begins
	// childPartial is each subagent's streamed text since its last other
	// event, as the pieces appended to the log (parallel children
	// interleave), so a child's retry retracts only its own text.
	childPartial map[string][]logSpan
	// logTrimmed counts bytes cut from the front of log, so a span's
	// absolute position stays valid after trimming.
	logTrimmed int
	// inputWidth is the composer's text width at the last render, so Up and
	// Down move by displayed rows.
	inputWidth int
	// servedModel is the model that answered the latest turn, which differs
	// from the configured one after a fallback.
	servedModel     string
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
	runStarted      time.Time
	lastInterrupt   time.Time
	spinFrame       int
	// act is what the run is doing now, for the status line.
	act                  activity
	ticking              bool
	todos                []core.Todo
	usage                int
	width, height        int
	status               string
	closed               chan struct{}
	runDone              chan struct{}
	navigation           string
	revertPrompt         string
	revertOccurrence     int
	blocked              string // why chat cannot run yet
	fix                  string // the navigation that resolves blocked; see setupStep
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
	previewText          string // approval preview or question, wrapped
	previewWidth         int
	previewRows          []transcriptRow
	infoPanel            *statusPanel
	timeline             *timelinePanel
	fileCompletion       *fileCompletion
	fileIndex            *fileIndexMsg
	jobStopConfirm       string
	slashPrefix          string
	anchorText           string
	anchorWidth          int
	fileIndexStale       bool // a run finished since the snapshot; files may have changed
	fileRescanning       bool
	fork                 *optionPicker
	forkBefore           int
	leader               bool
	keys                 core.Keymap
	themePicker          *optionPicker
	copyPicker           *optionPicker
	jobsPicker           *optionPicker
	orphans              map[string]tools.OrphanJob
	changesPicker        *optionPicker
	timestamps           bool
	contextTokens        int
	contextFallback      string // fallback profile that served the latest usage; its window is unknown here
	details              detailsMode
	hideThinking         bool
	sessionTitle         string
	skillNoticeShown     bool
	costUSD              float64
	priced               bool
	reasoning            strings.Builder
	find                 *findState
	queue                []string // prompts typed during a run, sent in order
	// queueHeld explains why queued prompts were not sent after the last
	// run; they stay queued, in order, until the user sends or edits them.
	queueHeld string
	// lastRun is the settled result of the latest run (nil while running).
	lastRun *runResult
	export  *exportPanel
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
		// A live spinner and elapsed time while a run is active.
		if m.active && m.runStarted.IsZero() {
			m.runStarted = time.Now()
		} else if !m.active {
			m.runStarted = time.Time{}
		}
		if m.active && !m.ticking {
			m.ticking = true
			command = tea.Batch(command, spinTick())
		}
		// The composer changes height when its completion panel opens or
		// closes. Invalidate the terminal diff to retire the old panel.
		if hadSuggestions != m.hasSlashSuggestions() || hadFiles != (m.fileCompletion != nil) {
			command = tea.Batch(command, tea.ClearScreen)
		}
	}()
	if wheel, ok := msg.(tea.MouseWheelMsg); ok && !m.overlayOpen() && m.pending == nil && m.question == nil {
		// The wheel scrolls the conversation, three rows per notch.
		switch wheel.Button {
		case tea.MouseWheelUp:
			m.scrollOffset += 3
		case tea.MouseWheelDown:
			m.scrollOffset = max(0, m.scrollOffset-3)
		}
		return m, nil
	}
	if key, ok, handled := wheelAsKey(msg); handled {
		if !ok {
			return m, nil
		}
		msg = key
	}
	switch v := msg.(type) {
	case spinMsg:
		m.ticking = false
		if m.active {
			m.spinFrame++
		}
		return m, nil
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
			m.fileCompletion.matches = m.completionMatches(v.files, m.fileCompletion.query)
			m.fileCompletion.selected = min(m.fileCompletion.selected, max(0, len(m.fileCompletion.matches)-1))
		}
	case tea.WindowSizeMsg:
		m.width = v.Width
		m.height = v.Height
	case core.Event:
		m.act.track(v)
		if v.Type == "tool_progress" {
			return m, m.wait() // shown in the status line only
		}
		if v.Type == "reasoning_updated" {
			b, _ := json.Marshal(v.Data)
			var d struct {
				Text string `json:"text"`
			}
			json.Unmarshal(b, &d)
			m.addReasoning(d.Text)
			return m, m.wait()
		}
		m.flushReasoning()
		if (strings.HasPrefix(m.status, "Provider busy") || strings.HasPrefix(m.status, "Rate limited")) && (v.Type == "assistant_updated" || v.Type == "tool_started" || v.Type == "assistant_completed") {
			m.status = "running" // the retry got through
		}
		if v.Type == "provider_retry" {
			if n, ok := v.Data.(core.RetryNotice); ok {
				m.status = retryStatus(n)
			}
		}
		if v.Type == "session_titled" {
			if d, ok := v.Data.(map[string]any); ok {
				if t, ok := d["title"].(string); ok {
					m.sessionTitle = Safe(t)
				}
			}
		}
		if v.Type != "assistant_updated" && v.Type != "stream_retry" && v.Type != "assistant_reset" {
			// The reply text so far is final; a code block it left open
			// ends with it.
			if m.partialReply != "" {
				m.log += closeFence(m.partialReply)
			}
			m.partialReply = ""
		}
		switch v.Type {
		case "subagent_event":
			b, _ := json.Marshal(v.Data)
			var d struct {
				TaskID string     `json:"task_id"`
				Event  core.Event `json:"event"`
			}
			json.Unmarshal(b, &d)
			switch d.Event.Type {
			case "assistant_updated":
				b, _ := json.Marshal(d.Event.Data)
				var delta struct {
					Text string `json:"text"`
				}
				json.Unmarshal(b, &delta)
				text := Safe(delta.Text)
				if m.childPartial == nil {
					m.childPartial = map[string][]logSpan{}
				}
				m.childPartial[d.TaskID] = append(m.childPartial[d.TaskID], logSpan{at: m.logTrimmed + len(m.log), text: text})
				m.log += text
			case "stream_retry", "assistant_reset":
				// The child's attempt was abandoned: retract only its text.
				m.dropChildPartial(d.TaskID)
				if d.Event.Type == "stream_retry" {
					m.log += "\n[subagent " + Safe(d.TaskID) + " retrying its reply]\n"
				}
			default:
				delete(m.childPartial, d.TaskID) // its streamed text is final
				if d.Event.Type == "tool_started" || d.Event.Type == "tool_finished" || d.Event.Type == "error" {
					m.log += childEventLine(d.Event)
				}
			}
		case "stream_retry":
			// Remove the abandoned partial reply; the retry streams it anew.
			m.dropPartialReply()
			m.log += "\n[" + strings.ToLower(StreamRetryText(v.Data)[:1]) + StreamRetryText(v.Data)[1:] + "]\n"
		case "assistant_reset":
			// The final reply differs from what streamed: show the final one.
			m.dropPartialReply()
		case "assistant_updated":
			b, _ := json.Marshal(v.Data)
			var d struct {
				Text string `json:"text"`
			}
			json.Unmarshal(b, &d)
			if m.partialReply == "" {
				// A reply after tool output starts its own paragraph so it is
				// not read as more of the output.
				if strings.HasSuffix(m.log, "\n") && !strings.HasSuffix(m.log, "\n\n") && lastLineIsToolOutput(m.log) {
					m.log += "\n"
				}
				m.partialStart = len(m.log)
			}
			m.log += Safe(d.Text)
			m.partialReply += Safe(d.Text)
		case "tool_started":
			b, _ := json.Marshal(v.Data)
			var tool struct {
				Name      string          `json:"name"`
				Arguments json.RawMessage `json:"arguments"`
			}
			json.Unmarshal(b, &tool)
			m.log += toolCallLine(tool.Name, tool.Arguments)
		case "tool_finished":
			b, _ := json.Marshal(v.Data)
			var tool struct {
				Result struct {
					Error   bool   `json:"error"`
					Content string `json:"content"`
				} `json:"result"`
			}
			json.Unmarshal(b, &tool)
			mark := "✓ "
			if tool.Result.Error {
				mark = "[error] "
			}
			m.log += toolResultLog(mark, tool.Result.Content)
		case "context_compaction_skipped":
			b, _ := json.Marshal(v.Data)
			var d struct {
				Manual bool   `json:"manual"`
				Reason string `json:"reason"`
			}
			json.Unmarshal(b, &d)
			if d.Manual {
				m.log += "\n[context_compaction_skipped] Nothing to compact: the conversation is already as short as compaction can make it.\n"
			}
		case "subagent_started", "subagent_finished", "tool_updated", "error", "context_compacted", "command_expanded", "images_attached":
			b, _ := json.Marshal(v.Data)
			if len(b) > 2048 {
				b = b[:2048]
			}
			m.log += "\n[" + v.Type + "] " + Safe(string(b)) + "\n"
		case "skill_diagnostics":
			// Readable once per session; details stay in /status.
			if !m.skillNoticeShown {
				m.skillNoticeShown = true
				n := 0
				if m.a != nil && m.a.Skills != nil {
					n = len(m.a.Skills.Diagnostics())
				}
				m.log += fmt.Sprintf("\n[notice] %d skill file(s) could not be loaded; see /status for the reasons.\n", n)
			}
		case "provider_failover":
			m.log += failoverLine(v.Data)
		case "integration_degraded":
			m.log += degradedLine(v.Data)
		case "request_cap_learned":
			m.log += "\n[notice] " + RequestCapText(v.Data) + "\n"
		case "todos_updated":
			b, _ := json.Marshal(v.Data)
			var d struct {
				Todos []core.Todo `json:"todos"`
			}
			json.Unmarshal(b, &d)
			m.todos = append([]core.Todo{}, d.Todos...)
			for i := range m.todos {
				// Model-written text: strip terminal controls before it is drawn.
				m.todos[i].Content = Safe(strings.Join(strings.Fields(m.todos[i].Content), " "))
			}
		case "assistant_completed":
			b, _ := json.Marshal(v.Data)
			var d struct {
				Usage    *core.Usage `json:"usage"`
				Model    string      `json:"model"`
				Fallback string      `json:"fallback"`
			}
			json.Unmarshal(b, &d)
			if d.Usage != nil {
				m.contextFallback = d.Fallback
			}
			if d.Model != "" {
				m.servedModel = d.Model // after a fallback, not the configured one
			}
			if d.Usage != nil {
				n := d.Usage.Total
				if n == 0 {
					n = d.Usage.Prompt + d.Usage.Completion
				}
				m.usage += n
				if m.a != nil {
					priced := m.a.Config.Model
					if d.Model != "" {
						priced = d.Model // usage is charged to the model that served it
					}
					if p, ok := m.a.Config.Prices[priced]; ok {
						m.costUSD += float64(d.Usage.Prompt)/1e6*p.Input + float64(d.Usage.Completion)/1e6*p.Output
						m.priced = true
					}
				}
				// The latest reply's prompt plus output approximates how full
				// the context window is now.
				m.contextTokens = d.Usage.Prompt + d.Usage.Completion
			}
		case "run_started":
			m.lastRun = nil
		case "run_finished":
			m.fileIndexStale = true
			m.log += "\n"
			b, _ := json.Marshal(v.Data)
			var d struct {
				Status   string `json:"status"`
				Duration int64  `json:"duration_ms"`
			}
			json.Unmarshal(b, &d)
			r := parseRunResult(v)
			m.lastRun = &r
			m.status = r.label()
			if r.failedVerification() {
				m.log += r.failureLine()
			}
			m.durationMS = d.Duration
		}
		if len(m.log) > 128<<10 {
			m.trimLog()
		}
		return m, m.wait()
	case editorDoneMsg:
		m.applyEditor(v)
		return m, nil
	case questionMsg:
		if len(v.request.Questions) == 0 {
			v.response <- questionResponse{reply: core.QuestionReply{}}
			return m, m.wait()
		}
		m.question = &questionState{message: v, selected: map[int]bool{}}
		m.reviewOffset, m.reviewSeen = 0, 0
		m.question.focusFreeText(m)
		return m, m.wait()
	case permissionMsg:
		m.pending = &v
		m.reviewOffset, m.reviewSeen = 0, 0
		return m, m.wait()
	case queuedSubmitMsg:
		return m.submitQueued()
	case doneMsg:
		m.active = false
		m.pending = nil
		m.question = nil
		if m.a != nil && m.a.RemoteRefresh != nil {
			m.a.RemoteRefresh() // on the event loop: the view reads the store
		}
		// The run's own terminal event already set the status; the
		// transport's completion only ends the wait unless it differs.
		if m.lastRun == nil || m.lastRun.Status != v.status {
			m.status = v.status
		}
		if v.err != nil {
			m.log += "\n" + Safe(v.err.Error()) + "\n"
			var httpErr *provider.HTTPError
			if errors.Is(v.err, core.ErrChatGPTLogin) {
				m.status = "ChatGPT sign-in required"
			} else if errors.As(v.err, &httpErr) {
				switch httpErr.Kind {
				case "authentication":
					m.blocked = "Provider rejected the API key. Open /providers and press k to replace it, or t to test the connection."
					if m.a != nil && m.a.Config.EffectiveProtocol() == core.ChatGPTProtocol {
						m.blocked = "ChatGPT authorization was rejected. Open /providers and press k on ChatGPT to sign in again, or switch to another provider."
					}
					m.status = "Provider authentication failed"
				case "request":
					m.log += "Check the URL, protocol and model in /providers (t tests the connection, e edits a custom provider).\n"
				case "rate_limit":
					m.log += "Provider rate limit reached. Wait and retry, or switch model or provider with /models or /providers.\n"
				}
			}
		}
		return m, m.afterRun(v)
	case mcpPromptsMsg:
		m.log += "\nMCP prompts (run as /mcp:<server>:<prompt> [name=value ...]):\n"
		for _, p := range v.list {
			m.log += "/mcp:" + Safe(p.Server) + ":" + Safe(p.Name) + " " + Safe(strings.Join(p.Arguments, " ")) + " — " + Safe(p.Description) + "\n"
		}
		if v.err != nil {
			m.log += "error: " + Safe(v.err.Error()) + "\n"
		} else if len(v.list) == 0 {
			m.log += "(the configured servers offer no prompts)\n"
		}
		if len(m.log) > 128<<10 {
			m.trimLog()
		}
		return m, m.wait()
	case jobStoppedMsg:
		m.status = v.id + " " + v.status
		return m, nil
	case tea.PasteMsg:
		// Windows (CRLF) and old Mac (CR) line endings paste as newlines,
		// not as a control character.
		v.Content = strings.ReplaceAll(strings.ReplaceAll(v.Content, "\r\n", "\n"), "\r", "\n")
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
		if m.jobsPicker != nil {
			return m.jobsUpdate(v)
		}
		if m.changesPicker != nil {
			return m.changesUpdate(v)
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
			if len(m.paletteQuery)+len(v.Content) <= 256 {
				m.paletteQuery += oneLineQuery(v.Content)
				m.paletteIndex = 0
			}
		} else if m.pending == nil && m.question == nil {
			// While a run is active the composer queues a follow-up, so a
			// paste must land there just like typed text.
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
		if m.jobsPicker != nil {
			return m.jobsUpdate(v)
		}
		if m.changesPicker != nil {
			return m.changesUpdate(v)
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
				m.cancelRun()
				return m, nil
			}
			// Conventional idle Ctrl+C: clear the draft first, then a second
			// press within two seconds exits.
			if m.input != "" {
				m.input, m.inputCursor = "", 0
				return m, nil
			}
			if !m.lastInterrupt.IsZero() && time.Since(m.lastInterrupt) < 2*time.Second {
				return m, tea.Quit
			}
			m.lastInterrupt = time.Now()
			return m, tea.Tick(2100*time.Millisecond, func(time.Time) tea.Msg { return spinMsg{} })
		}
		if key == "ctrl+d" {
			if m.active {
				m.cancelRun()
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
			page := max(1, m.reviewHeight)
			switch strings.ToLower(key) {
			case "pgdown":
				m.reviewOffset += page
			case "down":
				m.reviewOffset += min(5, page)
			case "pgup":
				m.reviewOffset = max(0, m.reviewOffset-page)
			case "up":
				m.reviewOffset = max(0, m.reviewOffset-min(5, page))
			case "y", "s", "a":
				if m.attached() && strings.ToLower(key) != "y" {
					// The server takes a once-only decision; its session
					// grants and access are not this terminal's to change.
					m.status = "Attached: the server accepts y (allow once) or n (deny)"
					return m, nil
				}
				level, covers := coveringLevel(m.pending.request)
				if strings.ToLower(key) == "a" && (!covers || m.accessLevel().Rank() >= level.Rank()) {
					return m, nil
				}
				if m.reviewHidden {
					// Approving needs the operation on screen.
					m.status = "Enlarge the terminal to see the operation before approving (n denies)"
					return m, nil
				}
				if commandPreview(m.pending.request.Preview) && m.reviewSeen < m.reviewTotal {
					// A command runs as written: every line of it must have
					// been on screen.
					m.status = "Scroll with PgDn through the whole command before approving (n denies)"
					return m, nil
				}
				if strings.ToLower(key) == "a" {
					// Raise this session's access so requests like this one
					// stop asking; this one is allowed now.
					m.setAccess(level)
				}
				m.pending.response <- core.Decision{Allow: true, Session: strings.ToLower(key) == "s"}
				m.pending = nil
			case "n", "esc":
				m.pending.response <- core.Decision{}
				m.pending = nil
			}
			return m, nil
		}
		if m.palette && !m.active {
			return m.paletteKey(v)
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
			m.scrollOffset = 0 // back to the latest output
			if m.stashedDraft != "" && strings.HasPrefix(strings.TrimSpace(m.input), "/") {
				// Cancel a command the palette started: the draft comes back.
				m.input = ""
				m.restoreStashedDraft()
			}
			return m, nil
		}
		if key == "f2" {
			m.navigation = "provider"
			return m, tea.Quit
		}
		switch key {
		case "enter":
			if strings.TrimSpace(m.input) == "" && len(m.queue) > 0 && m.blocked == "" {
				// Held follow-ups: Enter on an empty composer sends the next.
				m.queueHeld = ""
				return m.submitQueued()
			}
			if strings.TrimSpace(m.input) == "" {
				if m.blocked != "" {
					// Nothing can run yet: Enter goes to setup.
					m.navigation, _ = m.setupStep()
					return m, tea.Quit
				}
				return m, nil
			}
			prompt := m.input
			if m.stashedDraft != "" {
				// A command opened from the palette over a draft: put the
				// draft back once the composer is free again.
				defer m.restoreStashedDraft()
			}
			if strings.TrimSpace(prompt) == "/export" {
				m.input, m.inputCursor = "", 0
				m.openExport()
				return m, nil
			}
			if strings.TrimSpace(prompt) == "/fork" {
				m.input, m.inputCursor = "", 0
				return m, m.openFork()
			}
			if trimmed := strings.TrimSpace(prompt); trimmed == "/effort" || strings.HasPrefix(trimmed, "/effort ") {
				m.input, m.inputCursor = "", 0
				m.setEffort(strings.TrimPrefix(trimmed, "/effort"))
				return m, nil
			}
			if strings.TrimSpace(prompt) == "/thinking" {
				m.input, m.inputCursor = "", 0
				m.toggleThinking()
				return m, nil
			}
			if trimmed := strings.TrimSpace(prompt); trimmed == "/exit" || trimmed == "/quit" {
				m.input, m.inputCursor = "", 0
				if m.active {
					// Same rule as Ctrl+D: never leave a run mid-effect.
					m.cancelRun()
					m.log += "\nCancelling; Ctrl+D exits after cleanup.\n"
					return m, nil
				}
				return m, tea.Quit
			}
			if trimmed := strings.TrimSpace(prompt); trimmed == "/rename" || strings.HasPrefix(trimmed, "/rename ") {
				m.input, m.inputCursor = "", 0
				m.renameSession(strings.TrimPrefix(trimmed, "/rename"))
				return m, nil
			}
			if trimmed := strings.TrimSpace(prompt); trimmed == "/find" || strings.HasPrefix(trimmed, "/find ") {
				m.input, m.inputCursor = "", 0
				m.startFind(strings.TrimPrefix(trimmed, "/find"))
				return m, nil
			}
			if strings.TrimSpace(prompt) == "/details" {
				m.input, m.inputCursor = "", 0
				m.toggleDetails()
				return m, nil
			}
			if strings.TrimSpace(prompt) == "/timestamps" {
				m.input, m.inputCursor = "", 0
				m.toggleTimestamps()
				return m, nil
			}
			if strings.TrimSpace(prompt) == "/changes" {
				m.input, m.inputCursor = "", 0
				m.openChanges()
				return m, nil
			}
			if strings.TrimSpace(prompt) == "/snapshots" {
				m.input, m.inputCursor = "", 0
				m.showSnapshots()
				return m, nil
			}
			if strings.TrimSpace(prompt) == "/jobs" {
				m.input, m.inputCursor = "", 0
				m.openJobs()
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
			case "/connect", "/provider", "/providers", "/settings", "/workspace", "/access", "/sessions", "/models", "/agents", "/new":
				m.navigation = strings.TrimSpace(prompt)[1:]
				if m.navigation == "providers" {
					m.navigation = "provider"
				}
				m.input, m.inputCursor = "", 0
				return m, tea.Quit
			}
			if m.blocked != "" {
				if !commands.IsCommand(prompt) {
					// A message typed before chat can run opens setup; the
					// draft comes back in the composer once it is done.
					m.navigation, _ = m.setupStep()
					return m, tea.Quit
				}
				_, step := m.setupStep()
				m.log = m.blocked + "\n\n" + strings.Fields(prompt)[0] + " can't run yet. Press Enter to " + step + "."
				return m, nil
			}
			if strings.TrimSpace(prompt) == "/commands" {
				m.input = ""
				m.log += "\nCommands: /providers, /connect, /models, /agents, /access, /new, /settings, /workspace, /sessions, /compact, /todos, /skills, /commands, /mcp\n"
				// The same set the palette shows: configured and project commands.
				set := m.a.CommandSet()
				names := make([]string, 0, len(set))
				for name := range set {
					names = append(names, name)
				}
				sort.Strings(names)
				for _, name := range names {
					m.log += "/" + name + " — " + Safe(set[name].Description) + "\n"
				}
				if len(m.log) > 128<<10 {
					m.trimLog()
				}
				return m, nil
			}
			if strings.TrimSpace(prompt) == "/mcp" {
				// Listing starts the configured servers (with approval, as a
				// run would), so it runs in the background like a prompt.
				m.input, m.inputCursor = "", 0
				if len(m.a.Config.MCP) == 0 {
					m.log += "\nMCP prompts: none (no MCP servers are configured)\n"
					return m, nil
				}
				m.active, m.status = true, "Listing MCP prompts"
				ctx, cancel := context.WithCancel(context.Background())
				m.cancel = cancel
				runDone := make(chan struct{})
				m.runDone = runDone
				go func(done chan struct{}) {
					defer close(done)
					list, err := m.a.ListMCPPrompts(ctx)
					for _, msg := range []tea.Msg{mcpPromptsMsg{list, err}, doneMsg{"completed", nil}} {
						select {
						case m.events <- msg:
						case <-m.closed:
							return
						}
					}
				}(runDone)
				return m, m.wait()
			}
			if strings.TrimSpace(prompt) == "/skills" {
				m.input = ""
				m.log += "\nAvailable local skills:\n"
				for _, info := range m.a.Skills.List() {
					m.log += Safe(info.Name) + " — " + Safe(info.Description) + "\n"
				}
				if len(m.log) > 128<<10 {
					m.trimLog()
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
					m.trimLog()
				}
				return m, nil
			}
			m.rememberPrompt(prompt)
			m.input, m.inputCursor = "", 0
			m.scrollOffset = 0
			m.active = true
			m.status = "running"
			m.log += stampLine(time.Now()) + "\nYou: " + prompt + "\n" + closeFence(prompt+"\n")
			ctx, cancel := context.WithCancel(context.Background())
			m.cancel = cancel
			runDone := make(chan struct{})
			m.runDone = runDone
			go func(done chan struct{}) {
				defer close(done)
				var s string
				var e error
				if trimmed := strings.TrimSpace(prompt); trimmed == "/compact" || strings.HasPrefix(trimmed, "/compact ") {
					s, e = m.a.CompactWith(ctx, strings.TrimPrefix(trimmed, "/compact"))
				} else if command, ok := strings.CutPrefix(strings.TrimSpace(prompt), "!"); ok {
					s, e = m.a.RunShell(ctx, command)
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
			m.moveCursor(false)
		case "right":
			m.moveCursor(true)
		case "home", "ctrl+a":
			m.inputLineEnd(false)
		case "end", "ctrl+e":
			m.inputLineEnd(true)
		case "up":
			if strings.TrimSpace(m.input) == "" && len(m.queue) > 0 {
				m.unqueueLast()
				break
			}
			m.inputUpDown(false)
		case "down":
			m.inputUpDown(true)
		case "ctrl+u":
			m.input, m.inputCursor = "", 0
		case "ctrl+w":
			m.deleteWord()
		case "tab":
			if m.completeSlash() {
			} else if m.input == "" && m.a != nil {
				m.navigation = "cycle_agent"
				return m, tea.Quit
			}
		case "shift+tab":
			if m.input == "" && m.a != nil {
				m.cycleAccess()
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

// screenHistoryBytes bounds how much journaled conversation a reopened chat
// renders; the transcript keeps 128 KiB of it (see trimLog).
const screenHistoryBytes = 256 << 10

// screenHistory is what a reopened chat shows: the journaled messages, not
// the model's context, which compaction replaces with a summary. Only the
// recent end that fits the transcript is returned.
func screenHistory(a *agent.Agent) []core.Message {
	if a.Store == nil {
		return a.History
	}
	var messages []core.Message
	for _, record := range a.Store.Records() {
		if record.Type != "message" {
			continue
		}
		var message core.Message
		if json.Unmarshal(record.Data, &message) == nil {
			messages = append(messages, message)
		}
	}
	if len(messages) == 0 {
		return a.History
	}
	size := 0
	for i := len(messages) - 1; i >= 0; i-- {
		size += min(len(messages[i].Content), 4096) + 64
		for _, call := range messages[i].ToolCalls {
			size += min(len(call.Function.Arguments), 256) + 64
		}
		if size > screenHistoryBytes {
			return messages[i+1:]
		}
	}
	return messages
}

// loadHistory writes a resumed conversation the way a live turn writes it.
func (m *model) loadHistory(history []core.Message, promptTimes map[string][]time.Time) {
	for _, message := range history {
		switch message.Role {
		case "user":
			if message.Automatic {
				m.log += "\n[GoHarness] " + Safe(message.Content) + "\n"
				continue
			}
			m.rememberPrompt(message.Content)
			m.log += stampLine(popTime(promptTimes, message.Content)) + "\nYou: " + Safe(message.Content) + "\n" + closeFence(Safe(message.Content)+"\n")
		case "assistant":
			if message.Content != "" && lastLineIsToolOutput(m.log) && !strings.HasSuffix(m.log, "\n\n") {
				m.log += "\n"
			}
			m.log += Safe(message.Content) + closeFence(Safe(message.Content))
			for _, call := range message.ToolCalls {
				m.log += toolCallLine(call.Function.Name, json.RawMessage(call.Function.Arguments))
			}
		case "tool":
			m.log += toolResultLog(historyToolParts(message.Content))
		}
	}
}

// retryStatus explains a provider retry. A rate limit (HTTP 429) is the
// provider holding back this account's requests, not the provider being
// busy, and its waits are long; saying so tells the user where the time goes.
func retryStatus(n core.RetryNotice) string {
	wait := float64(n.DelayMS) / 1000
	if strings.Contains(n.Reason, "HTTP 429") {
		return fmt.Sprintf("Rate limited by the provider (requests or tokens per minute); waiting %.0fs, then attempt %d of %d", wait, n.Attempt, n.Max)
	}
	return fmt.Sprintf("Provider busy (%s); retrying in %.1fs, attempt %d of %d", Safe(n.Reason), wait, n.Attempt, n.Max)
}

// toolCallLine is the "▸ name  arguments" line above a tool's result.
func toolCallLine(name string, arguments json.RawMessage) string {
	args := Safe(toolArgumentText(arguments))
	if len(args) > 256 {
		args = strings.ToValidUTF8(args[:256], "�") + "…"
	}
	return "\n▸ " + Safe(name) + "  " + args + "\n"
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
	controls := "Esc / Ctrl+C cancel  ·  Enter queues a prompt  ·  Shift+Tab access  ·  Ctrl+D exits after cleanup"
	if n := len(m.queue); n > 0 {
		controls = fmt.Sprintf("%d queued (↑ edit last)  ·  ", n) + controls
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
	header := []string{"GoHarness", m.statusLine()}
	if m.a != nil {
		header[0] = "GoHarness  ·  " + m.a.Config.Agent + "  ·  " + m.a.Config.Model
		// Where it acts and with how much access come first at any width;
		// the model and agent drop off before them.
		scope := "local"
		if m.attached() {
			scope = "attached"
		}
		header = append([]string{filepath.Base(m.a.Store.Project) + " · " + scope + " · " + m.accessLabel()}, header...)
		header[1] = m.a.Config.Agent + "  ·  " + m.a.Config.Model
	}
	// The same transcript as the wide layout: hidden reasoning and details
	// stay hidden, and scrolling, /find and turn jumps work.
	body, review := m.displayLog(), ""
	if m.pending != nil {
		review = "Allow " + m.pending.request.Tool + "?  " + m.approvalChoices(" · ") + " · PgUp/PgDn scroll"
		body = relativizePreview(m.pending.request.Preview, m.a.Store.Project)
	} else if m.question != nil {
		review = "INTERACTIVE QUESTION · answers grant no permissions · PgUp/PgDn scroll"
		body = m.question.view()
	} else if len(m.todos) > 0 {
		remaining := 0
		for _, todo := range m.todos {
			if todo.Status != "completed" {
				remaining++
			}
		}
		header = append(header, fmt.Sprintf("Tasks: %d remaining / %d total · /todos lists all", remaining, len(m.todos)))
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
			rows = append(rows, ansi.Truncate("Ctrl+C cancel · Ctrl+D exit", width, "…"))
		}
		// Nothing of a pending operation fits: it cannot be approved unseen.
		m.reviewHidden = review != ""
		m.reviewHeight = 0
		view := tea.NewView(strings.Join(rows, "\n"))
		view.AltScreen = true
		view.MouseMode = mouseMode()
		return view
	}

	palette := "/ commands"
	if key := m.keyLabel("command_palette"); key != "" {
		palette = key + " commands"
	}
	controls := "Enter send  ·  Alt+Enter newline  ·  " + palette + "  ·  Ctrl+D exit"
	if m.active {
		controls = m.runControls()
	} else if m.blocked != "" {
		_, step := m.setupStep()
		controls = "Enter " + step + "  ·  " + palette + "  ·  /settings  ·  Ctrl+D exit"
	}
	hints := fitHints(controls, width)
	if hints != controls {
		// Too narrow for every hint: keep the two that matter most.
		short := palette + "  ·  Ctrl+D exit"
		if m.active {
			short = "Ctrl+C cancel  ·  Ctrl+D exit"
		} else if m.blocked != "" {
			_, step := m.setupStep()
			short = "Enter " + step + "  ·  Ctrl+D exit"
		}
		hints = fitHints(short, width)
	}
	footer := []string{ansi.Truncate(hints, width, "…")}
	draft := m.input
	if draft == "" && m.blocked != "" {
		_, step := m.setupStep()
		draft = "Press Enter to " + step
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
	if m.hasSlashSuggestions() && available > 2 {
		// Tab cycles these, so they must be visible here too.
		names := []string{}
		for _, match := range m.slashMatches() {
			names = append(names, match.command)
		}
		files = append(files, ansi.Truncate("Tab: "+strings.Join(names, " "), width, "…"))
		available--
	}
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
	needle := ""
	if m.find != nil && review == "" {
		needle = strings.ToLower(m.find.query)
	}
	shown := 0
	if visible > 0 && m.a == nil && m.blocked != "" && review == "" {
		// The setup screen is prose: wrap it at word boundaries.
		var lines []string
		for _, line := range strings.Split(body, "\n") {
			lines = append(lines, wordWrap(Safe(line), width)...)
		}
		for _, line := range lines[max(0, len(lines)-visible):] {
			rows = append(rows, ansi.Truncate(line, width, "…"))
			shown++
		}
	} else if visible > 0 {
		for _, row := range m.visibleRows(body, width, visible, review != "") {
			text := row.text
			if needle != "" && strings.Contains(strings.ToLower(text), needle) {
				text = "▸ " + text
			}
			rows = append(rows, ansi.Truncate(text, width, "…"))
			shown++
		}
	}
	m.reviewHidden = review != "" && shown == 0
	rows = append(rows, files...)
	rows = append(rows, input...)
	rows = append(rows, footer...)
	view := tea.NewView(strings.Join(rows, "\n"))
	view.AltScreen = true
	view.MouseMode = mouseMode()
	return view
}
func Run(a *agent.Agent, notice, draft string) error {
	applyTheme(a.Config.Theme)
	title := ""
	if a.Config.SessionDir != "" && a.Store != nil {
		if meta, err := session.ReadMeta(a.Config.SessionDir, a.Store.ID); err == nil {
			title = Safe(meta.Title)
		}
	}
	m := &model{a: a, events: make(chan tea.Msg, 128), width: 80, height: 24, status: "ready", closed: make(chan struct{}), todos: append([]core.Todo{}, a.Todos...), keys: keymapFor(&a.Config), sessionTitle: title}
	m.insertInput(Safe(draft))
	if !m.unparkView() {
		var promptTimes map[string][]time.Time
		if a.Store != nil {
			promptTimes = userPromptTimes(a.Store.Records())
		}
		m.loadHistory(screenHistory(a), promptTimes)
		if len(m.log) > 128<<10 {
			m.trimLog()
		}
	}
	m.restoreLastRun()
	if a.Store != nil {
		// Prompts held when this same chat was left for a picker or settings.
		m.queue, m.queueHeld = unparkQueue(a.Store.ID)
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
	e := runProgram(m)
	close(m.closed)
	if m.cancel != nil {
		m.cancelRun()
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
		parkQueue(a.Store.ID, m.queue, m.queueHeld)
		m.parkView()
		return &ChatNavigation{Action: m.navigation, Session: a.Store.ID, Project: a.Store.Project, Config: &a.Config, Backdrop: m.View().Content, Draft: m.input, BeforeTurn: m.forkBefore, RevertPrompt: m.revertPrompt, RevertOccurrence: m.revertOccurrence}
	}
	return e
}

// Setup steps a blocked chat can lead to.
const (
	FixConnect   = "connect"   // add a provider
	FixWorkspace = "workspace" // choose a project folder
	FixSessions  = "sessions"  // choose another session (the requested one cannot open)
)

// setupStep is what Enter does on a blocked chat: the navigation and how
// the hints name it.
func (m *model) setupStep() (action, label string) {
	switch {
	case m.fix == FixWorkspace:
		return FixWorkspace, "choose a project folder"
	case m.fix == FixSessions:
		return FixSessions, "choose another session"
	case m.a != nil:
		return "provider", "fix the provider"
	default:
		return FixConnect, "add a provider"
	}
}

// RunBlocked renders the chat surface before a usable agent can be created.
// reason's first paragraph is shown on the home screen; fix is the setup
// step Enter opens (FixConnect or FixWorkspace).
func RunBlocked(status, reason, fix, draft string) error {
	m := &model{width: 80, height: 24, status: status, blocked: reason, fix: fix, log: reason, keys: keymapFor(nil)}
	m.insertInput(Safe(draft))
	err := runProgram(m)
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

// dropPartialReply removes the streamed text of an abandoned reply, with any
// reasoning shown after it, from the transcript.
// logSpan is text appended to log at an absolute position (logTrimmed
// included).
type logSpan struct {
	at   int
	text string
}

// trimLog keeps the transcript bounded, remembering how much was cut.
func (m *model) trimLog() {
	over := len(m.log) - (128 << 10)
	if over <= 0 {
		return
	}
	// Cut at a line start: a line cut through loses its marker, and a
	// collapsed tool detail or page preview would then show as text.
	if next := strings.IndexByte(m.log[over:], '\n'); next >= 0 {
		over += next + 1
	} else {
		over = len(m.log)
	}
	dropped := m.log[:over]
	m.log = m.log[over:]
	m.logTrimmed += over
	// A code block that began in the dropped part keeps its opening.
	if fence, _ := fenceContext(dropped); fence != "" {
		m.log = fence + "\n" + m.log
		m.logTrimmed -= len(fence) + 1
	}
}

// dropChildPartial removes one subagent's provisional text from exactly the
// places it was appended, newest first, and shifts every later span; the
// parent's and other children's text is never touched, even when equal.
func (m *model) dropChildPartial(task string) {
	pieces := m.childPartial[task]
	delete(m.childPartial, task)
	for i := len(pieces) - 1; i >= 0; i-- {
		p := pieces[i]
		at := p.at - m.logTrimmed
		if at < 0 || at+len(p.text) > len(m.log) || m.log[at:at+len(p.text)] != p.text {
			continue // trimmed away, or not where it was written
		}
		m.log = m.log[:at] + m.log[at+len(p.text):]
		for id, spans := range m.childPartial {
			for j := range spans {
				if spans[j].at > p.at {
					spans[j].at -= len(p.text)
				}
			}
			m.childPartial[id] = spans
		}
		if m.partialStart > at {
			m.partialStart -= len(p.text)
		}
	}
}

func (m *model) dropPartialReply() {
	switch {
	case m.partialReply == "":
	case m.partialStart <= len(m.log) && strings.HasPrefix(m.log[m.partialStart:], m.partialReply):
		m.log = m.log[:m.partialStart]
	case strings.HasSuffix(m.log, m.partialReply): // the log was trimmed since
		m.log = m.log[:len(m.log)-len(m.partialReply)]
	}
	m.partialReply = ""
}

// commandPreview reports whether an approval preview is for a command
// (shell, background job or formatter), which runs exactly as written.
func commandPreview(preview string) bool {
	for _, prefix := range []string{"Command: ", "Background command: ", "Formatter "} {
		if strings.HasPrefix(preview, prefix) {
			return true
		}
	}
	return false
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
	case "suspend":
		return m, tea.Suspend
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

// sessionScopeLabel says what "s" (allow for this session) will cover.
func sessionScopeLabel(p core.Permission) string {
	scope := agent.GrantScope(p)
	switch {
	case scope == "":
		return "session"
	case strings.HasPrefix(scope, "prefix:"):
		return "session for `" + Safe(strings.TrimPrefix(scope, "prefix:")) + " …`"
	case strings.HasPrefix(scope, "origin:"):
		return "session for " + Safe(strings.TrimPrefix(scope, "origin:"))
	}
	return "session for this exact command"
}

// overlayOpen reports whether a picker or panel covers the conversation; the
// wheel then moves its selection instead of scrolling the transcript.
func (m *model) overlayOpen() bool {
	return m.palette || m.export != nil || m.fork != nil || m.changesPicker != nil || m.jobsPicker != nil ||
		m.copyPicker != nil || m.themePicker != nil || m.timeline != nil || m.infoPanel != nil
}

// restoreStashedDraft puts a stashed draft back if the composer is free.
func (m *model) restoreStashedDraft() {
	if m.stashedDraft == "" {
		return
	}
	if strings.TrimSpace(m.input) == "" {
		m.input, m.inputCursor = m.stashedDraft, len([]rune(m.stashedDraft))
	}
	m.stashedDraft = ""
}

// stashDraft sets a non-empty draft aside before a palette command writes
// into the composer.
func (m *model) stashDraft() {
	if strings.TrimSpace(m.input) != "" && !strings.HasPrefix(strings.TrimSpace(m.input), "/") {
		m.stashedDraft = m.input
	}
}
