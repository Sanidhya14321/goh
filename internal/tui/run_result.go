package tui

import (
	"encoding/json"
	"fmt"

	"github.com/Sanidhya14321/goh/internal/core"
	"github.com/Sanidhya14321/goh/internal/session"
)

// lastRunFromRecords returns the journal's latest settled run, or nil when
// the latest run never finished (or there is none).
func lastRunFromRecords(records []session.Record) *runResult {
	var last *runResult
	for _, r := range records {
		switch r.Type {
		case "run_started":
			last = nil
		case "run_finished":
			var data map[string]any
			if json.Unmarshal(r.Data, &data) != nil {
				continue
			}
			id, _ := data["run_id"].(string)
			rr := parseRunResult(core.Event{RunID: id, Data: data})
			last = &rr
		}
	}
	return last
}

// runResult is a run's settled terminal record (its run_finished event). The
// status line, the transcript and the queue decision all read it, so they
// agree on how the run ended and what its checks established.
type runResult struct {
	RunID     string
	Status    string
	Outcome   string
	Operation string
	Tracking  string
	Command   string // the check behind the outcome
	ExitCode  *int
	Source    string
}

func parseRunResult(e core.Event) runResult {
	b, _ := json.Marshal(e.Data)
	var d struct {
		Status    string `json:"status"`
		Outcome   string `json:"outcome"`
		Operation string `json:"operation"`
		Detail    struct {
			ChangeTracking string `json:"change_tracking"`
			Evidence       *struct {
				Source   string `json:"source"`
				Command  string `json:"command"`
				ExitCode *int   `json:"exit_code"`
			} `json:"evidence"`
		} `json:"verification_detail"`
	}
	json.Unmarshal(b, &d)
	r := runResult{RunID: e.RunID, Status: d.Status, Outcome: d.Outcome, Operation: d.Operation, Tracking: d.Detail.ChangeTracking}
	if ev := d.Detail.Evidence; ev != nil {
		r.Command, r.ExitCode, r.Source = ev.Command, ev.ExitCode, ev.Source
	}
	return r
}

// failedVerification reports whether the configured check still failed.
func (r *runResult) failedVerification() bool {
	return r != nil && r.Outcome == "failed_verification"
}

// label is the status text for a settled run: the status plus, for a
// completed run, what its checks established, in words (not only color).
func (r *runResult) label() string {
	if r == nil {
		return ""
	}
	if r.Status != "completed" {
		return r.Status
	}
	switch r.Outcome {
	case "failed_verification":
		s := "Verification failed"
		if r.Command != "" {
			s += ": " + Safe(oneLine(r.Command, 60))
		}
		if r.ExitCode != nil {
			s += fmt.Sprintf(" (exit %d)", *r.ExitCode)
		}
		return s
	case "verified":
		if r.Source == "configured" {
			return "completed · verified by your check"
		}
		return "completed · verified"
	case "unverified":
		if r.Tracking == "unknown" {
			return "completed · unverified (changes could not be tracked)"
		}
		return "completed · unverified: no check passed after the changes"
	case "no_changes":
		if r.Tracking == "unknown" {
			return "completed · change tracking unavailable"
		}
		return "completed"
	}
	return "completed"
}

// failureLine is the transcript line kept after a failed configured check.
func (r *runResult) failureLine() string {
	return "\n[" + r.label() + ". The changes were kept; review them, then ask for a fix or run the check again.]\n"
}

// restoreLastRun shows, in a reopened chat, how its latest run ended. The
// result stays tied to that run: the next turn's run_started clears it.
func (m *model) restoreLastRun() {
	if m.a == nil || m.a.Store == nil {
		return
	}
	if r := lastRunFromRecords(m.a.Store.Records()); r != nil && r.Outcome != "" {
		m.lastRun = r
		m.status = "Last run: " + r.label()
		if r.failedVerification() {
			m.log += r.failureLine()
		}
	}
}
