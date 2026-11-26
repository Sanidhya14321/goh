package tools

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/Sanidhya14321/goh/internal/platform"
	"github.com/Sanidhya14321/goh/internal/sandbox"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const maxArtifact = 1 << 20

// MaxArtifact is the most a retained artifact keeps.
const MaxArtifact = maxArtifact

// captureWriter drains every byte, retaining only bounded output. The file is
// private and contains a combined stdout/stderr stream in observed write order.
type captureWriter struct {
	mu              sync.Mutex
	preview         strings.Builder
	tail            []byte // the latest previewTail bytes once preview is full
	file            *os.File
	total, retained int64
	err             error
}

// The inline preview keeps the start and the end of long output: test
// summaries, compiler errors and stack traces come last, so a head-only
// preview hides exactly the lines that explain a failure.
const (
	previewHead = MaxInline / 16
	previewTail = MaxInline / 16
)

func (w *captureWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.total += int64(len(p))
	n := previewHead - w.preview.Len()
	if n > len(p) {
		n = len(p)
	}
	if n > 0 {
		w.preview.Write(p[:n])
	}
	if rest := p[max(n, 0):]; len(rest) > 0 {
		w.tail = append(w.tail, rest...)
		if over := len(w.tail) - previewTail; over > 0 {
			w.tail = append(w.tail[:0], w.tail[over:]...)
		}
	}
	n = maxArtifact - int(w.retained)
	if n > len(p) {
		n = len(p)
	}
	if n > 0 && w.err == nil {
		written, err := w.file.Write(p[:n])
		w.retained += int64(written)
		w.err = err
	}
	return len(p), nil
}

// text is the preview: all output when it fits, else its start and end
// around a marker naming the omitted size and where the rest is kept.
func (w *captureWriter) text(artifactID string) string {
	head := strings.ToValidUTF8(w.preview.String(), "�")
	if len(w.tail) == 0 {
		return head
	}
	tail := w.tail
	omitted := w.total - int64(w.preview.Len()) - int64(len(tail))
	if omitted <= 0 {
		return strings.ToValidUTF8(w.preview.String()+string(tail), "�")
	}
	// Start the tail on a whole line (and a whole character) when possible.
	if i := bytes.IndexByte(tail, '\n'); i >= 0 && i < len(tail)-1 {
		omitted += int64(i + 1)
		tail = tail[i+1:]
	}
	return head + fmt.Sprintf("\n[… %d bytes omitted; read_artifact %s has the full output …]\n", omitted, artifactID) + strings.ToValidUTF8(string(tail), "�")
}

func (e *Engine) shell(ctx context.Context, a args) Result {
	a.Cwd, _ = e.clean(a.Cwd, true)
	ctx, cancel := context.WithTimeout(ctx, time.Duration(a.Timeout)*time.Second)
	defer cancel()
	if err := os.MkdirAll(e.artifacts, 0700); err != nil {
		return failure(err)
	}
	if err := os.Chmod(e.artifacts, 0700); err != nil {
		return failure(err)
	}
	artifact, err := os.CreateTemp(e.artifacts, "shell-*.log")
	if err != nil {
		return failure(err)
	}
	defer artifact.Close()
	if err = artifact.Chmod(0600); err != nil {
		return failure(err)
	}
	output := &captureWriter{file: artifact}
	stdoutR, stdoutW, err := os.Pipe()
	if err != nil {
		return failure(err)
	}
	defer stdoutR.Close()
	defer stdoutW.Close()
	stderrR, stderrW, err := os.Pipe()
	if err != nil {
		return failure(err)
	}
	defer stderrR.Close()
	defer stderrW.Close()
	cmd := shellCommand(a.Command)
	cmd.Env = platform.ChildEnv()
	cmd.Dir = filepath.Join(e.root, a.Cwd)
	if err := sandbox.Wrap(cmd, e.sandboxPolicy(ctx)); err != nil {
		return failure(err)
	}
	platform.NewGroup(cmd)
	cmd.Stdout = stdoutW
	cmd.Stderr = stderrW
	if err := ctx.Err(); err != nil {
		return failure(err)
	}
	if err = cmd.Start(); err != nil {
		return failure(err)
	}
	stdoutW.Close()
	stderrW.Close()
	drain := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); _, _ = io.Copy(output, stdoutR) }()
	go func() { defer wg.Done(); _, _ = io.Copy(output, stderrR) }()
	go func() { wg.Wait(); close(drain) }()
	waited := make(chan error, 1)
	go func() { waited <- cmd.Wait() }()
	var waitErr error
	interrupted := false
	select {
	case waitErr = <-waited:
	case <-ctx.Done():
		interrupted = true
	}
	// Always settle descendants that remain in the owned group, including after
	// a successful parent exit. Deliberately detached processes are outside scope.
	pid := cmd.Process.Pid
	// Processes the command left running in its group (`server &`, nohup)
	// are stopped below; the result tells the model how to keep one.
	leftBehind := !interrupted && !platform.GroupExited(pid)
	termErr := platform.TerminateGroup(pid)
	cleanupIncomplete := termErr != nil
	timer := time.NewTimer(2 * time.Second)
	defer timer.Stop()
	processDone := !interrupted
	drainDone := false
	for !processDone || !drainDone {
		select {
		case err := <-waited:
			waitErr = err
			processDone = true
			waited = nil
		case <-drain:
			drainDone = true
			drain = nil
		case <-timer.C:
			if err := platform.KillGroup(pid); err != nil {
				cleanupIncomplete = true
			}
			stdoutR.Close()
			stderrR.Close()
			if !processDone {
				select {
				case waitErr = <-waited:
					processDone = true
				case <-time.After(time.Second):
					cleanupIncomplete = true
				}
			}
			if !drainDone {
				select {
				case <-drain:
					drainDone = true
				case <-time.After(time.Second):
					cleanupIncomplete = true
				}
			}
			goto settled
		}
	}
	// Quiet descendants can close their pipes while remaining in the group.
	// Give those processes the same two-second termination window.
	if !platform.GroupExited(pid) {
		<-timer.C
	}
	if err := platform.KillGroup(pid); err != nil {
		cleanupIncomplete = true
	}
