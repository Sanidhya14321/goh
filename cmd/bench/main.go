// Command bench runs pinned coding tasks through goh and regrades every
// result in a clean copy with hidden tests, following the SWE-bench design:
// the agent never sees the hidden tests, and a task counts as resolved only
// when both its fail-to-pass and pass-to-pass checks succeed.
//
// A task is a folder with task.json, a repo/ starting tree and an optional
// hidden/ tree that is copied over the agent's result before grading:
//
//	{"id": "go-off-by-one", "prompt": "...", "setup": ["go mod download"],
//	 "fail_to_pass": "go test ./... -run TestSum", "pass_to_pass": "go test ./...",
//	 "max_run_seconds": 600, "max_tool_calls": 80}
//
// Nothing is sent anywhere except through goh itself, which uses the user's
// configured provider. Run with --dry-run to check tasks without a model.
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

type task struct {
	ID            string   `json:"id"`
	Prompt        string   `json:"prompt"`
	Setup         []string `json:"setup,omitempty"`
	FailToPass    string   `json:"fail_to_pass"`
	PassToPass    string   `json:"pass_to_pass,omitempty"`
	MaxRunSeconds int      `json:"max_run_seconds,omitempty"`
	MaxToolCalls  int      `json:"max_tool_calls,omitempty"`
	// Image, when set, isolates the task in Docker: setup runs in a
	// container with network, while the agent's shell commands and grading
	// run in containers with no network. Only the attempt folder and a
	// shared dependency cache are mounted; no host credentials.
	Image string `json:"image,omitempty"`
	// Git, when set, is the starting tree: a repository at a pinned commit
	// (fetched once into a local cache) instead of a bundled repo/ folder.
	Git *struct {
		URL    string `json:"url"`
		Commit string `json:"commit"`
	} `json:"git,omitempty"`
	dir string
}

var fullCommit = regexp.MustCompile(`^[0-9a-f]{40}$`)

// sourceTree returns the folder holding the task's starting tree.
func (t task) sourceTree() (string, error) {
	if t.Git == nil {
		return filepath.Join(t.dir, "repo"), nil
	}
	if !fullCommit.MatchString(t.Git.Commit) {
		return "", fmt.Errorf("task %s: git.commit must be a full 40-character commit ID", t.ID)
	}
	cache, err := filepath.Abs(filepath.Join(filepath.Dir(filepath.Dir(t.dir)), ".cache", t.ID+"-"+t.Git.Commit[:12]))
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(filepath.Join(cache, ".ok")); err == nil {
		return cache, nil
	}
	os.RemoveAll(cache)
	for _, args := range [][]string{{"init", "-q", cache}, {"-C", cache, "fetch", "-q", "--depth", "1", t.Git.URL, t.Git.Commit}, {"-C", cache, "checkout", "-q", "FETCH_HEAD"}} {
		if out, err := exec.Command("git", args...).CombinedOutput(); err != nil {
			return "", fmt.Errorf("git %v: %v %s", args, err, out)
		}
	}
	os.RemoveAll(filepath.Join(cache, ".git"))
	return cache, os.WriteFile(filepath.Join(cache, ".ok"), nil, 0o644)
}

// Result is one task attempt under one arm.
type Result struct {
	Task           string  `json:"task"`
	Arm            string  `json:"arm"`
	Attempt        int     `json:"attempt"`
	Status         string  `json:"status"`  // goh run status
	Outcome        string  `json:"outcome"` // goh outcome
	ExitCode       int     `json:"exit_code"`
	Resolved       bool    `json:"resolved"`
	FalseComplete  bool    `json:"false_completion"` // completed but the grader fails
	ToolCalls      int     `json:"tool_calls"`
	PromptTokens   int     `json:"prompt_tokens"`
	OutputTokens   int     `json:"output_tokens"`
	CacheRead      int     `json:"cache_read_tokens"`
	Seconds        float64 `json:"seconds"`
	FailToPassOK   bool    `json:"fail_to_pass_ok"`
	PassToPassOK   bool    `json:"pass_to_pass_ok"`
	HiddenTampered bool    `json:"hidden_tests_overwritten"` // agent edited files the hidden tree replaces
	Error          string  `json:"error,omitempty"`
}

