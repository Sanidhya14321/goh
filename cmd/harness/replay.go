package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/Sanidhya14321/goh/internal/session"
)

const maxReplayTurns = 50

// runReplay implements `goh replay SESSION --models A,B`: the session's user
// prompts are sent again, in order, to each model, each in a fresh worktree
// (<repo>-NAME-N on goh/NAME-N, from HEAD or --from REF). The first prompt
// starts a headless run and the rest continue that run's session, so every
// model sees the same conversation from its own side. The summary is the
// `goh parallel` one: status, changed files, tokens and time per model.
func runReplay(args []string, stdout io.Writer) error {
	var passthrough []string
	for i, a := range args {
		if a == "--" {
			args, passthrough = args[:i], args[i+1:]
			break
		}
	}
	usage := "usage: goh replay SESSION --models A,B [--from REF] [--turns N] [--jobs N] [--name PREFIX] [--commit] [--format text|json] [-- RUN FLAGS]"
	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		if len(args) > 0 && (args[0] == "--help" || args[0] == "-h") {
			fmt.Fprintln(stdout, usage)
			return nil
		}
		return errors.New(usage)
	}
	id := args[0]
	f := flag.NewFlagSet("replay", flag.ContinueOnError)
	modelList := f.String("models", "", "comma-separated models to replay the session with (required)")
	from := f.String("from", "", "commit the worktrees start from (default HEAD)")
	turns := f.Int("turns", 0, "replay only the first N prompts (default all)")
	jobs := f.Int("jobs", 0, "models run at once (default: up to 4)")
	name := f.String("name", "", "worktree and branch name prefix (default r-MMDD-HHMM)")
	commit := f.Bool("commit", false, "commit each worktree's changes on its branch")
	format := f.String("format", "text", "summary format: text or json")
	dir := f.String("session-dir", configuredSessionDir(), "private session storage")
	if err := f.Parse(args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if f.NArg() > 0 {
		return errors.New(usage)
	}
	var models []string
	for _, m := range strings.Split(*modelList, ",") {
		if m = strings.TrimSpace(m); m != "" {
			models = append(models, m)
		}
	}
	if len(models) == 0 || len(models) > maxParallelTasks {
		return fmt.Errorf("%s; 1-%d models", usage, maxParallelTasks)
	}
	for _, a := range passthrough {
		for _, set := range []string{"--project", "--format", "--prompt", "--model", "--session", "--continue", "--fork"} {
			if a == set || strings.HasPrefix(a, set+"=") {
				return fmt.Errorf("%s is set by goh replay for each run", a)
			}
		}
	}
	if *format != "text" && *format != "json" {
		return errors.New("--format must be text or json")
	}
	if *jobs == 0 {
		*jobs = min(4, len(models))
	}
	if *jobs < 1 || *jobs > maxParallelJobs {
		return fmt.Errorf("--jobs must be 1-%d", maxParallelJobs)
	}
	if *turns < 0 {
		return errors.New("--turns must be positive")
	}
	if *name == "" {
		*name = "r-" + time.Now().Format("0102-1504")
	}
	if !worktreeName.MatchString(*name) || strings.Contains(*name, "..") || len(*name) > 56 {
		return errors.New("--name: letters, digits, . _ - (up to 56)")
	}

	store, err := session.OpenReadOnly(*dir, id, "")
	if err != nil {
		return fmt.Errorf("session %s: %w", id, err)
	}
	prompts, project := store.Prompts(), store.Project
	store.Close()
	if *turns > 0 && *turns < len(prompts) {
		prompts = prompts[:*turns]
	}
	if len(prompts) == 0 {
		return errors.New("the session has no prompts to replay")
	}
	if len(prompts) > maxReplayTurns {
		return fmt.Errorf("the session has %d prompts; replay at most %d (--turns)", len(prompts), maxReplayTurns)
	}

	git, err := exec.LookPath("git")
	if err != nil {
		return errors.New("goh replay needs git on PATH")
	}
	// The worktrees come from the session's own repository, wherever goh
	// replay runs; a session started in a subfolder is refused rather than
	// replayed at the wrong level.
	root, err := gitTop(git, project)
	if err != nil {
		return fmt.Errorf("session %s: %s is not in a Git repository", id, project)
	}
	if !sameDir(root, project) {
		return fmt.Errorf("session %s ran in %s, a subfolder of %s; replay needs a session started at the repository root", id, project, root)
	}
	if err := os.Chdir(root); err != nil { // worktree helpers run git here
		return err
	}
	base := "HEAD"
	if *from != "" {
		base = *from
	}
	rev, err := exec.Command(git, "-C", root, "rev-parse", "--verify", "--quiet", base+"^{commit}").Output()
	if err != nil {
		return fmt.Errorf("--from %s is not a commit", base)
	}
	start := strings.TrimSpace(string(rev))
	self, err := os.Executable()
	if err != nil {
		return err
	}

	results := make([]parallelResult, len(models))
	for i, model := range models {
		path, branch, err := ensureWorktree(git, root, fmt.Sprintf("%s-%d", *name, i+1), io.Discard)
		if err != nil {
			return fmt.Errorf("%s: %w", model, err)
		}
		// A worktree reused from an earlier replay must be clean before it is moved.
		if dirty, _ := exec.Command(git, "-C", path, "status", "--porcelain").Output(); len(dirty) > 0 {
			return fmt.Errorf("%s has uncommitted changes; choose another --name", path)
		}
		if out, err := exec.Command(git, "-C", path, "reset", "-q", "--hard", start).CombinedOutput(); err != nil {
			return fmt.Errorf("%s: %s", path, firstLine(string(out), 160))
		}
		results[i] = parallelResult{Task: replayLabel(prompts), Model: model, Worktree: path, Branch: branch}
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	fmt.Fprintf(os.Stderr, "Replaying %d prompts of %s with %d models from %s, %d at a time…\n", len(prompts), id, len(models), start[:min(12, len(start))], *jobs)
	var wg sync.WaitGroup
	var mu sync.Mutex
	slots := make(chan struct{}, *jobs)
	for i := range results {
		wg.Add(1)
		go func(r *parallelResult) {
			defer wg.Done()
			select {
			case slots <- struct{}{}:
			case <-ctx.Done():
				r.Status = "cancelled"
				return
			}
			defer func() { <-slots }()
			replayRun(ctx, self, git, r, prompts, passthrough, *commit, func(turn int) {
				mu.Lock()
				fmt.Fprintf(os.Stderr, "[%s] prompt %d/%d: %s\n", r.Model, turn, len(prompts), r.Status)
				mu.Unlock()
			})
		}(&results[i])
	}
	wg.Wait()
	failed := writeParallel(stdout, results, *format)
	if failed > 0 {
		return fmt.Errorf("%d of %d replays did not complete", failed, len(results))
	}
	return nil
}

// replayRun sends prompts one after another to r.Model in r.Worktree,
// continuing the first run's session, and stops at the first prompt that
// does not complete. Tokens and seconds add up over the prompts.
func replayRun(ctx context.Context, self, git string, r *parallelResult, prompts, passthrough []string, commit bool, progress func(int)) {
	tokens, seconds := 0, 0
	for i, prompt := range prompts {
		args := append([]string(nil), passthrough...)
		if i > 0 {
			if r.Session == "" {
				r.Status, r.Error = "failed", "the first run reported no session to continue"
				break
			}
			args = append(args, "--session", r.Session)
		}
		r.Tokens, r.Status, r.Error = 0, "", ""
		runHeadless(ctx, self, r, prompt, args)
		tokens += r.Tokens
		seconds += r.Seconds
		progress(i + 1)
		if r.Status != "completed" {
			if len(prompts) > 1 {
				r.Error = strings.TrimSpace(fmt.Sprintf("prompt %d: %s", i+1, r.Error))
			}
			break
		}
	}
	r.Tokens, r.Seconds = tokens, seconds
	finishWorktree(ctx, git, r, commit, "goh replay: "+r.Model, "Replayed: "+r.Task)
}

func replayLabel(prompts []string) string {
	label := firstLine(prompts[0], 80)
	if n := len(prompts) - 1; n == 1 {
		label += " (+1 more prompt)"
	} else if n > 1 {
		label += fmt.Sprintf(" (+%d more prompts)", n)
	}
	return label
}
