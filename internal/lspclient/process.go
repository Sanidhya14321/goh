package lspclient

import (
	"context"
	"errors"
	"fmt"
	"github.com/Sanidhya14321/goh/internal/platform"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/Sanidhya14321/goh/internal/core"
)

type process struct {
	cmd    *exec.Cmd
	rpc    *rpcClient
	done   chan struct{}
	once   sync.Once
	err    error
	stderr *tailBuffer // the end of the server's error output, for start failures
}

// tailBuffer keeps the last bytes written to it.
type tailBuffer struct {
	mu  sync.Mutex
	buf []byte
}

const stderrTail = 2048

// escapeSequence matches terminal colour and cursor sequences (CSI and OSC).
var escapeSequence = regexp.MustCompile(`\x1b(\[[0-9;?]*[ -/]*[@-~]|\][^\x07\x1b]*(\x07|\x1b\\)?)`)

func (t *tailBuffer) Write(b []byte) (int, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.buf = append(t.buf, b...)
	if len(t.buf) > stderrTail {
		t.buf = append([]byte(nil), t.buf[len(t.buf)-stderrTail:]...)
	}
	return len(b), nil
}

// lastLines is the last two non-empty lines written (the cause and any
// hint after it), without terminal controls, at most 300 bytes.
func (t *tailBuffer) lastLines() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	var kept []string
	text := escapeSequence.ReplaceAllString(string(t.buf), "")
	lines := strings.Split(strings.ReplaceAll(text, "\r", "\n"), "\n")
	for i := len(lines) - 1; i >= 0 && len(kept) < 2; i-- {
		line := strings.TrimSpace(strings.Map(func(r rune) rune {
			if r < 0x20 || r == 0x7f {
				return -1
			}
			return r
		}, strings.ToValidUTF8(lines[i], "")))
		if line != "" {
			kept = append([]string{line}, kept...)
		}
	}
	text = strings.Join(kept, " / ")
	if len(text) > 300 {
		text = strings.ToValidUTF8(text[:300], "") + "…"
	}
	return text
}

// startError adds the server's last error lines to a failure to start, so
// "connection closed" says why (a missing component, a bad flag).
func (p *process) startError(err error) error {
	if p == nil || p.stderr == nil {
		return err
	}
	select {
	case <-p.done:
	case <-time.After(200 * time.Millisecond):
	}
	if line := p.stderr.lastLines(); line != "" {
		return fmt.Errorf("%w; the server said: %s", err, line)
	}
	return err
}

func startProcess(config core.LSPConfig, project string) (*process, error) {
	cmd := exec.Command(config.Command, config.Args...)
	cmd.Dir = project
	platform.NewGroup(cmd)
	cmd.WaitDelay = time.Second
	env := platform.ChildEnvMap() // secrets withheld; config.Env adds explicit values
	for name, value := range config.Env {
		if name == "" || strings.ContainsAny(name, "=\x00") || strings.ContainsRune(value, '\x00') {
			return nil, errors.New("invalid LSP environment")
		}
		env[name] = value
	}
	for name, value := range env {
		cmd.Env = append(cmd.Env, name+"="+value)
	}
	childIn, parentIn, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	parentOut, childOut, err := os.Pipe()
	if err != nil {
		childIn.Close()
		parentIn.Close()
		return nil, err
	}
	cmd.Stdin = childIn
	cmd.Stdout = childOut
	stderr := &tailBuffer{}
	cmd.Stderr = stderr
	if err := cmd.Start(); err != nil {
		childIn.Close()
		parentIn.Close()
		parentOut.Close()
		childOut.Close()
		return nil, err
	}
	childIn.Close()
	childOut.Close()
	process := &process{cmd: cmd, done: make(chan struct{}), stderr: stderr}
	process.rpc = newRPC(parentOut, parentIn, config.Initialization)
	go func() { cmd.Wait(); close(process.done) }()
	return process, nil
}
func (p *process) stop() error {
	p.once.Do(func() {
		// Graceful protocol shutdown has a strict deadline; blocked pipes are then
		// closed before group termination, including server-owned descendants.
		p.rpc.settleServerRequests(200 * time.Millisecond)
		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		_, _ = p.rpc.call(ctx, "shutdown", nil)
		cancel()
		exited := make(chan struct{})
		go func() { p.rpc.notify("exit", nil); close(exited) }()
		select {
		case <-exited:
		case <-time.After(100 * time.Millisecond):
		}
		p.rpc.fail(errors.New("LSP manager closed"))
		pgid := p.cmd.Process.Pid
		if err := platform.StopGroup(pgid, 2*time.Second); err != nil {
			p.err = fmt.Errorf("LSP cleanup_incomplete: %w", err)
		}
		select {
		case <-p.done:
		case <-time.After(time.Second):
			p.err = errors.Join(p.err, errors.New("LSP cleanup_incomplete: server did not settle"))
		}
	})
	return p.err
}
