package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/Sanidhya14321/goh/internal/core"
	"github.com/Sanidhya14321/goh/internal/platform"
	"github.com/Sanidhya14321/goh/internal/tui"
)

// goh watch reruns the project's tests when files change and settle. When
// they fail, goh can fix them on the side: the working copy (committed and
// uncommitted changes, untracked files) is copied into the worktree
// <repo>-watch, a headless run fixes it there, the fix is checked with the
// same test command and saved as a patch. The working copy changes only when
// the user applies the patch.

const (
	watchOutputCap   = 64 << 10
	watchPromptTail  = 12 << 10
	maxSnapshotFiles = 5000
	maxSnapshotBytes = 64 << 20
)

type watchOptions struct {
	test        string
	timeout     time.Duration
	quiet       time.Duration
	auto        bool
	maxFixes    int
	passthrough []string
}

// testResult is one run of the test command.
type testResult struct {
	passed  bool
	output  string // tail, at most watchOutputCap
	seconds float64
	err     string // the command could not run or timed out
}

func runWatch(args []string, stdin io.Reader, stdout io.Writer) error {
	var passthrough []string
	for i, a := range args {
		if a == "--" {
			args, passthrough = args[:i], args[i+1:]
			break
		}
	}
	f := flag.NewFlagSet("watch", flag.ContinueOnError)
	test := f.String("test", "", "test command (default: verify.command from settings, else detected from the project)")
	timeout := f.Duration("timeout", 5*time.Minute, "test command time limit")
	quiet := f.Duration("quiet", 2*time.Second, "how long files must be unchanged before the tests run")
	auto := f.Bool("auto", false, "start a fix on the side whenever the tests fail (otherwise press f)")
	maxFixes := f.Int("max-fixes", 5, "with --auto: fixes started in this session at most")
	if err := f.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if f.NArg() > 0 {
		return errors.New("usage: goh watch [--test CMD] [--auto] [--quiet 2s] [--timeout 5m] [-- RUN FLAGS]")
	}
	if *timeout <= 0 || *timeout > time.Hour || *quiet < 200*time.Millisecond || *quiet > time.Minute || *maxFixes < 1 || *maxFixes > 50 {
		return errors.New("--timeout up to 1h, --quiet 200ms-1m, --max-fixes 1-50")
	}
	for _, a := range passthrough {
		if a == "--project" || strings.HasPrefix(a, "--project=") || a == "--format" || strings.HasPrefix(a, "--format=") || a == "--prompt" || strings.HasPrefix(a, "--prompt=") {
			return fmt.Errorf("%s is set by goh watch for each fix", a)
		}
	}
	git, err := exec.LookPath("git")
	if err != nil {
		return errors.New("goh watch needs git on PATH")
	}
	out, err := exec.Command(git, "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return errors.New("goh watch must run inside a Git repository")
	}
	root := filepath.Clean(strings.TrimSpace(string(out)))
	command := *test
	if command == "" {
		if c, err := core.LoadConfig(""); err == nil && c.Verify != nil {
			command = c.Verify.Command
		}
	}
	if command == "" {
		command = detectTestCommand(root)
	}
	if strings.TrimSpace(command) == "" {
		return errors.New("goh watch could not tell how to run this project's tests; pass --test CMD")
	}
	self, err := os.Executable()
	if err != nil {
		return err
	}
	if !hasAccessFlag(passthrough) {
		passthrough = append([]string{"--access", "edits"}, passthrough...)
	}
	opts := watchOptions{test: command, timeout: *timeout, quiet: *quiet, auto: *auto, maxFixes: *maxFixes, passthrough: passthrough}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	w := &watcher{git: git, root: root, self: self, opts: opts, out: stdout}
	var commands <-chan string
	if stdin != nil && terminalInput() {
		commands = readCommands(stdin)
	} else if !opts.auto {
		fmt.Fprintln(stdout, "Not a terminal: only the tests run (add --auto to start fixes).")
	}
	return w.loop(ctx, commands)
}

// terminalInput reports whether standard input is a terminal someone types
// in (the null device is a character device too).
func terminalInput() bool {
	if !interactive() {
		return false
	}
	in, err1 := os.Stdin.Stat()
	null, err2 := os.Stat(os.DevNull)
	return err1 == nil && (err2 != nil || !os.SameFile(in, null))
}

