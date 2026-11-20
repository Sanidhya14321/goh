package main

import (
	"bufio"
	"encoding/json"
	"os"
	"path"
	"path/filepath"
	"regexp"
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
	reads := map[string]string{} // read_file call id -> path, until its result
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
					ID        string `json:"id"`
					Name      string `json:"name"`
					Arguments struct {
						Path string `json:"path"`
					} `json:"arguments"`
				}
				if json.Unmarshal(rec.Data, &d) == nil && d.Name == "read_file" && d.Arguments.Path != "" {
					reads[d.ID] = path.Clean(filepath.ToSlash(d.Arguments.Path))
				}
			case "tool_result":
				var d struct {
					ID     string `json:"id"`
					Result struct {
						Error   bool   `json:"error"`
						Content string `json:"content"`
					} `json:"result"`
				}
				if json.Unmarshal(rec.Data, &d) != nil || d.Result.Error {
					delete(reads, d.ID)
					continue // a failed call retrieved nothing
				}
				if p, ok := reads[d.ID]; ok {
					// Credited only once the read succeeded.
					read[p]++
					note(p, rec.Time)
					delete(reads, d.ID)
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
// baseline) in two isolated copies: the pre-run baseline plus those files,
// and the graded candidate plus those files. Only the same test, defined in
// an added file, observed failing before and passing after is evidence
// ("fail_to_pass"); other outcomes are "no_signal", "fails_on_candidate"
// or "unknown", never a failure of the fix.
func auditReproducers(t task, pristine, work, grade string, baseline, candidate map[string]string, hidden map[string]bool, base string) string {
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
	command := reproducerCommand(added)
	if command == "" {
		return "unknown: no runner for " + strings.Join(added, ",")
	}
	names := addedTestNames(work, added)
	if len(names) == 0 {
		return "unknown: no test functions found in " + strings.Join(added, ",")
	}
	prepare := func(name, from string) (string, error) {
		dir := filepath.Join(base, name)
		if err := copyTree(from, dir); err != nil {
			return "", err
		}
		for _, rel := range added {
			b, err := os.ReadFile(filepath.Join(work, filepath.FromSlash(rel)))
			if err != nil {
				return "", err
			}
			target := filepath.Join(dir, filepath.FromSlash(rel))
			os.MkdirAll(filepath.Dir(target), 0o755)
			if err := os.WriteFile(target, b, 0o644); err != nil {
				return "", err
			}
		}
		return dir, nil
	}
	before, err := prepare("reproducer-baseline", pristine)
	if err != nil {
		return "unknown: " + err.Error()
	}
	after, err := prepare("reproducer-candidate", grade)
	if err != nil {
		return "unknown: " + err.Error()
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
	b, a := run(before), run(after)
	if !b.Structured || !a.Structured {
		return "unknown: test results not recognised"
	}
	signal, fails := false, false
	for _, n := range names {
		bs, as := lookupTest(b.Tests, n), lookupTest(a.Tests, n)
		if bs == "fail" && as == "pass" {
			return "fail_to_pass"
		}
		if as == "fail" {
			fails = true
		}
		if bs == "pass" && as == "pass" {
			signal = true
		}
	}
	switch {
	case fails:
		return "fails_on_candidate"
	case signal:
		return "no_signal" // passes without the fix: not a reproducer
	}
	return "unknown: baseline " + b.Kind + ", candidate " + a.Kind
}

var (
	goTestFuncRe = regexp.MustCompile(`(?m)^func (Test\w+)\(`)
	pyTestFuncRe = regexp.MustCompile(`(?m)^\s+def (test\w*)\(`)
)

// addedTestNames lists the test functions the added files define, as
// identity suffixes (Go: TestX; unittest: module...test_x).
func addedTestNames(work string, added []string) []string {
	var out []string
	for _, rel := range added {
		b, err := os.ReadFile(filepath.Join(work, filepath.FromSlash(rel)))
		if err != nil {
			continue
		}
		if strings.HasSuffix(rel, ".go") {
			for _, m := range goTestFuncRe.FindAllSubmatch(b, -1) {
				out = append(out, string(m[1]))
			}
			continue
		}
		for _, m := range pyTestFuncRe.FindAllSubmatch(b, -1) {
			out = append(out, string(m[1]))
		}
	}
	return out
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