func main() {
	tasksDir := flag.String("tasks", "bench/tasks", "folder of task folders")
	gohBin := flag.String("goh", "goh", "goh binary to benchmark")
	out := flag.String("out", "bench/results", "results folder (one run per timestamped subfolder)")
	only := flag.String("only", "", "comma-separated task IDs")
	attempts := flag.Int("attempts", 1, "attempts per task and arm")
	dry := flag.Bool("dry-run", false, "check that each task's fail-to-pass fails on repo/ and passes with hidden/ reference, without a model")
	var arms []string
	flag.Func("arm", `name=extra goh run flags, repeatable (e.g. "base=" or "nocheck=--completion-check=false")`, func(v string) error {
		arms = append(arms, v)
		return nil
	})
	flag.Parse()
	if len(arms) == 0 {
		arms = []string{"base="}
	}
	if resolved, err := exec.LookPath(*gohBin); err == nil {
		*gohBin, _ = filepath.Abs(resolved) // tasks run in their own folders
	}
	tasks, err := loadTasks(*tasksDir, *only)
	if err != nil {
		fail(err)
	}
	if *dry {
		os.Exit(dryRun(tasks))
	}
	runDir := filepath.Join(*out, time.Now().UTC().Format("20060102T150405Z"))
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		fail(err)
	}
	f, err := os.Create(filepath.Join(runDir, "results.jsonl"))
	if err != nil {
		fail(err)
	}
	defer f.Close()
	var all []Result
	for _, armSpec := range arms {
		name, extra, _ := strings.Cut(armSpec, "=")
		for _, t := range tasks {
			for n := 1; n <= *attempts; n++ {
				r := runTask(t, *gohBin, name, strings.Fields(extra), n, filepath.Join(runDir, "work"))
				all = append(all, r)
				b, _ := json.Marshal(r)
				f.Write(append(b, '\n'))
				fmt.Fprintf(os.Stderr, "%-10s %-28s #%d resolved=%v status=%s outcome=%s %.0fs\n", name, t.ID, n, r.Resolved, r.Status, r.Outcome, r.Seconds)
			}
		}
	}
	summary := summarize(all)
	os.WriteFile(filepath.Join(runDir, "summary.md"), []byte(summary), 0o644)
	fmt.Print(summary)
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "bench:", err)
	os.Exit(2)
}

func loadTasks(dir, only string) ([]task, error) {
	want := map[string]bool{}
	for _, id := range strings.Split(only, ",") {
		if id != "" {
			want[id] = true
		}
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var tasks []task
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		b, err := os.ReadFile(filepath.Join(dir, e.Name(), "task.json"))
		if err != nil {
			continue
		}
		var t task
		if err := json.Unmarshal(b, &t); err != nil || t.ID == "" || t.Prompt == "" || t.FailToPass == "" {
			return nil, fmt.Errorf("%s: task.json needs id, prompt and fail_to_pass", e.Name())
		}
		if len(want) > 0 && !want[t.ID] {
			continue
		}
		t.dir = filepath.Join(dir, e.Name())
		tasks = append(tasks, t)
	}
	sort.Slice(tasks, func(i, j int) bool { return tasks[i].ID < tasks[j].ID })
	if len(tasks) == 0 {
		return nil, errors.New("no tasks found")
	}
	return tasks, nil
}