func hasAccessFlag(args []string) bool {
	for _, a := range args {
		if a == "--access" || strings.HasPrefix(a, "--access=") {
			return true
		}
	}
	return false
}

// detectTestCommand guesses the test command from the project's files.
func detectTestCommand(root string) string {
	exists := func(name string) bool { _, err := os.Stat(filepath.Join(root, name)); return err == nil }
	switch {
	case exists("go.mod"):
		return "go test ./..."
	case exists("Cargo.toml"):
		return "cargo test"
	case exists("package.json"):
		b, _ := os.ReadFile(filepath.Join(root, "package.json"))
		if bytes.Contains(b, []byte(`"test"`)) {
			return "npm test"
		}
	case exists("pyproject.toml") || exists("pytest.ini") || exists("setup.cfg") || exists("tox.ini"):
		return "python3 -m pytest -q"
	}
	if b, err := os.ReadFile(filepath.Join(root, "Makefile")); err == nil && bytes.Contains(b, []byte("\ntest:")) {
		return "make test"
	}
	return ""
}

func readCommands(r io.Reader) <-chan string {
	ch := make(chan string)
	go func() {
		defer close(ch)
		scanner := bufio.NewScanner(r)
		for scanner.Scan() {
			ch <- strings.ToLower(strings.TrimSpace(scanner.Text()))
		}
	}()
	return ch
}

type watcher struct {
	git, root, self string
	opts            watchOptions
	out             io.Writer

	failure  *testResult // the latest failing result, while failing
	failedAt string      // fingerprint it failed on
	fixedFor string      // fingerprint a fix was last started for
	fixes    int
	patch    string // the latest fix's patch file, not yet applied
}

func (w *watcher) say(format string, a ...any) {
	fmt.Fprintf(w.out, "%s  %s\n", time.Now().Format("15:04:05"), fmt.Sprintf(format, a...))
}

func (w *watcher) help() string {
	h := "Enter or f: fix the failure on the side · t: test now · a: apply the latest fix · q: quit"
	if w.opts.auto {
		h = "fixes start on failure · t: test now · a: apply the latest fix · q: quit"
	}
	return h
}

func (w *watcher) loop(ctx context.Context, commands <-chan string) error {
	w.say("Watching %s; tests: %s", tui.Safe(w.root), tui.Safe(w.opts.test))
	if commands != nil {
		w.say("%s", w.help())
	}
	tick := time.NewTicker(500 * time.Millisecond)
	defer tick.Stop()
	seen, tested := "", ""
	changedAt := time.Now()
	for {
		select {
		case <-ctx.Done():
			w.say("Stopped.")
			return nil
		case c, ok := <-commands:
			if !ok {
				commands = nil
				continue
			}
			switch c {
			case "q", "quit", "exit":
				return nil
			case "t":
				tested = w.test(ctx)
			case "a":
				if w.apply() {
					w.patch = ""
				}
			case "", "f":
				if w.failure == nil {
					w.say("Tests are passing; nothing to fix.")
					continue
				}
				w.fix(ctx)
			case "?", "h", "help":
				w.say("%s", w.help())
			default:
				w.say("Unknown command %q. %s", tui.Safe(c), w.help())
			}
		case <-tick.C:
			fp := w.fingerprint()
			if fp != seen {
				seen, changedAt = fp, time.Now()
			}
			// The first test runs at once; later ones once files settle.
			if fp == tested || tested != "" && time.Since(changedAt) < w.opts.quiet {
				continue
			}
			tested = w.test(ctx)
			fp = tested
			if w.failure != nil && w.opts.auto && w.fixedFor != fp {
				if w.fixes >= w.opts.maxFixes {
					w.say("Not starting a fix: --max-fixes %d reached.", w.opts.maxFixes)
				} else {
					w.fix(ctx)
				}
			}
		}
	}
}

// test runs the tests on the working copy and returns its fingerprint
// afterwards, so files the tests themselves write do not start another run.
func (w *watcher) test(ctx context.Context) string {
	w.say("Running tests…")
	r := runTests(ctx, w.root, w.opts.test, w.opts.timeout)
	fp := w.fingerprint()
	if ctx.Err() != nil {
		return fp
	}
	if r.passed {
		w.say("✓ tests pass (%.1fs)", r.seconds)
		w.failure, w.failedAt = nil, ""
		return fp
	}
	w.failure, w.failedAt = &r, fp
	reason := "✗ tests fail"
	if r.err != "" {
		reason = "✗ " + r.err
	}
	w.say("%s (%.1fs):", reason, r.seconds)
	fmt.Fprintln(w.out, indentTail(r.output, 15))
	return fp
}

