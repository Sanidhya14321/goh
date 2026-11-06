//go:build darwin || linux

package tools

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
)

const maxArtifact = 1 << 20

// captureWriter drains every byte, retaining only bounded output. The file is
// private and contains a combined stdout/stderr stream in observed write order.
type captureWriter struct {
	mu              sync.Mutex
	preview         strings.Builder
	file            *os.File
	total, retained int64
	err             error
}

func (w *captureWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.total += int64(len(p))
	n := MaxInline/8 - w.preview.Len()
	if n > len(p) {
		n = len(p)
	}
	if n > 0 {
		w.preview.Write(p[:n])
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
func (e *Engine) shell(ctx context.Context, a args) Result {
	a.Cwd, _ = clean(a.Cwd, true)
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
	cmd := exec.Command("/bin/sh", "-c", a.Command)
	cmd.Dir = filepath.Join(e.root, a.Cwd)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
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
	signal := func(sig syscall.Signal) error {
		err := syscall.Kill(-cmd.Process.Pid, sig)
		if err == syscall.ESRCH {
			return nil
		}
		return err
	}
	termErr := signal(syscall.SIGTERM)
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
			if err := signal(syscall.SIGKILL); err != nil {
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
	if syscall.Kill(-cmd.Process.Pid, 0) == nil {
		<-timer.C
	}
	if err := signal(syscall.SIGKILL); err != nil {
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
	value := map[string]any{"exit_code": code, "status": state, "output": strings.ToValidUTF8(output.preview.String(), "�"), "artifact": artifact.Name(), "artifact_id": filepath.Base(artifact.Name()), "output_bytes": output.total, "retained_bytes": output.retained, "truncated": output.total > output.retained, "preview_truncated": output.total > int64(output.preview.Len())}
	if waitErr != nil {
		value["error"] = waitErr.Error()
	}
	if interrupted {
		value["effects"] = "Execution interrupted; effects may already have occurred and were not rolled back."
	}
	if termErr != nil {
		value["cleanup_error"] = fmt.Sprint(termErr)
	}
	res := result(value)
	res.Error = res.Error || waitErr != nil || interrupted || cleanupIncomplete
	return res
}
