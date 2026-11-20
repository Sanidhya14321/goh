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
	"github.com/Sanidhya14321/goh/internal/sandbox"

	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
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
	// FailToPassTests and PassToPassTests, when set, are the test
	// identities that must pass (go test names or unittest/pytest ids). A
	// task without them is graded by exit code and labelled "aggregate".
	FailToPassTests []string `json:"fail_to_pass_tests,omitempty"`
	PassToPassTests []string `json:"pass_to_pass_tests,omitempty"`
	// GoldFiles are the files a fix needs (benchmark-only telemetry, never
	// shown to the agent): retrieval recall and time to first useful
	// evidence are measured against them.
	GoldFiles []string `json:"gold_files,omitempty"`
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
	FalseComplete  bool    `json:"false_completion"`   // completed but the grader fails
	FalseVerified  bool    `json:"false_verification"` // goh said verified but the grader fails
	ToolCalls      int     `json:"tool_calls"`
	PromptTokens   int     `json:"prompt_tokens"`
	OutputTokens   int     `json:"output_tokens"`
	CacheRead      int     `json:"cache_read_tokens"`
	Seconds        float64 `json:"seconds"`
	FailToPassOK   bool    `json:"fail_to_pass_ok"`
	PassToPassOK   bool    `json:"pass_to_pass_ok"`
	HiddenTampered bool    `json:"hidden_tests_overwritten"` // agent edited files the hidden tree replaces
	Error          string  `json:"error,omitempty"`
	// FailureKind separates infrastructure failures (copy, setup,
	// container, start, regrade, grader) from the agent's own result;
	// Scored is false for infrastructure failures, which are excluded from
	// the resolution rate but count against end-to-end reliability.
	FailureKind string `json:"failure_kind,omitempty"`
	Scored      bool   `json:"scored"`
	// Grading is "identity" when expected test identities were checked,
	// "aggregate" when only exit codes were available (legacy tasks).
	Grading    string      `json:"grading"`
	FailToPass checkResult `json:"fail_to_pass"`
	PassToPass checkResult `json:"pass_to_pass"`
	// ProtectedChanged lists test and discovery files the candidate changed
	// relative to the pre-run baseline; they are restored before grading.
	// CandidateTestsResolved is the separate evaluation with the
	// candidate's own test changes kept.
	ProtectedChanged       []string `json:"protected_changed,omitempty"`
	CandidateTestsResolved *bool    `json:"candidate_tests_resolved,omitempty"`
	// Stream checks: malformed lines and terminal events in goh's output.
	MalformedEvents int    `json:"malformed_events"`
	TerminalEvents  int    `json:"terminal_events"`
	StreamProblem   string `json:"stream_problem,omitempty"`
	// Route and accounting from the run itself.
	Model    string                   `json:"model,omitempty"`
	Protocol string                   `json:"protocol,omitempty"`
	Profile  string                   `json:"profile,omitempty"`
	Usage    map[string]purposeTokens `json:"usage_by_purpose,omitempty"` // from the session journals: turn, compaction, title, fallback; children included
	Children int                      `json:"child_sessions"`
	// Retrieval telemetry (tasks with gold_files) and the optional
	// generated-reproducer audit.
	Retrieval  *retrievalStats `json:"retrieval,omitempty"`
	Reproducer string          `json:"reproducer,omitempty"`
	Scheduled  time.Time       `json:"scheduled"`
}

// purposeTokens is reported usage for one purpose across every session the
// attempt wrote (the top-level run and its children).
type purposeTokens struct {
	Calls      int `json:"calls"`
	Prompt     int `json:"prompt_tokens"`
	Completion int `json:"completion_tokens"`
	CacheRead  int `json:"cache_read_tokens"`
}