func indentTail(s string, lines int) string {
	parts := strings.Split(strings.TrimRight(s, "\n"), "\n")
	if len(parts) > lines {
		parts = parts[len(parts)-lines:]
	}
	for i, p := range parts {
		parts[i] = "    " + tui.Safe(firstLine(p, 200))
	}
	return strings.Join(parts, "\n")
}

// runTests runs command in dir with a time limit, keeping the output's tail.
func runTests(ctx context.Context, dir, command string, timeout time.Duration) testResult {
	start := time.Now()
	cmd := testShell(command)
	cmd.Dir = dir
	var out tailBuffer
	out.limit = watchOutputCap
	cmd.Stdout, cmd.Stderr = &out, &out
	platform.NewGroup(cmd)
	r := testResult{}
	if err := cmd.Start(); err != nil {
		r.err = "the test command could not start: " + err.Error()
		return r
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	var err error
	select {
	case err = <-done:
	case <-timer.C:
		platform.KillGroup(cmd.Process.Pid)
		<-done
		r.err = fmt.Sprintf("the tests timed out after %s", timeout)
	case <-ctx.Done():
		platform.KillGroup(cmd.Process.Pid)
		<-done
		r.err = "cancelled"
	}
	r.seconds = time.Since(start).Seconds()
	r.output = out.String()
	r.passed = err == nil && r.err == ""
	return r
}

// tailBuffer keeps the last limit bytes written.
type tailBuffer struct {
	buf   []byte
	limit int
	cut   bool
}

func (t *tailBuffer) Write(p []byte) (int, error) {
	t.buf = append(t.buf, p...)
	if len(t.buf) > t.limit {
		t.buf = append(t.buf[:0], t.buf[len(t.buf)-t.limit:]...)
		t.cut = true
	}
	return len(p), nil
}

func (t *tailBuffer) String() string {
	s := strings.ToValidUTF8(string(t.buf), "")
	if t.cut {
		if i := strings.IndexByte(s, '\n'); i >= 0 {
			s = s[i+1:]
		}
		return "…\n" + s
	}
	return s
}

// fingerprint identifies the working copy's state: git's view of changed
// and untracked files, plus each one's size and modification time (a
// second edit to a modified file leaves the status unchanged). Ignored
// files are not watched.
func (w *watcher) fingerprint() string {
	out, err := exec.Command(w.git, "-C", w.root, "status", "--porcelain=v1", "-z", "--untracked-files=all").Output()
	if err != nil {
		return "error"
	}
	h := sha256.New()
	h.Write(out)
	if head, err := exec.Command(w.git, "-C", w.root, "rev-parse", "HEAD").Output(); err == nil {
		h.Write(head)
	}
	entries := strings.Split(string(out), "\x00")
	for i := 0; i < len(entries); i++ {
		e := entries[i]
		if len(e) < 4 {
			continue
		}
		path := e[3:]
		if e[0] == 'R' || e[0] == 'C' {
			i++ // the source path follows
		}
		if info, err := os.Lstat(filepath.Join(w.root, path)); err == nil {
			fmt.Fprintf(h, "%s %d %d\n", path, info.Size(), info.ModTime().UnixNano())
		}
	}
	return hex.EncodeToString(h.Sum(nil))
}

// gitIn runs git in dir and returns its trimmed output, with stderr in the
// error.
func gitIn(git, dir string, stdin []byte, args ...string) (string, error) {
	cmd := exec.Command(git, append([]string{"-C", dir}, args...)...)
	if stdin != nil {
		cmd.Stdin = bytes.NewReader(stdin)
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git %s: %s", args[0], firstLine(stderr.String()+" "+err.Error(), 200))
	}
	return strings.TrimSpace(string(out)), nil
}

// fixTree prepares <root>-watch as a copy of the working copy and returns
// its path and the commit holding that copy.
func fixTree(git, root string) (dir, base string, err error) {
	dir = root + "-watch"
	head, err := gitIn(git, root, nil, "rev-parse", "HEAD")
	if err != nil {
		return "", "", errors.New("goh watch needs at least one commit")
	}
	if _, statErr := os.Stat(dir); statErr == nil {
		top, err := gitTop(git, dir)
		listed := false
		if list, e := gitIn(git, root, nil, "worktree", "list", "--porcelain"); e == nil {
			for _, line := range strings.Split(list, "\n") {
				if p, ok := strings.CutPrefix(line, "worktree "); ok && sameDir(p, dir) {
					listed = true
				}
			}
		}
		if err != nil || !sameDir(top, dir) || !listed {
			return "", "", fmt.Errorf("%s exists and is not a worktree of this repository; move it aside", dir)
		}
	} else if _, err := gitIn(git, root, nil, "worktree", "add", "--detach", dir, head); err != nil {
		return "", "", err
	}
	if _, err := gitIn(git, dir, nil, "checkout", "-q", "-f", "--detach", head); err != nil {
		return "", "", err
	}
	if _, err := gitIn(git, dir, nil, "clean", "-fdq"); err != nil {
		return "", "", err
	}
	diff, err := exec.Command(git, "-C", root, "diff", "--binary", "HEAD").Output()
	if err != nil {
		return "", "", errors.New("git diff failed in the working copy")
	}
	if len(diff) > 0 {
		if _, err := gitIn(git, dir, diff, "apply", "--binary", "--whitespace=nowarn"); err != nil {
			return "", "", fmt.Errorf("could not copy your changes: %w", err)
		}
	}
	untracked, err := exec.Command(git, "-C", root, "ls-files", "-z", "--others", "--exclude-standard").Output()
	if err != nil {
		return "", "", errors.New("git ls-files failed in the working copy")
	}
	if err := copyUntracked(root, dir, strings.Split(strings.TrimSuffix(string(untracked), "\x00"), "\x00")); err != nil {
		return "", "", err
	}
	if _, err := gitIn(git, dir, nil, "add", "-A"); err != nil {
		return "", "", err
	}
	if _, err := gitIn(git, dir, nil, "-c", "user.name=goh", "-c", "user.email=goh@localhost", "-c", "commit.gpgsign=false",
		"commit", "-q", "--no-verify", "--allow-empty", "-m", "goh watch: copy of the working copy"); err != nil {
		return "", "", err
	}
	base, err = gitIn(git, dir, nil, "rev-parse", "HEAD")
	return dir, base, err
}

// copyUntracked copies untracked, non-ignored files (and symbolic links, as
// links) from root to dir, within bounds.
func copyUntracked(root, dir string, paths []string) error {
	if len(paths) > maxSnapshotFiles {
		return fmt.Errorf("more than %d untracked files; commit or ignore some", maxSnapshotFiles)
	}
	total := int64(0)
	for _, p := range paths {
		if p == "" {
			continue
		}
		src, dst := filepath.Join(root, filepath.FromSlash(p)), filepath.Join(dir, filepath.FromSlash(p))
		info, err := os.Lstat(src)
		if err != nil {
			continue // removed since git listed it
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return err
		}
		switch {
		case info.Mode()&os.ModeSymlink != 0:
			target, err := os.Readlink(src)
			if err != nil {
				return err
			}
			if err := os.Symlink(target, dst); err != nil {
				return err
			}
		case info.Mode().IsRegular():
			if total += info.Size(); total > maxSnapshotBytes {
				return fmt.Errorf("untracked files exceed %d MiB; commit or ignore some", maxSnapshotBytes>>20)
			}
			b, err := os.ReadFile(src)
			if err != nil {
				return err
			}
			if err := os.WriteFile(dst, b, info.Mode().Perm()); err != nil {
				return err
			}
		}
	}
	return nil
}

// fixPatch stages everything in dir and returns the change from base as a
// binary patch, with its summary.
func fixPatch(git, dir, base string) (patch []byte, stat string, err error) {
	if _, err := gitIn(git, dir, nil, "add", "-A"); err != nil {
		return nil, "", err
	}
	patch, err = exec.Command(git, "-C", dir, "diff", "--cached", "--binary", base).Output()
	if err != nil {
		return nil, "", errors.New("git diff failed in the fix worktree")
	}
	stat, _ = gitIn(git, dir, nil, "diff", "--cached", "--stat", base)
	return patch, stat, nil
}

func fixPrompt(command string, failure testResult) string {
	out := failure.output
	if len(out) > watchPromptTail {
		out = "…" + strings.ToValidUTF8(out[len(out)-watchPromptTail:], "")
	}
	what := "fails"
	if failure.err != "" {
		what = "fails (" + failure.err + ")"
	}
	return "The test command `" + command + "` " + what + " in this project. Fix the code so the tests pass.\n\n" +
		"Its output (the end):\n```\n" + out + "\n```\n\n" +
		"Make the smallest change that fixes the cause. Do not delete, skip or weaken tests unless a test itself is plainly wrong; " +
		"say so if you change one. When you can run commands, run the test command to confirm. End with one or two sentences on what was wrong."
}

// fix copies the working copy aside, has a headless run fix the failure
// there, checks the fix with the test command and saves it as a patch.
func (w *watcher) fix(ctx context.Context) {
	failure := *w.failure
	w.fixedFor = w.failedAt
	w.fixes++
	w.say("Fixing on the side in %s-watch (your files are not touched)…", tui.Safe(w.root))
	dir, base, err := fixTree(w.git, w.root)
	if err != nil {
		w.say("Could not prepare the fix worktree: %s", tui.Safe(err.Error()))
		return
	}
	r := parallelResult{Worktree: dir}
	runHeadless(ctx, w.self, &r, fixPrompt(w.opts.test, failure), w.opts.passthrough)
	if ctx.Err() != nil {
		return
	}
	if r.Status != "completed" {
		msg := r.Status
		if r.Error != "" {
			msg += ": " + r.Error
		}
		w.say("The fix run did not complete (%s).", tui.Safe(firstLine(msg, 200)))
		if r.Status == "needs_permission" {
			w.say("Allow what it needs after --, for example: goh watch -- --allow shell")
		}
		return
	}
	patch, stat, err := fixPatch(w.git, dir, base)
	if err != nil {
		w.say("%s", tui.Safe(err.Error()))
		return
	}
	if len(patch) == 0 {
		w.say("The fix run changed nothing. Its reply: %s", tui.Safe(firstLine(r.Reply, 200)))
		return
	}
	check := runTests(ctx, dir, w.opts.test, w.opts.timeout)
	if ctx.Err() != nil {
		return
	}
	file, err := w.savePatch(patch)
	if err != nil {
		w.say("Could not save the patch: %s", tui.Safe(err.Error()))
		return
	}
	w.patch = file
	verdict := "✓ the tests pass with it"
	if !check.passed {
		verdict = "✗ the tests still fail with it"
	}
	w.say("Fix ready (%ds, %s): %s", r.Seconds, verdict, tui.Safe(firstLine(r.Reply, 200)))
	fmt.Fprintln(w.out, indentTail(stat, 12))
	if !check.passed {
		fmt.Fprintln(w.out, indentTail(check.output, 8))
	}
	w.say("Patch: %s", tui.Safe(file))
	w.say("Type a to apply it, or later: git apply %s", tui.Safe(file))
}

// savePatch writes patch under the repository's .git/goh-watch.
func (w *watcher) savePatch(patch []byte) (string, error) {
	common, err := gitIn(w.git, w.root, nil, "rev-parse", "--git-common-dir")
	if err != nil {
		return "", err
	}
	if !filepath.IsAbs(common) {
		common = filepath.Join(w.root, common)
	}
	dir := filepath.Join(common, "goh-watch")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	file := filepath.Join(dir, "fix-"+time.Now().Format("20060102-150405")+".patch")
	return file, os.WriteFile(file, patch, 0o600)
}

// apply applies the latest fix to the working copy when it still applies
// cleanly.
func (w *watcher) apply() bool {
	if w.patch == "" {
		w.say("No fix to apply.")
		return false
	}
	if _, err := gitIn(w.git, w.root, nil, "apply", "--check", "--binary", w.patch); err != nil {
		w.say("The fix no longer applies cleanly (files changed since). Try: git apply --3way %s", tui.Safe(w.patch))
		return false
	}
	if _, err := gitIn(w.git, w.root, nil, "apply", "--binary", w.patch); err != nil {
		w.say("Applying failed: %s", tui.Safe(err.Error()))
		return false
	}
	w.say("Applied %s; the tests run again when files settle.", tui.Safe(filepath.Base(w.patch)))
	return true
}
