package runtime

import (
	"os"
	"sort"
	"sync"
	"time"
)

// Phase timing: where a run's wall time goes, for measuring the live path.
// Off by default; GOH_PHASES=1 turns it on for top-level runs, which then
// emit one run_phases event just before run_finished (which stays last).
// The collector is run-local and bounded, holds integer durations, sizes and
// counts only (no prompts, payloads, headers or URLs), and writes nothing to
// the journal, so measuring does not add durable writes to what it measures.
// Every method is safe on a nil collector, which is what a run without
// GOH_PHASES has.

// ProcessStarted is when the command-line process began, set by the entry
// point; the first run reports the time from it to the run's start.
var ProcessStarted time.Time

var processReported sync.Once

const (
	maxPhaseRequests  = 64
	maxPhaseIntervals = 1024
)

type requestPhase struct {
	Purpose     string `json:"purpose"`
	StartMS     int64  `json:"start_ms"`
	FirstTextMS int64  `json:"first_text_ms,omitempty"` // 0: no text (a tool-only reply)
	EndMS       int64  `json:"end_ms"`
	RetryWaitMS int64  `json:"retry_wait_ms,omitempty"`
	Model       string `json:"model,omitempty"`
	Fallback    bool   `json:"fallback,omitempty"`
	Failed      bool   `json:"failed,omitempty"`
}

type toolPhase struct {
	Calls   int   `json:"calls"`
	TotalMS int64 `json:"total_ms"`
	MaxMS   int64 `json:"max_ms"`
}

type runPhases struct {
	mu    sync.Mutex
	start time.Time

	startupMS     int64 // process start to run start (first run of a process)
	firstRequest  int64 // run start to the first model request (setup)
	assemblyMS    int64
	assemblies    int
	promptBytes   int // the latest request's size
	compactMS     int64
	compactions   int
	requests      []requestPhase
	moreRequests  int
	lastReplyEnd  int64
	tools         map[string]*toolPhase
	running       map[string]toolRun
	intervals     [][2]int64
	approvalMS    int64
	approvals     int
	questionMS    int64
	questions     int
	finish        map[string]int64
	finishStarted int64
}

type toolRun struct {
	name string
	at   int64
}

func phasesEnabled() bool { return os.Getenv("GOH_PHASES") == "1" }

func newRunPhases() *runPhases {
	p := &runPhases{start: time.Now(), firstRequest: -1, tools: map[string]*toolPhase{}, running: map[string]toolRun{}, finish: map[string]int64{}}
	if !ProcessStarted.IsZero() {
		processReported.Do(func() { p.startupMS = p.start.Sub(ProcessStarted).Milliseconds() })
	}
	return p
}

// at is milliseconds since the run started (monotonic).
func (p *runPhases) at() int64 { return time.Since(p.start).Milliseconds() }

// since returns a function that reports the milliseconds elapsed since now.
func since() func() int64 {
	t := time.Now()
	return func() int64 { return time.Since(t).Milliseconds() }
}

func (p *runPhases) assembled(ms int64, bytes int) {
	if p == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.assemblyMS += ms
	p.assemblies++
	p.promptBytes = bytes
}

func (p *runPhases) compacted(ms int64, happened bool) {
	if p == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.compactMS += ms
	if happened {
		p.compactions++
	}
}