// runTask: fresh copy -> setup -> goh run -> clean regrade with hidden tests.
func runTask(t task, goh, arm string, extra []string, attempt int, workRoot string) Result {
	r := Result{Task: t.ID, Arm: arm, Attempt: attempt}
	base, err := filepath.Abs(filepath.Join(workRoot, fmt.Sprintf("%s-%s-%d", arm, t.ID, attempt)))
	if err != nil {
		r.Error = err.Error()
		return r
	}
	work, grade, sessions := filepath.Join(base, "agent"), filepath.Join(base, "grade"), filepath.Join(base, "sessions")
	src, err := t.sourceTree()
	if err != nil {
		r.Error = "source: " + err.Error()
		return r
	}
	if err := copyTree(src, work); err != nil {
		r.Error = "copy: " + err.Error()
		return r
	}
	var agentBox string
	if t.Image != "" {
		setupBox, err := startContainer(t.Image, base, true)
		if err != nil {
			r.Error = err.Error()
			return r
		}
		for _, s := range t.Setup {
			if code, out := csh(setupBox, work, s, 900); code != 0 {
				stopContainer(setupBox)
				r.Error = fmt.Sprintf("setup %q exit %d: %s", s, code, tail(out))
				return r
			}
		}
		stopContainer(setupBox)
		if agentBox, err = startContainer(t.Image, base, false); err != nil {
			r.Error = err.Error()
			return r
		}
		defer stopContainer(agentBox)
	} else {
		for _, s := range t.Setup {
			if code, out := sh(work, s, 600); code != 0 {
				r.Error = fmt.Sprintf("setup %q exit %d: %s", s, code, tail(out))
				return r
			}
		}
	}
	args := []string{"run", "--project", work, "--prompt", t.Prompt, "--format", "json", "--allow", "all", "--session-dir", sessions}
	if t.MaxRunSeconds > 0 {
		args = append(args, "--max-run-seconds", fmt.Sprint(t.MaxRunSeconds))
	}
	if t.MaxToolCalls > 0 {
		args = append(args, "--max-tool-calls", fmt.Sprint(t.MaxToolCalls))
	}
	if agentBox != "" {
		args = append(args, "--sandbox-container", agentBox)
	}
	args = append(args, extra...)
	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(max(t.MaxRunSeconds, 600)+120)*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, goh, args...)
	cmd.Dir = work
	stdout, _ := cmd.StdoutPipe()
	// goh's stderr is kept per attempt so failures can be diagnosed.
	if errFile, err := os.Create(filepath.Join(base, "goh-stderr.log")); err == nil {
		defer errFile.Close()
		cmd.Stderr = errFile
	}
	if err := cmd.Start(); err != nil {
		r.Error = "start goh: " + err.Error()
		return r
	}
	readEvents(stdout, &r)
	err = cmd.Wait()
	r.Seconds = time.Since(start).Seconds()
	if ee := (*exec.ExitError)(nil); errors.As(err, &ee) {
		r.ExitCode = ee.ExitCode()
	}
	if r.Status == "" {
		if b, _ := os.ReadFile(filepath.Join(base, "goh-stderr.log")); len(b) > 0 {
			r.Error = "goh: " + tail(string(b))
		}
	}
	// Clean regrade: the agent's final tree, then the hidden tests on top.
	if err := copyTree(work, grade); err != nil {
		r.Error = "regrade copy: " + err.Error()
		return r
	}
	hidden := filepath.Join(t.dir, "hidden")
	if _, err := os.Stat(hidden); err == nil {
		r.HiddenTampered = differs(hidden, work)
		if err := copyTree(hidden, grade); err != nil {
			r.Error = "hidden tests: " + err.Error()
			return r
		}
	}
	var gradeBox string
	if t.Image != "" {
		// A fresh container: nothing the agent left running can interfere.
		if gradeBox, err = startContainer(t.Image, base, false); err != nil {
			r.Error = err.Error()
			return r
		}
		defer stopContainer(gradeBox)
	}
	code, _ := csh(gradeBox, grade, t.FailToPass, 900)
	r.FailToPassOK = code == 0
	r.PassToPassOK = true
	if t.PassToPass != "" {
		code, _ = csh(gradeBox, grade, t.PassToPass, 900)
		r.PassToPassOK = code == 0
	}
	r.Resolved = r.FailToPassOK && r.PassToPassOK
	r.FalseComplete = r.Status == "completed" && !r.Resolved
	return r
}

// readEvents folds goh's JSON event stream into the result.
func readEvents(rd io.Reader, r *Result) {
	sc := bufio.NewScanner(rd)
	sc.Buffer(make([]byte, 1<<20), 64<<20)
	for sc.Scan() {
		var e struct {
			Type string          `json:"type"`
			Data json.RawMessage `json:"data"`
		}
		if json.Unmarshal(sc.Bytes(), &e) != nil {
			continue
		}
		switch e.Type {
		case "tool_requested":
			r.ToolCalls++
		case "assistant_completed":
			var d struct {
				Usage *struct {
					Prompt     int `json:"prompt_tokens"`
					Completion int `json:"completion_tokens"`
					CacheRead  int `json:"cache_read_tokens"`
				} `json:"usage"`
			}
			if json.Unmarshal(e.Data, &d) == nil && d.Usage != nil {
				r.PromptTokens += d.Usage.Prompt
				r.OutputTokens += d.Usage.Completion
				r.CacheRead += d.Usage.CacheRead
			}
		case "run_finished":
			var d struct{ Status, Outcome string }
			if json.Unmarshal(e.Data, &d) == nil {
				r.Status, r.Outcome = d.Status, d.Outcome
			}
		}
	}
}

// sh runs a task's own setup command on the host with the full environment
// (it may need network credentials or caches); model-edited code goes
// through csh instead.
func sh(dir, command string, seconds int) (int, string) {
	return shEnv(dir, command, seconds, nil)
}

// shEnv runs command with env (nil: the current environment).
func shEnv(dir, command string, seconds int, env []string) (int, string) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(seconds)*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "/bin/sh", "-c", command)
	cmd.Dir = dir
	cmd.Env = env
	out, err := cmd.CombinedOutput()
	if err == nil {
		return 0, string(out)
	}
	if ee := (*exec.ExitError)(nil); errors.As(err, &ee) {
		return ee.ExitCode(), string(out)
	}
	return -1, string(out) + err.Error()
}

