package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"example.com/independent-harness/internal/core"
	"example.com/independent-harness/internal/localapi"
)

func serve(args []string) error {
	flags := flag.NewFlagSet("serve", flag.ContinueOnError)
	address := flags.String("listen", "127.0.0.1:4096", "numeric loopback address and port")
	project := flags.String("project", ".", "fixed project directory")
	configPath := flags.String("config", "", "explicit user JSON configuration")
	sessionDir := flags.String("session-dir", "", "private session storage")
	profile := flags.String("profile", "", "configured model profile")
	agent := flags.String("agent", "", "configured agent")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if flags.NArg() != 0 || !localapi.IsLoopbackHost(*address) {
		return errors.New("serve requires a numeric loopback listen address and no positional arguments")
	}
	token := os.Getenv("HARNESS_SERVER_TOKEN")
	// Keep the server credential out of tool subprocess environments.
	if err := os.Unsetenv("HARNESS_SERVER_TOKEN"); err != nil {
		return err
	}
	config, err := core.LoadConfig(*configPath)
	if err != nil {
		return err
	}
	config, err = config.Resolve(*profile, *agent)
	if err != nil {
		return err
	}
	if *sessionDir != "" {
		config.SessionDir = *sessionDir
	}
	handler, err := localapi.New(config, *project, token)
	if err != nil {
		return err
	}

	listener, err := net.Listen("tcp", *address)
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	server := &http.Server{
		Handler:           handler,
		BaseContext:       func(net.Listener) context.Context { return ctx },
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       30 * time.Second,
		MaxHeaderBytes:    16 << 10,
	}
	done := make(chan error, 1)
	go func() { done <- server.Serve(listener) }()
	fmt.Fprintln(os.Stderr, "Harness API listening on", listener.Addr())
	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		shutdown, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdown); err != nil {
			return errors.Join(err, server.Close())
		}
		err := <-done
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}
