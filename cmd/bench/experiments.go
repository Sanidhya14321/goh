package main

import (
	"bufio"
	"encoding/json"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// reproducerAudit enables auditReproducers (the -reproducer flag).
var reproducerAudit bool

// retrievalStats measures how the agent found the code a fix needs, from
// the tools it called (session journals), against the task's gold files.
type retrievalStats struct {
	Gold          int      `json:"gold_files"`
	Read          int      `json:"gold_files_read"`
	Recall        float64  `json:"recall"`
	FirstGoldSecs *float64 `json:"seconds_to_first_gold,omitempty"`
	FilesTouched  int      `json:"files_touched"` // distinct paths read or listed by a search
	RepeatReads   int      `json:"repeat_reads"`  // read_file calls of a path already read
	Missed        []string `json:"missed,omitempty"`
}

// retrievalTelemetry reads tool_start records (arguments) and tool_result
// records (search and repo_map paths) from every session journal.
func retrievalTelemetry(sessions string, gold []string, start time.Time) *retrievalStats {
	want := map[string]bool{}
	for _, g := range gold {
		want[path.Clean(g)] = true
	}
	touched, read := map[string]bool{}, map[string]int{}
	var first *float64
	journals, _ := filepath.Glob(filepath.Join(sessions, "*.jsonl"))
	note := func(p string, at int64) {
		p = path.Clean(filepath.ToSlash(p))
		touched[p] = true
		if want[p] && first == nil && at > 0 {
			secs := time.UnixMilli(at).Sub(start).Seconds()
			if secs < 0 {
				secs = 0
			}
			first = &secs
		}
	}
	for _, j := range journals {
		f, err := os.Open(j)
		if err != nil {
			continue
		}
		sc := bufio.NewScanner(f)
		sc.Buffer(make([]byte, 64<<10), 16<<20)
		for sc.Scan() {
			var rec struct {
				Type string          `json:"type"`
				Time int64           `json:"time"`
				Data json.RawMessage `json:"data"`
			}
			if json.Unmarshal(sc.Bytes(), &rec) != nil {
				continue
			}
			switch rec.Type {
			case "tool_start":
				var d struct {
					Name      string `json:"name"`
					Arguments struct {
						Path string `json:"path"`
					} `json:"arguments"`
				}
				if json.Unmarshal(rec.Data, &d) == nil && d.Name == "read_file" && d.Arguments.Path != "" {
					p := path.Clean(filepath.ToSlash(d.Arguments.Path))
					read[p]++
					note(p, rec.Time)
				}
			case "tool_result":
				var d struct {
					Result struct {
						Content string `json:"content"`
					} `json:"result"`
				}
				if json.Unmarshal(rec.Data, &d) != nil {
					continue
				}
				var listed struct {
					Matches []struct{ Path string } `json:"matches"`
					Symbols []struct{ Path string } `json:"symbols"`
					Files   []string                `json:"files"`
				}
				if json.Unmarshal([]byte(d.Result.Content), &listed) == nil {
					for _, m := range listed.Matches {
						note(m.Path, rec.Time)
					}
					for _, m := range listed.Symbols {
						note(m.Path, rec.Time)
					}
					for _, p := range listed.Files {
						note(p, rec.Time)
					}
				}
			}
		}
		f.Close()
	}
	s := &retrievalStats{Gold: len(want), FirstGoldSecs: first, FilesTouched: len(touched)}
	for _, n := range read {
		if n > 1 {
			s.RepeatReads += n - 1
		}
	}
	for g := range want {
		if read[g] > 0 {
			s.Read++
		} else {
			s.Missed = append(s.Missed, g)
		}
	}
	sort.Strings(s.Missed)
	if s.Gold > 0 {
		s.Recall = float64(s.Read) / float64(s.Gold)
	}
	return s
}

// auditReproducers runs test files the agent added (not hidden, not in the
// baseline) on two isolated copies: the pre-run baseline plus those files,
// and the graded candidate. A test that fails on the baseline by a test
// failure and passes on the candidate is evidence ("fail_to_pass"). An
// invalid or missing test is "unknown", never a failure of the fix.
func auditReproducers(t task, pristine, grade string, baseline, candidate map[string]string, hidden map[string]bool, base string) string {
	var added []string
	for rel := range candidate {
		if _, ok := baseline[rel]; !ok && !hidden[rel] && testLikePath(rel) {
			added = append(added, rel)
		}
	}
	if len(added) == 0 {
		return "none"
	}
	sort.Strings(added)
	before := filepath.Join(base, "reproducer-baseline")
	if copyTree(pristine, before) != nil {
		return "unknown: copy failed"
	}
	for _, rel := range added {
		b, err := os.ReadFile(filepath.Join(grade, filepath.FromSlash(rel)))
		if err != nil {
			return "unknown: " + err.Error()
		}
		target := filepath.Join(before, filepath.FromSlash(rel))
		os.MkdirAll(filepath.Dir(target), 0o755)
		if os.WriteFile(target, b, 0o644) != nil {
			return "unknown: copy failed"
		}
	}
	command := reproducerCommand(added)
	if command == "" {
		return "unknown: no runner for " + strings.Join(added, ",")
	}
	run := func(dir string) checkResult {
		var box string
		if t.Image != "" {
			var err error
			if box, err = startContainer(t.Image, dir, false); err != nil {
				return checkResult{Kind: checkToolError}
			}
			defer stopContainer(box)
		}
		return runCheck(box, dir, command, 600, nil)
	}
	b, a := run(before), run(grade)
	switch {
	case b.Kind == checkTestFailure && a.Kind == checkPassed:
		return "fail_to_pass"
	case b.Kind == checkPassed && a.Kind == checkPassed:
		return "no_signal" // passes without the fix: not a reproducer
	case a.Kind == checkTestFailure:
		return "fails_on_candidate"
	}
	return "unknown: baseline " + b.Kind + ", candidate " + a.Kind
}

// reproducerCommand runs the added tests: their Go packages, or the Python
// modules through unittest.
func reproducerCommand(added []string) string {
	var pkgs, mods []string
	seen := map[string]bool{}
	for _, rel := range added {
		switch {
		case strings.HasSuffix(rel, "_test.go"):
			p := "./" + path.Dir(rel)
			if !seen[p] {
				seen[p] = true
				pkgs = append(pkgs, p)
			}
		case strings.HasSuffix(rel, ".py"):
			mods = append(mods, strings.ReplaceAll(strings.TrimSuffix(rel, ".py"), "/", "."))
		}
	}
	switch {
	case len(pkgs) > 0:
		return "go test " + strings.Join(pkgs, " ")
	case len(mods) > 0:
		return "python3 -m unittest -v " + strings.Join(mods, " ")
	}
	return ""
}