func tail(s string) string {
	if len(s) > 400 {
		return "…" + s[len(s)-400:]
	}
	return s
}

// copyTree copies regular files and folders (no symlinks) from src into dst.
func copyTree(src, dst string) error {
	return filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, p)
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		if !d.Type().IsRegular() || d.Name() == ".ok" {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, info.Mode().Perm())
	})
}

// differs reports whether any hidden file exists with other content in the
// agent's tree (the agent edited a file the graders replace).
func differs(hidden, work string) bool {
	changed := false
	filepath.WalkDir(hidden, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(hidden, p)
		a, _ := os.ReadFile(p)
		b, err := os.ReadFile(filepath.Join(work, rel))
		if err == nil && string(a) != string(b) {
			changed = true
		}
		return nil
	})
	return changed
}

// dryRun checks each task is well formed: fail_to_pass must fail on the
// starting tree with hidden tests and pass with the reference solution.
func dryRun(tasks []task) int {
	bad := 0
	for _, t := range tasks {
		tmp, _ := os.MkdirTemp(dryRoot(), "goh-bench-dry-")
		defer os.RemoveAll(tmp)
		var box string
		if t.Image != "" {
			var err error
			if box, err = startContainer(t.Image, tmp, true); err != nil {
				fmt.Printf("%-30s BROKEN container: %v\n", t.ID, err)
				bad++
				continue
			}
		}
		src, err := t.sourceTree()
		if err != nil {
			fmt.Printf("%-30s BROKEN source: %v\n", t.ID, err)
			bad++
			continue
		}
		start := filepath.Join(tmp, "start")
		copyTree(src, start)
		copyTree(filepath.Join(t.dir, "hidden"), start)
		for _, s := range t.Setup {
			csh(box, start, s, 600)
		}
		before, _ := csh(box, start, t.FailToPass, 900)
		ref := filepath.Join(tmp, "reference")
		copyTree(src, ref)
		copyTree(filepath.Join(t.dir, "solution"), ref)
		copyTree(filepath.Join(t.dir, "hidden"), ref)
		for _, s := range t.Setup {
			csh(box, ref, s, 600)
		}
		after, _ := csh(box, ref, t.FailToPass, 900)
		p2p := 0
		if t.PassToPass != "" {
			p2p, _ = csh(box, ref, t.PassToPass, 900)
		}
		stopContainer(box)
		ok := before != 0 && after == 0 && p2p == 0
		if !ok {
			bad++
		}
		fmt.Printf("%-30s fails-before=%v passes-with-solution=%v pass-to-pass=%v %s\n", t.ID, before != 0, after == 0, p2p == 0, map[bool]string{true: "OK", false: "BROKEN"}[ok])
	}
	if bad > 0 {
		return 1
	}
	return 0
}

func summarize(rs []Result) string {
	type agg struct {
		n, resolved, falseDone, budget int
		seconds                        float64
		prompt, output, cache, calls   int
	}
	by := map[string]*agg{}
	var names []string
	for _, r := range rs {
		a := by[r.Arm]
		if a == nil {
			a = &agg{}
			by[r.Arm] = a
			names = append(names, r.Arm)
		}
		a.n++
		if r.Resolved {
			a.resolved++
		}
		if r.FalseComplete {
			a.falseDone++
		}
		if r.Status == "budget_exhausted" {
			a.budget++
		}
		a.seconds += r.Seconds
		a.prompt += r.PromptTokens
		a.output += r.OutputTokens
		a.cache += r.CacheRead
		a.calls += r.ToolCalls
	}
	var b strings.Builder
	b.WriteString("| Arm | Runs | Resolved | False completions | Budget stops | Avg tool calls | Avg seconds | Input tokens | Cached | Output tokens |\n|---|---|---|---|---|---|---|---|---|---|\n")
	for _, n := range names {
		a := by[n]
		fmt.Fprintf(&b, "| %s | %d | %d (%.0f%%) | %d | %d | %.1f | %.0f | %d | %d | %d |\n", n, a.n, a.resolved, 100*float64(a.resolved)/float64(a.n), a.falseDone, a.budget, float64(a.calls)/float64(a.n), a.seconds/float64(a.n), a.prompt, a.cache, a.output)
	}
	return b.String()
}

func asExit(err error, target **exec.ExitError) bool { return errors.As(err, target) }

// dryRoot is a folder Docker can mount (Colima shares the home folder, not
// the system temporary folder).
func dryRoot() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	dir := filepath.Join(home, ".cache", "goh-bench-dry")
	os.MkdirAll(dir, 0o755)
	return dir
}