func main() {
	tasksDir := flag.String("tasks", "bench/tasks", "folder of task folders")
	gohBin := flag.String("goh", "goh", "goh binary to benchmark")
	out := flag.String("out", "bench/results", "results folder (one run per timestamped subfolder)")
	only := flag.String("only", "", "comma-separated task IDs")
	attempts := flag.Int("attempts", 1, "attempts per task and arm")
	flag.BoolVar(&reproducerAudit, "reproducer", false, "also run test files the agent added on the baseline and the candidate, in isolated copies, and record whether they fail before and pass after (evidence only; never changes the grade)")
	flag.BoolVar(&hostGrading, "host-grading", false, "run tasks that have no image: the agent's shell and grading then run model-edited code on this machine, unsandboxed during grading")
	flag.BoolVar(&recordIdentities, "record-identities", false, "with -dry-run: save the observed fail-to-pass and pass-to-pass test identities into each valid task.json, so later grading must see those exact tests pass")
	dry := flag.Bool("dry-run", false, "check that each task's fail-to-pass fails on repo/ and passes with hidden/ reference, without a model")
	var armSpecs []string
	flag.Func("arm", `name=extra goh run flags, repeatable (e.g. "base=" or "nocheck=--completion-check=false"); flags are split like shell words but never evaluated`, func(v string) error {
		armSpecs = append(armSpecs, v)
		return nil
	})
	flag.Parse()
	if len(armSpecs) == 0 {
		armSpecs = []string{"base="}
	}
	arms, err := parseArms(armSpecs)
	if err != nil {
		fail(err)
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
	// The effective isolation of every task, before anything runs.
	for _, t := range tasks {
		switch {
		case t.Image != "":
			fmt.Fprintf(os.Stderr, "isolation %-28s container %s (setup with network; agent and grader without)\n", t.ID, t.Image)
		case hostGrading && !sandbox.Supported():
			fail(fmt.Errorf("task %s: -host-grading needs goh's native sandbox for the agent's shell, which this system does not have", t.ID))
		case hostGrading:
			fmt.Fprintf(os.Stderr, "isolation %-28s HOST: agent shell confined by goh's native sandbox (--access auto, shell not granted), grading unsandboxed on this machine\n", t.ID)
		default:
			fmt.Fprintf(os.Stderr, "isolation %-28s refused (no image; pass -host-grading)\n", t.ID)
		}
	}
	if err := os.MkdirAll(*out, 0o755); err != nil {
		fail(err)
	}
	// A fresh folder per invocation, even when two start in one second.
	runDir, err := os.MkdirTemp(*out, time.Now().UTC().Format("20060102T150405Z")+"-")
	if err != nil {
		fail(err)
	}
	if err := writeManifest(runDir, *gohBin, tasks, arms, *attempts); err != nil {
		fail(err)
	}
	f, err := os.Create(filepath.Join(runDir, "results.jsonl"))
	if err != nil {
		fail(err)
	}
	defer f.Close()
	var all []Result
	// Arms are interleaved per task and attempt, in an order that rotates,
	// so drift in the provider or machine affects every arm alike.
	for _, t := range tasks {
		for n := 1; n <= *attempts; n++ {
			for i := range arms {
				a := arms[(i+n-1)%len(arms)]
				r := runTask(t, *gohBin, a.Name, a.Args, n, filepath.Join(runDir, "work"))
				all = append(all, r)
				b, _ := json.Marshal(r)
				f.Write(append(b, '\n'))
				fmt.Fprintf(os.Stderr, "%-10s %-28s #%d resolved=%v status=%s outcome=%s grading=%s %s %.0fs\n", a.Name, t.ID, n, r.Resolved, r.Status, r.Outcome, r.Grading, r.FailureKind, r.Seconds)
			}
		}
	}
	summary := summarize(all)
	os.WriteFile(filepath.Join(runDir, "summary.md"), []byte(summary), 0o644)
	fmt.Print(summary)
}

// writeManifest records what the run used: the harness build, the tasks
// (source, image, task.json digest), the arms and every scheduled attempt.
func writeManifest(runDir, goh string, tasks []task, arms []arm, attempts int) error {
	digest := func(path string) string {
		b, err := os.ReadFile(path)
		if err != nil {
			return ""
		}
		sum := sha256.Sum256(b)
		return hex.EncodeToString(sum[:])
	}
	version, _ := exec.Command(goh, "version").Output()
	type taskInfo struct {
		ID, Image, Git, Commit, TaskDigest string
	}
	var ts []taskInfo
	var schedule []string
	for _, t := range tasks {
		info := taskInfo{ID: t.ID, Image: t.Image, TaskDigest: digest(filepath.Join(t.dir, "task.json"))}
		if t.Git != nil {
			info.Git, info.Commit = t.Git.URL, t.Git.Commit
		}
		ts = append(ts, info)
		for n := 1; n <= attempts; n++ {
			for i := range arms {
				schedule = append(schedule, fmt.Sprintf("%s/%s/%d", t.ID, arms[(i+n-1)%len(arms)].Name, n))
			}
		}
	}
	m := map[string]any{"started": time.Now().UTC(), "goh": goh, "goh_version": strings.TrimSpace(string(version)), "goh_sha256": digest(goh),
		"host_grading": hostGrading, "attempts": attempts, "arms": arms, "tasks": ts, "schedule": schedule,
		"grading_policy": "candidate tree with the pre-run test and discovery files restored and the hidden tests on top; candidate test changes are evaluated separately"}
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(runDir, "manifest.json"), b, 0o644)
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

// hostGrading allows tasks without an image (see the -host-grading flag).
var hostGrading bool

// runTask: fresh copy -> setup -> goh run -> clean regrade with hidden tests.
// With an image, each stage has its own container that mounts only its own
// tree: the agent's container sees agent/ and is stopped before grading, and
// the grader's sees only grade/, so nothing the agent leaves running or
// writes can reach the grade.
func runTask(t task, goh, arm string, extra []string, attempt int, workRoot string) (r Result) {
	r = Result{Task: t.ID, Arm: arm, Attempt: attempt, Scheduled: time.Now().UTC(), Grading: "aggregate"}
	if len(t.FailToPassTests) > 0 {
		r.Grading = "identity"
	}
	infra := func(kind, msg string) Result {
		r.FailureKind, r.Error, r.Scored = kind, msg, false
		return r
	}
	if t.Image == "" && !hostGrading {
		return infra("isolation", "task has no image, so grading would run model-edited code on this machine; pass -host-grading to accept that")
	}
	if err := os.MkdirAll(workRoot, 0o755); err != nil {
		return infra("copy", err.Error())
	}
	// A fresh folder for every attempt, never reused.
	base, err := os.MkdirTemp(workRoot, fmt.Sprintf("%s-%s-%d-", arm, t.ID, attempt))
	if err != nil {
		return infra("copy", err.Error())
	}
	if base, err = filepath.Abs(base); err != nil {
		return infra("copy", err.Error())
	}
	work, grade, sessions, pristine := filepath.Join(base, "agent"), filepath.Join(base, "grade"), filepath.Join(base, "sessions"), filepath.Join(base, "baseline")
	src, err := t.sourceTree()
	if err != nil {
		return infra("source", err.Error())
	}
	if err := copyTree(src, work); err != nil {
		return infra("copy", err.Error())
	}
	var agentBox string
	if t.Image != "" {
		setupBox, err := startContainer(t.Image, work, true)
		if err != nil {
			return infra("container", err.Error())
		}
		for _, s := range t.Setup {
			if code, out := csh(setupBox, work, s, 900); code != 0 {
				stopContainer(setupBox)
				return infra("setup", fmt.Sprintf("setup %q exit %d: %s", s, code, tail(out)))
			}
		}
		stopContainer(setupBox)
		if agentBox, err = startContainer(t.Image, work, false); err != nil {
			return infra("container", err.Error())
		}
		defer stopContainer(agentBox)
	} else {
		for _, s := range t.Setup {
			if code, out := sh(work, s, 600); code != 0 {
				return infra("setup", fmt.Sprintf("setup %q exit %d: %s", s, code, tail(out)))
			}
		}
	}
	// The pre-run baseline: what the agent started from, after setup.
	if err := copyTree(work, pristine); err != nil {
		return infra("copy", err.Error())
	}
	baseline, err := snapshotHashes(pristine)
	if err != nil {
		return infra("copy", err.Error())
	}
	args := []string{"run", "--project", work, "--prompt", t.Prompt, "--format", "json", "--session-dir", sessions}
	if t.Image != "" {
		args = append(args, "--allow", "all") // shell runs in the agent's container
	} else {
		// Host mode: every local tool except shell is granted; commands run
		// only where Auto access confines them to this task in goh's native
		// sandbox, and otherwise stop for permission.
		args = append(args, "--access", "auto", "--allow", hostGrants)
	}
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
		return infra("start", "start goh: "+err.Error())
	}
	readEvents(stdout, &r)
	err = cmd.Wait()
	r.Seconds = time.Since(start).Seconds()
	r.Scored = true // the agent ran: from here on the result is its own
	// Nothing the agent started may outlive its run into grading.
	stopContainer(agentBox)
	if ee := (*exec.ExitError)(nil); errors.As(err, &ee) {
		r.ExitCode = ee.ExitCode()
	}
	if ctx.Err() != nil {
		r.FailureKind = "agent_timeout"
	}
	if r.Status == "" {
		if b, _ := os.ReadFile(filepath.Join(base, "goh-stderr.log")); len(b) > 0 {
			r.Error = "goh: " + tail(string(b))
		}
	}
	r.Usage, r.Children = journalUsage(sessions)
	if len(t.GoldFiles) > 0 {
		r.Retrieval = retrievalTelemetry(sessions, t.GoldFiles, start)
	}
	candidate, err := snapshotHashes(work)
	if err != nil {
		return infra("regrade_copy", err.Error())
	}
	hidden := filepath.Join(t.dir, "hidden")
	hiddenPaths := map[string]bool{}
	if hs, err := snapshotHashes(hidden); err == nil {
		for rel := range hs {
			hiddenPaths[rel] = true
		}
	}
	r.ProtectedChanged = protectedChanges(baseline, candidate, hiddenPaths)
	for _, c := range r.ProtectedChanged {
		rel, _, _ := strings.Cut(c, " (")
		if hiddenPaths[rel] {
			r.HiddenTampered = true
		}
	}
	// Grade from the pinned baseline: the pre-run tree plus the
	// candidate's changes to everything but test and discovery files (and
	// the hidden tests' paths), then the hidden tests on top. Tests the
	// candidate added or changed stay out, so the grader's own fixtures and
	// discovery decide; they are graded separately below.
	if err := applyCandidate(pristine, work, grade, baseline, candidate, hiddenPaths); err != nil {
		return infra("regrade_copy", "regrade copy: "+err.Error())
	}
	if len(hiddenPaths) > 0 {
		if err := copyTree(hidden, grade); err != nil {
			return infra("regrade_copy", "hidden tests: "+err.Error())
		}
	}
	gradeOne := func(dir string) (checkResult, checkResult, error) {
		var box string
		if t.Image != "" {
			// A fresh container that mounts only this tree.
			var err error
			if box, err = startContainer(t.Image, dir, false); err != nil {
				return checkResult{}, checkResult{}, err
			}
			defer stopContainer(box)
		}
		f2p := runCheck(box, dir, t.FailToPass, 900, t.FailToPassTests)
		p2p := checkResult{Kind: checkPassed}
		if t.PassToPass != "" {
			p2p = runCheck(box, dir, t.PassToPass, 900, t.PassToPassTests)
		}
		return f2p, p2p, nil
	}
	f2p, p2p, err := gradeOne(grade)
	if err != nil {
		return infra("grade_container", err.Error())
	}
	if f2p.Kind == checkToolError || p2p.Kind == checkToolError {
		return infra("grader", "a grading command could not run: "+tail(f2p.Output+p2p.Output))
	}
	r.FailToPass, r.PassToPass = f2p, p2p
	r.FailToPassOK, r.PassToPassOK = f2p.Kind == checkPassed, p2p.Kind == checkPassed
	r.Resolved = r.FailToPassOK && r.PassToPassOK
	r.FalseComplete = r.Status == "completed" && !r.Resolved
	r.FalseVerified = r.Outcome == "verified" && !r.Resolved
	if reproducerAudit {
		r.Reproducer = auditReproducers(t, pristine, work, grade, baseline, candidate, hiddenPaths, base)
	}
	if len(r.ProtectedChanged) > 0 {
		// Legitimate test changes are assessed apart: the candidate's own
		// tree with the hidden tests on top.
		own := filepath.Join(base, "grade-candidate-tests")
		if copyTree(work, own) == nil && (len(hiddenPaths) == 0 || copyTree(hidden, own) == nil) {
			if f, p, err := gradeOne(own); err == nil {
				ok := f.Kind == checkPassed && p.Kind == checkPassed
				r.CandidateTestsResolved = &ok
			}
		}
	}
	return r
}

