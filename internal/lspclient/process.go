package lspclient

import (
	"context"
	"errors"
	"fmt"
	"github.com/Sanidhya14321/goh/internal/platform"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/Sanidhya14321/goh/internal/core"
)

type process struct {
	cmd  *exec.Cmd
	rpc  *rpcClient
	done chan struct{}
	once sync.Once
	err  error
}

func startProcess(config core.LSPConfig, project string) (*process, error) {
	cmd := exec.Command(config.Command, config.Args...)
	cmd.Dir = project
	platform.NewGroup(cmd)
	cmd.WaitDelay = time.Second
	env := map[string]string{}
	for _, entry := range os.Environ() {
		name, value, _ := strings.Cut(entry, "=")
		env[name] = value
	}
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
	cmd.Stderr = io.Discard
	if err := cmd.Start(); err != nil {
		childIn.Close()
		parentIn.Close()
		parentOut.Close()
		childOut.Close()
		return nil, err
	}
	childIn.Close()
	childOut.Close()
	process := &process{cmd: cmd, done: make(chan struct{})}
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