settled:
	output.mu.Lock()
	defer output.mu.Unlock()
	if err := artifact.Sync(); err != nil {
		cleanupIncomplete = true
	}
	if output.err != nil {
		cleanupIncomplete = true
	}
	code := 0
	if waitErr != nil {
		code = -1
		var exit *exec.ExitError
		if errors.As(waitErr, &exit) {
			code = exit.ExitCode()
		}
	}
	state := "completed"
	if interrupted {
		state = "cancelled"
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			state = "timed_out"
		}
	}
	if cleanupIncomplete {
		state = "cleanup_incomplete"
	}
	value := map[string]any{"exit_code": code, "status": state, "output": output.text(filepath.Base(artifact.Name())), "artifact": artifact.Name(), "artifact_id": filepath.Base(artifact.Name()), "output_bytes": output.total, "retained_bytes": output.retained, "truncated": output.total > output.retained, "preview_truncated": output.total > int64(output.preview.Len()+len(output.tail))}
	if waitErr != nil {
		value["error"] = waitErr.Error()
	}
	if code == 127 {
		if hint := missingProgramHint(output.text("")); hint != "" {
			value["hint"] = hint
		}
	}
	if interrupted {
		value["effects"] = "Execution interrupted; effects may already have occurred and were not rolled back."
	}
	if state == "timed_out" {
		value["hint"] = "stopped at its timeout; for a longer command raise timeout_seconds (up to 600) or start it with job_start and check it with job_output"
	}
	if leftBehind && state == "completed" {
		value["note"] = "processes this command left running in the background were stopped when it finished; start a server or other service that must keep running with job_start"
	}
	if termErr != nil {
		value["cleanup_error"] = fmt.Sprint(termErr)
	}
	res := result(value)
	res.Error = res.Error || waitErr != nil || interrupted || cleanupIncomplete
	return res
}

// usualSubstitutes are programs commonly missing under their unversioned
// name (macOS and many Linux systems have only python3).
var usualSubstitutes = map[string]string{"python": "python3", "pip": "pip3"}

// missingProgramHint names the installed substitute when a command failed
// because "python" or "pip" does not exist, so the model does not have to
// rediscover it (benchmark runs lost a call to this on every Python task).
func missingProgramHint(output string) string {
	for name, alt := range usualSubstitutes {
		if !strings.Contains(output, name+": command not found") && !strings.Contains(output, name+": not found") {
			continue
		}
		if _, err := exec.LookPath(name); err == nil {
			continue
		}
		if _, err := exec.LookPath(alt); err == nil {
			return fmt.Sprintf("%s is not installed here; use %s", name, alt)
		}
	}
	return ""
}