// hostGrants are the tools granted in host mode: --allow all minus shell.
const hostGrants = "write_file,create_file,edit_file,apply_patch,todowrite,skill,task,child:write_file,child:create_file,child:edit_file,child:apply_patch,snapshot_undo,snapshot_redo"

// applyCandidate builds the grade tree: a copy of the pre-run baseline with
// the candidate's added, modified and deleted files applied, except test
// and discovery files and the hidden tests' paths.
func applyCandidate(pristine, work, grade string, baseline, candidate map[string]string, hidden map[string]bool) error {
	if err := copyTree(pristine, grade); err != nil {
		return err
	}
	for rel, sum := range candidate {
		if testLikePath(rel) || hidden[rel] || baseline[rel] == sum {
			continue
		}
		b, err := os.ReadFile(filepath.Join(work, filepath.FromSlash(rel)))
		if err != nil {
			return err
		}
		target := filepath.Join(grade, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		mode := os.FileMode(0o644)
		if info, err := os.Stat(filepath.Join(work, filepath.FromSlash(rel))); err == nil {
			mode = info.Mode().Perm()
		}
		if err := os.WriteFile(target, b, mode); err != nil {
			return err
		}
	}
	for rel := range baseline {
		if _, kept := candidate[rel]; kept || testLikePath(rel) || hidden[rel] {
			continue
		}
		if err := os.Remove(filepath.Join(grade, filepath.FromSlash(rel))); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

// journalUsage sums reported usage by purpose across every session journal
// the attempt wrote (children have their own), and counts the children.
func journalUsage(dir string) (map[string]purposeTokens, int) {
	out := map[string]purposeTokens{}
	journals, _ := filepath.Glob(filepath.Join(dir, "*.jsonl"))
	for _, path := range journals {
		f, err := os.Open(path)
		if err != nil {
			continue
		}
		sc := bufio.NewScanner(f)
		sc.Buffer(make([]byte, 64<<10), 16<<20)
		for sc.Scan() {
			var rec struct {
				Type string `json:"type"`
				Data struct {
					Purpose    string `json:"purpose"`
					Prompt     int    `json:"prompt_tokens"`
					Completion int    `json:"completion_tokens"`
					CacheRead  int    `json:"cache_read_tokens"`
				} `json:"data"`
			}
			if json.Unmarshal(sc.Bytes(), &rec) != nil || rec.Type != "usage" {
				continue
			}
			p := out[rec.Data.Purpose]
			p.Calls++
			p.Prompt += rec.Data.Prompt
			p.Completion += rec.Data.Completion
			p.CacheRead += rec.Data.CacheRead
			out[rec.Data.Purpose] = p
		}
		f.Close()
	}
	return out, max(0, len(journals)-1)
}

// readEvents folds goh's JSON event stream into the result and checks it:
// every line must parse and the stream must end with exactly one
// run_finished.
func readEvents(rd io.Reader, r *Result) {
	sc := bufio.NewScanner(rd)
	sc.Buffer(make([]byte, 1<<20), 64<<20)
	last := ""
	for sc.Scan() {
		var e struct {
			Type string          `json:"type"`
			Data json.RawMessage `json:"data"`
		}
		if json.Unmarshal(sc.Bytes(), &e) != nil || e.Type == "" {
			r.MalformedEvents++
			continue
		}
		last = e.Type
		switch e.Type {
		case "run_started":
			var d struct{ Model, Protocol, Profile string }
			if json.Unmarshal(e.Data, &d) == nil && r.Model == "" {
				r.Model, r.Protocol, r.Profile = d.Model, d.Protocol, d.Profile
			}
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
			r.TerminalEvents++
			var d struct{ Status, Outcome string }
			if json.Unmarshal(e.Data, &d) == nil {
				r.Status, r.Outcome = d.Status, d.Outcome
			}
		}
	}
	switch {
	case sc.Err() != nil:
		r.StreamProblem = "stream read error: " + sc.Err().Error()
	case r.TerminalEvents == 0:
		r.StreamProblem = "truncated: no run_finished"
	case r.TerminalEvents > 1:
		r.StreamProblem = fmt.Sprintf("%d run_finished events", r.TerminalEvents)
	case last != "run_finished":
		r.StreamProblem = "events after run_finished"
	case r.MalformedEvents > 0:
		r.StreamProblem = fmt.Sprintf("%d malformed event lines", r.MalformedEvents)
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

// dryRun checks each task is well formed, from isolated inputs per stage:
// on the starting tree (with hidden tests) the fail-to-pass check must fail
// by a test failure, not a build or setup error; with the reference
// solution it must pass; and the pass-to-pass tests must pass on both
// revisions, except the fail-to-pass tests themselves. Copy and setup
// errors make the task BROKEN. Tests run without network.
func dryRun(tasks []task) int {
	bad := 0
	for _, t := range tasks {
		ok, report := dryRunTask(t)
		if !ok {
			bad++
		}
		fmt.Printf("%-30s %s %s\n", t.ID, map[bool]string{true: "OK", false: "BROKEN"}[ok], report)
	}
	if bad > 0 {
		return 1
	}
	return 0
}

// dryStage builds one revision (source, overlays, setup) in its own folder
// and runs check commands in their own network-less container.
func dryStage(t task, tmp, name string, overlays ...string) (dir string, err error) {
	src, err := t.sourceTree()
	if err != nil {
		return "", fmt.Errorf("source: %w", err)
	}
	dir = filepath.Join(tmp, name)
	if err := copyTree(src, dir); err != nil {
		return "", fmt.Errorf("copy: %w", err)
	}
	for _, o := range overlays {
		if _, err := os.Stat(o); err != nil {
			continue
		}
		if err := copyTree(o, dir); err != nil {
			return "", fmt.Errorf("copy %s: %w", filepath.Base(o), err)
		}
	}
	var box string
	if t.Image != "" {
		if box, err = startContainer(t.Image, dir, true); err != nil {
			return "", fmt.Errorf("container: %w", err)
		}
		defer stopContainer(box)
	}
	for _, s := range t.Setup {
		if code, out := csh(box, dir, s, 600); code != 0 {
			return "", fmt.Errorf("setup %q exit %d: %s", s, code, tail(out))
		}
	}
	return dir, nil
}

func dryCheck(t task, dir, command string, expected []string) (checkResult, error) {
	var box string
	if t.Image != "" {
		var err error
		if box, err = startContainer(t.Image, dir, false); err != nil {
			return checkResult{}, fmt.Errorf("container: %w", err)
		}
		defer stopContainer(box)
	}
	return runCheck(box, dir, command, 900, expected), nil
}

// recordIdentities saves dry-run identities into task.json (-record-identities).
var recordIdentities bool

// saveIdentities writes the observed identities into task.json, keeping
// its other fields and their order.
func saveIdentities(path string, f2p, p2p []string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(b, &fields); err != nil {
		return err
	}
	sort.Strings(f2p)
	sort.Strings(p2p)
	if p2p == nil {
		p2p = []string{}
	}
	enc := func(v []string) string {
		out, _ := json.Marshal(v)
		return string(out)
	}
	text := strings.TrimRight(string(b), " \n")
	if _, ok := fields["fail_to_pass_tests"]; ok {
		// Already recorded: rewrite the whole object.
		fields["fail_to_pass_tests"] = json.RawMessage(enc(f2p))
		fields["pass_to_pass_tests"] = json.RawMessage(enc(p2p))
		out, err := json.MarshalIndent(fields, "", "  ")
		if err != nil {
			return err
		}
		return os.WriteFile(path, append(out, '\n'), 0o644)
	}
	if !strings.HasSuffix(text, "}") {
		return errors.New("task.json does not end with an object")
	}
	text = strings.TrimSuffix(text, "}")
	text = strings.TrimRight(text, " \n") + ",\n  \"fail_to_pass_tests\": " + enc(f2p) + ",\n  \"pass_to_pass_tests\": " + enc(p2p) + "\n}\n"
	var check map[string]any
	if err := json.Unmarshal([]byte(text), &check); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(text), 0o644)
}

func dryRunTask(t task) (bool, string) {
	tmp, err := os.MkdirTemp(dryRoot(), "goh-bench-dry-")
	if err != nil {
		return false, "temp: " + err.Error()
	}
	defer os.RemoveAll(tmp)
	hidden, solution := filepath.Join(t.dir, "hidden"), filepath.Join(t.dir, "solution")
	before, err := dryStage(t, tmp, "start", hidden)
	if err != nil {
		return false, "baseline " + err.Error()
	}
	ref, err := dryStage(t, tmp, "reference", solution, hidden)
	if err != nil {
		return false, "reference " + err.Error()
	}
	b, err := dryCheck(t, before, t.FailToPass, nil)
	if err != nil {
		return false, err.Error()
	}
	a, err := dryCheck(t, ref, t.FailToPass, t.FailToPassTests)
	if err != nil {
		return false, err.Error()
	}
	var problems []string
	// The baseline must fail on a test assertion: a build error or a
	// missing test would let any change look like a fix.
	if b.Kind != checkTestFailure {
		problems = append(problems, "fail-to-pass on baseline: "+b.Kind+" (want test_failure)")
	}
	if a.Kind != checkPassed {
		problems = append(problems, "fail-to-pass with solution: "+a.Kind+strings.Join(a.Missing, ","))
	}
	failing := b.Failed()
	for _, want := range t.FailToPassTests {
		if lookupTest(b.Tests, want) != "fail" {
			problems = append(problems, "expected failing test "+want+" did not fail on baseline ("+orNone(lookupTest(b.Tests, want))+")")
		}
	}
	p2pCount := 0
	var p2pIDs []string
	if t.PassToPass != "" {
		pb, err := dryCheck(t, before, t.PassToPass, nil)
		if err != nil {
			return false, err.Error()
		}
		pa, err := dryCheck(t, ref, t.PassToPass, t.PassToPassTests)
		if err != nil {
			return false, err.Error()
		}
		if pa.Kind != checkPassed {
			problems = append(problems, "pass-to-pass with solution: "+pa.Kind)
		}
		// Pass-to-pass identities: passing with the solution must also pass
		// on the baseline, except the tests the fix is meant to change.
		expectedFail := map[string]bool{}
		for _, f := range failing {
			expectedFail[f] = true
		}
		for _, name := range pa.Passed() {
			if expectedFail[name] {
				continue
			}
			switch pb.Tests[name] {
			case "pass":
				p2pCount++
				p2pIDs = append(p2pIDs, name)
			case "":
				if pb.Structured {
					problems = append(problems, "pass-to-pass test "+name+" does not exist on baseline")
				}
			default:
				problems = append(problems, "pass-to-pass test "+name+" is "+pb.Tests[name]+" on baseline")
			}
		}
		if !pb.Structured && pb.Exit != 0 && b.Kind == checkTestFailure && len(failing) == 0 {
			// No identities and a broad command that also runs the failing
			// test: only aggregate grading is possible.
			problems = append(problems, "pass-to-pass fails on baseline and its tests cannot be identified")
		}
	}
	grading := "aggregate"
	if len(t.FailToPassTests) > 0 {
		grading = "identity"
	}
	// Fail-to-pass identities: tests that failed on the baseline and pass
	// with the solution.
	var f2pIDs []string
	for _, name := range failing {
		if a.Tests[name] == "pass" {
			f2pIDs = append(f2pIDs, name)
		}
	}
	if recordIdentities && len(problems) == 0 {
		if len(f2pIDs) == 0 {
			problems = append(problems, "no fail-to-pass test identity observed; cannot record identities")
		} else if err := saveIdentities(filepath.Join(t.dir, "task.json"), f2pIDs, p2pIDs); err != nil {
			problems = append(problems, "record identities: "+err.Error())
		} else {
			grading = "identity (recorded)"
		}
	}
	report := fmt.Sprintf("grading=%s baseline=%s(failing=%d) solution=%s pass-to-pass-identities=%d", grading, b.Kind, len(failing), a.Kind, p2pCount)
	if len(problems) > 0 {
		report += " · " + strings.Join(problems, "; ")
	}
	return len(problems) == 0, report
}

func summarize(rs []Result) string {
	type agg struct {
		n, scored, resolved, falseDone, falseVerified, budget, infra, identity int
		seconds                                                                []float64
		prompt, output, cache, calls, auxPrompt, auxOutput                     int
		childPrompt, estimated                                                 int
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
		if !r.Scored {
			a.infra++
			continue
		}
		a.scored++
		if r.Grading == "identity" {
			a.identity++
		}
		if r.Resolved {
			a.resolved++
		}
		if r.FalseComplete {
			a.falseDone++
		}
		if r.FalseVerified {
			a.falseVerified++
		}
		if r.Status == "budget_exhausted" {
			a.budget++
		}
		a.seconds = append(a.seconds, r.Seconds)
		a.calls += r.ToolCalls
		if len(r.Usage) > 0 {
			// The durable totals from every session journal (top-level
			// turns, children, compaction, titles, fallbacks) are the
			// authoritative cost; the event stream sees only the top level.
			for purpose, u := range r.Usage {
				a.prompt += u.Prompt
				a.output += u.Completion
				a.cache += u.CacheRead
				if purpose != "turn" {
					a.auxPrompt += u.Prompt
					a.auxOutput += u.Completion
				}
			}
			a.childPrompt += max(0, journalTurnPrompt(r)-r.PromptTokens)
		} else {
			a.prompt += r.PromptTokens
			a.output += r.OutputTokens
			a.cache += r.CacheRead
			a.estimated++
		}
	}
	pct := func(x, n int) string {
		if n == 0 {
			return "n/a"
		}
		return fmt.Sprintf("%d/%d (%.0f%%)", x, n, 100*float64(x)/float64(n))
	}
	quantile := func(xs []float64, q float64) float64 {
		if len(xs) == 0 {
			return 0
		}
		s := append([]float64(nil), xs...)
		sort.Float64s(s)
		return s[min(len(s)-1, int(q*float64(len(s)-1)+0.5))]
	}
	var b strings.Builder
	b.WriteString("Resolution counts scored attempts (the agent ran and was graded); reliability counts every scheduled attempt, so infrastructure failures lower it but not resolution. False verification: goh reported `verified` but the grader failed. Token totals come from every session journal the attempt wrote (top-level turns, subagents, compaction, titles, fallbacks); \"Runs without journal usage\" fell back to the top-level event stream.\n\n")
	b.WriteString("| Arm | Scheduled | Infra errors | Resolved (scored) | Reliability (all) | False completions | False verification | Budget stops | Identity-graded | Avg tool calls | p50 s | p95 s | Input tokens | Cached | Output tokens | Aux input | Aux output | Child turn input | Runs without journal usage |\n|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|\n")
	for _, n := range names {
		a := by[n]
		calls := 0.0
		if a.scored > 0 {
			calls = float64(a.calls) / float64(a.scored)
		}
		fmt.Fprintf(&b, "| %s | %d | %d | %s | %s | %d | %d | %d | %d | %.1f | %.0f | %.0f | %d | %d | %d | %d | %d | %d | %d |\n", n, a.n, a.infra, pct(a.resolved, a.scored), pct(a.resolved, a.n), a.falseDone, a.falseVerified, a.budget, a.identity, calls, quantile(a.seconds, .5), quantile(a.seconds, .95), a.prompt, a.cache, a.output, a.auxPrompt, a.auxOutput, a.childPrompt, a.estimated)
	}
	return b.String()
}

// journalTurnPrompt is the prompt tokens of turns across all journals,
// children included (the event stream reports only the top level's).
func journalTurnPrompt(r Result) int { return r.Usage["turn"].Prompt }

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
