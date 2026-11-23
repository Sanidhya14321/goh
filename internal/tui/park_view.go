package tui

import "sync"

// A picker, settings screen or agent switch closes the chat and opens it
// again (see ChatNavigation). The screen the user was looking at is parked
// here and shown again as it was, scroll position and display toggles
// included, instead of being rebuilt from the journal, so leaving and coming
// back looks continuous. It is only reused for the same session with an
// unchanged journal; anything else (a fork, a revert, another session)
// rebuilds the transcript from the journal.
type parkedViewState struct {
	session      string
	records      int
	log          string
	logTrimmed   int
	scrollOffset int
	details      detailsMode
	hideThinking bool
	timestamps   bool
	hideSidebar  bool
	skillNotice  bool
	durationMS   int64
	usage        int
}

var parkedView struct {
	sync.Mutex
	state *parkedViewState
}

func (m *model) parkView() {
	if m.a == nil || m.a.Store == nil {
		return
	}
	m.flushReasoning()
	parkedView.Lock()
	defer parkedView.Unlock()
	parkedView.state = &parkedViewState{
		session: m.a.Store.ID, records: len(m.a.Store.Records()),
		log: m.log, logTrimmed: m.logTrimmed, scrollOffset: m.scrollOffset,
		details: m.details, hideThinking: m.hideThinking, timestamps: m.timestamps,
		hideSidebar: m.hideSidebar, skillNotice: m.skillNoticeShown,
		durationMS: m.durationMS, usage: m.usage,
	}
}

// unparkView restores the parked screen when it belongs to this chat's
// session and the journal has not changed since; it reports whether it did.
// The parked screen is used at most once.
func (m *model) unparkView() bool {
	parkedView.Lock()
	p := parkedView.state
	parkedView.state = nil
	parkedView.Unlock()
	if p == nil || m.a == nil || m.a.Store == nil || p.session != m.a.Store.ID || p.records != len(m.a.Store.Records()) {
		return false
	}
	m.log, m.logTrimmed, m.scrollOffset = p.log, p.logTrimmed, p.scrollOffset
	m.details, m.hideThinking, m.timestamps, m.hideSidebar = p.details, p.hideThinking, p.timestamps, p.hideSidebar
	m.skillNoticeShown = p.skillNotice
	m.durationMS, m.usage = p.durationMS, p.usage // the last run's time and tokens
	// Prompt history (↑) comes from the journal, as when it is rebuilt.
	for _, message := range screenHistory(m.a) {
		if message.Role == "user" && !message.Automatic {
			m.rememberPrompt(message.Content)
		}
	}
	return true
}
