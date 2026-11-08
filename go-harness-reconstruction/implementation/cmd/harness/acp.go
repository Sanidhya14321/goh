package main

import (
	"context"
	"errors"
	"flag"
	"os"
	"os/signal"
	"syscall"

	"example.com/independent-harness/internal/acp"
	"example.com/independent-harness/internal/core"
)

func runACP(args []string) error {
	flags := flag.NewFlagSet("acp", flag.ContinueOnError)
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
	if flags.NArg() != 0 {
		return errors.New("acp takes no positional arguments")
	}
	config, err := core.LoadConfig(*configPath)
	if err != nil {
		return err
	}
	// Editors start the adapter in the workspace; agent files load from there.
	if dir, err := os.Getwd(); err == nil {
		config, _ = config.WithProjectAgents(dir)
	}
	config, err = config.Resolve(*profile, *agent)
	if err != nil {
		return err
	}
	if *configPath == "" && *profile == "" {
		config = withChatGPTDefault(config)
	}
	config = config.WithDetectedLSP(nil)
	if *sessionDir != "" {
		config.SessionDir = *sessionDir
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	input, closeInput, err := acpStream(os.Stdin)
	if err != nil {
		return err
	}
	defer closeInput()
	output, closeOutput, err := acpStream(os.Stdout)
	if err != nil {
		return err
	}
	defer closeOutput()
	return acp.Serve(ctx, input, output, config, *project, version)
}
