package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/Sanidhya14321/goh/internal/tui"
)

const (
	maxParallelTasks = 16
	maxParallelJobs  = 8
)

// parallelResult is one task's outcome.
type parallelResult struct {
	Task     string `json:"task"`
	Model    string `json:"model,omitempty"`
	Tokens   int    `json:"tokens,omitempty"`
	Worktree string `json:"worktree"`
	Branch   string `json:"branch"`
	Session  string `json:"session_id,omitempty"`
	Status   string `json:"status"`
	Error    string `json:"error,omitempty"`
	Changed  int    `json:"changed_files"`
	Commit   string `json:"commit,omitempty"`
	Reply    string `json:"reply,omitempty"`
	Seconds  int    `json:"seconds"`
}

// runParallel implements `goh parallel`: each task runs headless in its own
// Git worktree (<repo>-NAME-N on branch goh/NAME-N, from HEAD), several at a
// time, and a summary lists what each changed. Arguments after -- go to every
// `goh run` unchanged (for example --access edits or --allow shell).
func runParallel(args []string, stdout io.Writer) error {
	var passthrough []string
	for i, a := range args {
		if a == "--" {
			args, passthrough = args[:i], args[i+1:]
			break
		}
	}
	f := flag.NewFlagSet("parallel", flag.ContinueOnError)
	jobs := f.Int("jobs", 0, "tasks run at once (default: up to 4)")
	file := f.String("file", "", "read tasks from a file, one per line (# comments and blank lines skipped)")
	name := f.String("name", "", "worktree and branch name prefix (default p-MMDD-HHMM)")
	commit := f.Bool("commit", false, "commit each worktree's changes on its branch")
	format := f.String("format", "text", "summary format: text or json")
	modelList := f.String("models", "", "comma-separated models: run every task once with each, to compare them")
	if err := f.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	tasks := append([]string(nil), f.Args()...)
	if *file != "" {
		b, err := os.ReadFile(*file)
		if err != nil {
			return err
		}
		for _, line := range strings.Split(string(b), "\n") {
			if line = strings.TrimSpace(line); line != "" && !strings.HasPrefix(line, "#") {
				tasks = append(tasks, line)
			}
		}
	}
	var models []string
	for _, m := range strings.Split(*modelList, ",") {
		if m = strings.TrimSpace(m); m != "" {
			models = append(models, m)
		}
	}
	if len(models) > 0 {
		for _, a := range passthrough {
			if a == "--model" || strings.HasPrefix(a, "--model=") {
				return errors.New("--model after -- conflicts with --models")
			}
		}
	}
	runs := len(tasks) * max(1, len(models))
	if len(tasks) == 0 || runs > maxParallelTasks {
		return fmt.Errorf("usage: goh parallel [--jobs N] [--models A,B] [--name PREFIX] [--commit] [--file TASKS] TASK... [-- RUN FLAGS]; 1-%d runs (tasks × models)", maxParallelTasks)
	}
	if *format != "text" && *format != "json" {
		return errors.New("--format must be text or json")
	}
	if *jobs == 0 {
		*jobs = min(4, runs)
	}
	if *jobs < 1 || *jobs > maxParallelJobs {
		return fmt.Errorf("--jobs must be 1-%d", maxParallelJobs)
	}
	for _, a := range passthrough {
		if a == "--project" || strings.HasPrefix(a, "--project=") || a == "--format" || strings.HasPrefix(a, "--format=") || a == "--prompt" || strings.HasPrefix(a, "--prompt=") {
			return fmt.Errorf("%s is set by goh parallel for each task", a)
		}
	}
	if *name == "" {
		*name = "p-" + time.Now().Format("0102-1504")
	}
	if !worktreeName.MatchString(*name) || strings.Contains(*name, "..") || len(*name) > 56 {
		return errors.New("--name: letters, digits, . _ - (up to 56)")
	}
	git, err := exec.LookPath("git")
	if err != nil {
		return errors.New("goh parallel needs git on PATH")
	}
	out, err := exec.Command(git, "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return errors.New("goh parallel must run inside a Git repository")
	}
	root := filepath.Clean(strings.TrimSpace(string(out)))
	self, err := os.Executable()
	if err != nil {
		return err
	}
	if dirty, _ := exec.Command(git, "-C", root, "status", "--porcelain", "--untracked-files=no").Output(); len(dirty) > 0 {
		fmt.Fprintln(os.Stderr, "Note: the worktrees start from the last commit; uncommitted changes here are not in them.")
	}
	results := make([]parallelResult, 0, runs)
	// Worktrees are added one at a time (git locks its metadata).
	for _, task := range tasks {
		for _, model := range append([]string(nil), models...) {
			results = append(results, parallelResult{Task: task, Model: model})
		}
		if len(models) == 0 {
			results = append(results, parallelResult{Task: task})
		}
	}
	for i := range results {
		path, branch, err := ensureWorktree(git, root, fmt.Sprintf("%s-%d", *name, i+1), io.Discard)
		if err != nil {
			return fmt.Errorf("run %d: %w", i+1, err)
		}
		results[i].Worktree, results[i].Branch = path, branch
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	fmt.Fprintf(os.Stderr, "Running %d tasks, %d at a time…\n", len(results), *jobs)
	var wg sync.WaitGroup
	var mu sync.Mutex
	slots := make(chan struct{}, *jobs)
	for i := range results {
		wg.Add(1)
		go func(r *parallelResult, n int) {
			defer wg.Done()
			select {
			case slots <- struct{}{}:
			case <-ctx.Done():
				r.Status = "cancelled"
				return
			}
			defer func() { <-slots }()
			runParallelTask(ctx, self, git, r, passthrough, *commit)
			mu.Lock()
			fmt.Fprintf(os.Stderr, "[%d/%d] %s: %s\n", n+1, len(results), r.Branch, r.Status)
			mu.Unlock()
		}(&results[i], i)
	}
	wg.Wait()
	failed := writeParallel(stdout, results, *format)
	if failed > 0 {
		return fmt.Errorf("%d of %d tasks did not complete", failed, len(results))
	}
	return nil
}

func runParallelTask(ctx context.Context, self, git string, r *parallelResult, passthrough []string, commit bool) {
	runHeadless(ctx, self, r, r.Task, passthrough)
	finishWorktree(ctx, git, r, commit, "goh: "+firstLine(r.Task, 72), "Task: "+r.Task)
}

// finishWorktree counts r's changed files and, with commit, commits them on
// the worktree's branch.
func finishWorktree(ctx context.Context, git string, r *parallelResult, commit bool, subject, body string) {
	status, _ := exec.Command(git, "-C", r.Worktree, "status", "--porcelain").Output()
	for _, line := range strings.Split(strings.TrimSpace(string(status)), "\n") {
		if line != "" {
			r.Changed++
		}
	}
	if commit && r.Changed > 0 && ctx.Err() == nil {
		out, err := exec.Command(git, "-C", r.Worktree, "add", "-A").CombinedOutput()
		if err == nil {
			out, err = exec.Command(git, "-C", r.Worktree, "commit", "-q", "-m", subject, "-m", body).CombinedOutput()
		}
		if err != nil {
			r.Error = strings.TrimSpace("commit failed: " + firstLine(string(out), 160) + " " + r.Error)
		} else if head, err := exec.Command(git, "-C", r.Worktree, "rev-parse", "--short", "HEAD").Output(); err == nil {
			r.Commit = strings.TrimSpace(string(head))
		}
	}
}

// runHeadless runs `goh run` on prompt in r.Worktree and fills r from its
// JSON events: status, error, session, final reply, tokens and duration.
func runHeadless(ctx context.Context, self string, r *parallelResult, prompt string, passthrough []string) {
	start := time.Now()
	defer func() { r.Seconds = int(time.Since(start).Seconds() + 0.5) }()
	args := append([]string{"run", "--project", r.Worktree, "--format", "json"}, passthrough...)
	if r.Model != "" {
		args = append(args, "--model", r.Model)
	}
	args = append(args, "--prompt", prompt)
	cmd := exec.CommandContext(ctx, self, args...)
	cmd.Dir = r.Worktree
	cmd.Cancel = func() error { return cmd.Process.Signal(os.Interrupt) } // goh run settles its effects
	cmd.WaitDelay = 20 * time.Second
	var stderr strings.Builder
	cmd.Stderr = &limitedWriter{w: &stderr, n: 4 << 10}
	pipe, err := cmd.StdoutPipe()
	if err != nil {
		r.Status, r.Error = "failed", err.Error()
		return
	}
	if err := cmd.Start(); err != nil {
		r.Status, r.Error = "failed", err.Error()
		return
	}
	asked := ""
	scanner := bufio.NewScanner(pipe)
	scanner.Buffer(make([]byte, 64<<10), 32<<20)
	for scanner.Scan() {
		var event struct {
			Session string `json:"session_id"`
			Type    string `json:"type"`
			Data    struct {
				Tool    string `json:"tool"`
				Preview string `json:"preview"`
				Status  string `json:"status"`
				Error   string `json:"error"`
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
				Usage *struct {
					Prompt     int `json:"prompt_tokens"`
					Completion int `json:"completion_tokens"`
					Total      int `json:"total_tokens"`
				} `json:"usage"`
			} `json:"data"`
		}
		if json.Unmarshal(scanner.Bytes(), &event) != nil {
			continue
		}
		if r.Session == "" {
			r.Session = event.Session
		}
		switch event.Type {
		case "assistant_completed":
			if c := strings.TrimSpace(event.Data.Message.Content); c != "" {
				r.Reply = c
			}
			if u := event.Data.Usage; u != nil {
				r.Tokens += max(u.Total, u.Prompt+u.Completion)
			}
		case "permission_requested":
			asked = event.Data.Tool + ": " + firstLine(event.Data.Preview, 100)
		case "run_finished":
			r.Status, r.Error = event.Data.Status, event.Data.Error
			if r.Status == "needs_permission" && asked != "" {
				r.Error = "needs approval for " + asked + " (allow it after --, for example --allow " + strings.SplitN(asked, ":", 2)[0] + ")"
			}
		}
	}
	waitErr := cmd.Wait()
	if r.Status == "" {
		r.Status = "failed"
		if ctx.Err() != nil {
			r.Status = "cancelled"
		}
		if r.Error = strings.TrimSpace(stderr.String()); r.Error == "" && waitErr != nil {
			r.Error = waitErr.Error()
		}
	}
}

type limitedWriter struct {
	w io.Writer
	n int
}

func (l *limitedWriter) Write(p []byte) (int, error) {
	if l.n > 0 {
		k := min(len(p), l.n)
		l.w.Write(p[:k])
		l.n -= k
	}
	return len(p), nil
}

func firstLine(s string, n int) string {
	s, _, _ = strings.Cut(strings.TrimSpace(s), "\n")
	if r := []rune(s); len(r) > n {
		return string(r[:n-1]) + "…"
	}
	return s
}

// writeParallel prints the summary and returns how many tasks did not
// complete.
func writeParallel(w io.Writer, results []parallelResult, format string) int {
	failed := 0
	for _, r := range results {
		if r.Status != "completed" {
			failed++
		}
	}
	if format == "json" {
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		enc.Encode(map[string]any{"tasks": results})
		return failed
	}
	for i, r := range results {
		changed := fmt.Sprintf("%d files", r.Changed)
		if r.Changed == 1 {
			changed = "1 file"
		}
		extra := ""
		if r.Model != "" {
			extra += "  " + tui.Safe(r.Model)
		}
		if r.Tokens > 0 {
			extra += fmt.Sprintf("  %d tokens", r.Tokens)
		}
		fmt.Fprintf(w, "%d. %s  %s  %s  %ds%s\n", i+1, tui.Safe(r.Status), tui.Safe(r.Branch), changed, r.Seconds, extra)
		fmt.Fprintf(w, "   task: %s\n", tui.Safe(firstLine(r.Task, 100)))
		if r.Reply != "" {
			fmt.Fprintf(w, "   reply: %s\n", tui.Safe(firstLine(r.Reply, 100)))
		}
		if r.Error != "" {
			fmt.Fprintf(w, "   error: %s\n", tui.Safe(firstLine(r.Error, 160)))
		}
		if r.Commit != "" {
			fmt.Fprintf(w, "   commit: %s\n", r.Commit)
		}
		fmt.Fprintf(w, "   in: %s", tui.Safe(r.Worktree))
		if r.Session != "" {
			fmt.Fprintf(w, "  (continue: cd there; goh chat --session %s)", tui.Safe(r.Session))
		}
		fmt.Fprintln(w)
	}
	return failed
}