// requestStarted opens a provider attempt and returns its index (-1 when
// the bounded list is full or timing is off).
func (p *runPhases) requestStarted(purpose string) int {
	if p == nil {
		return -1
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	now := p.at()
	if p.firstRequest < 0 {
		p.firstRequest = now
	}
	if len(p.requests) >= maxPhaseRequests {
		p.moreRequests++
		return -1
	}
	p.requests = append(p.requests, requestPhase{Purpose: purpose, StartMS: now})
	return len(p.requests) - 1
}

func (p *runPhases) firstText(i int) {
	if p == nil || i < 0 {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.requests[i].FirstTextMS == 0 {
		p.requests[i].FirstTextMS = max(1, p.at())
	}
}

func (p *runPhases) requestEnded(i int, model string, fallback, failed bool) {
	if p == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	now := p.at()
	if !failed {
		p.lastReplyEnd = now
	}
	if i < 0 {
		return
	}
	r := &p.requests[i]
	r.EndMS, r.Model, r.Fallback, r.Failed = now, model, fallback, failed
}

// retryWaited records the pause taken before attempt i (an overloaded
// provider's back-off).
func (p *runPhases) retryWaited(i int, d time.Duration) {
	if p == nil || i < 0 {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.requests[i].RetryWaitMS = d.Milliseconds()
}

func (p *runPhases) toolStarted(id, name string) {
	if p == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.running) < maxPhaseIntervals {
		p.running[id] = toolRun{name: name, at: p.at()}
	}
}

func (p *runPhases) toolFinished(id string) {
	if p == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	run, ok := p.running[id]
	if !ok {
		return
	}
	delete(p.running, id)
	end := p.at()
	t := p.tools[run.name]
	if t == nil {
		if len(p.tools) >= 64 {
			return
		}
		t = &toolPhase{}
		p.tools[run.name] = t
	}
	d := end - run.at
	t.Calls++
	t.TotalMS += d
	t.MaxMS = max(t.MaxMS, d)
	if len(p.intervals) < maxPhaseIntervals {
		p.intervals = append(p.intervals, [2]int64{run.at, end})
	}
}

func (p *runPhases) waited(question bool, ms int64) {
	if p == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if question {
		p.questionMS += ms
		p.questions++
		return
	}
	p.approvalMS += ms
	p.approvals++
}

func (p *runPhases) finishing() {
	if p == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.finishStarted = p.at()
}

func (p *runPhases) finishStep(name string, ms int64) {
	if p == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.finish[name] += ms
}

// unionMS is the wall time covered by the intervals: parallel reads that
// overlap count once.
func unionMS(intervals [][2]int64) int64 {
	if len(intervals) == 0 {
		return 0
	}
	sorted := append([][2]int64(nil), intervals...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i][0] < sorted[j][0] })
	total, lo, hi := int64(0), sorted[0][0], sorted[0][1]
	for _, iv := range sorted[1:] {
		if iv[0] > hi {
			total += hi - lo
			lo, hi = iv[0], iv[1]
			continue
		}
		hi = max(hi, iv[1])
	}
	return total + hi - lo
}

// summary is the run_phases event data.
func (p *runPhases) summary() map[string]any {
	p.mu.Lock()
	defer p.mu.Unlock()
	total := p.at()
	var toolSum int64
	for _, t := range p.tools {
		toolSum += t.TotalMS
	}
	var requestMS, retryWait int64
	firstText := int64(0)
	for _, r := range p.requests {
		requestMS += r.EndMS - r.StartMS
		retryWait += r.RetryWaitMS
		if firstText == 0 && r.FirstTextMS > 0 && r.Purpose == "turn" {
			firstText = r.FirstTextMS
		}
	}
	d := map[string]any{
		"total_ms":              total,
		"setup_ms":              max(0, p.firstRequest),
		"assembly_ms":           p.assemblyMS,
		"assemblies":            p.assemblies,
		"prompt_bytes":          p.promptBytes,
		"compaction_ms":         p.compactMS,
		"compactions":           p.compactions,
		"requests":              p.requests,
		"request_ms":            requestMS,
		"retry_wait_ms":         retryWait,
		"first_text_ms":         firstText,
		"tools":                 p.tools,
		"tool_ms":               toolSum,
		"tool_wall_ms":          unionMS(p.intervals),
		"approval_wait_ms":      p.approvalMS,
		"approvals":             p.approvals,
		"question_wait_ms":      p.questionMS,
		"questions":             p.questions,
		"finish":                p.finish,
		"final_reply_to_end_ms": max(0, total-p.lastReplyEnd),
	}
	if p.firstRequest < 0 {
		d["setup_ms"] = total // no model request was made
	}
	if p.startupMS > 0 {
		d["process_startup_ms"] = p.startupMS
	}
	if p.moreRequests > 0 {
		d["requests_not_listed"] = p.moreRequests
	}
	return d
}
