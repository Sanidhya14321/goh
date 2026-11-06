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
	config, err = config.Resolve(*profile, *agent)
	if err != nil {
		return err
	}
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

// Inherited pipe descriptors need to be pollable for cancellable reads and
// write deadlines. Keep regular file redirection available for transcripts.
func acpStream(file *os.File) (*os.File, func(), error) {
	info, err := file.Stat()
	if err != nil {
		return nil, nil, err
	}
	if info.Mode()&(os.ModeNamedPipe|os.ModeSocket) == 0 {
		return file, func() {}, nil
	}
	fd, err := syscall.Dup(int(file.Fd()))
	if err != nil {
		return nil, nil, err
	}
	syscall.CloseOnExec(fd)
	flags, _, errno := syscall.Syscall(syscall.SYS_FCNTL, uintptr(fd), syscall.F_GETFL, 0)
	if errno != 0 {
		syscall.Close(fd)
		return nil, nil, errno
	}
	if err := syscall.SetNonblock(fd, true); err != nil {
		syscall.Close(fd)
		return nil, nil, err
	}
	stream := os.NewFile(uintptr(fd), file.Name())
	return stream, func() {
		_ = stream.Close()
		if flags&syscall.O_NONBLOCK == 0 {
			_ = syscall.SetNonblock(int(file.Fd()), false)
		}
	}, nil
}
