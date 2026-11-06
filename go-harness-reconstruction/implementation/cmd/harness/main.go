package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	goruntime "runtime"
	"strings"
	"syscall"

	"example.com/independent-harness/internal/core"
	"example.com/independent-harness/internal/formatters"
	"example.com/independent-harness/internal/mcpclient"
	agent "example.com/independent-harness/internal/runtime"
	"example.com/independent-harness/internal/session"
	"example.com/independent-harness/internal/tui"
)

var version = "0.1.0-dev"

const usage = "usage: goh [setup|settings|run|chat|resume|compact|sessions|changes|memory|serve|acp|doctor|version]; use --help after a command"

func main() {
	if e := run(os.Args[1:]); e != nil {
		fmt.Fprintln(os.Stderr, e)
		var x *exitError
		if errors.As(e, &x) {
			os.Exit(x.code)
		}
		os.Exit(1)
	}
}

type exitError struct {
	code   int
	status string
}

func (e *exitError) Error() string { return e.status }
func run(args []string) error {
	if len(args) == 0 {
		return chatFirst([]string{"chat"})
	}
	if args[0] == "chat" {
		return chatFirst(args)
	}
	return runCommand(args, false)
}

func runCommand(args []string, skipChooser bool) error {
	if args[0] == "--help" || args[0] == "-h" {
		fmt.Println(usage)
		return nil
	}
	if args[0] == "version" {
		fmt.Println(version)
		return nil
	}
	if (args[0] == "setup" || args[0] == "settings") && len(args) == 1 {
		if args[0] == "setup" {
			_, err := configureProvider()
			return err
		}
		return launch()
	}
	if args[0] == "acp" {
		return runACP(args[1:])
	}
	if args[0] == "serve" {
		return serve(args[1:])
	}
	if args[0] == "doctor" {
		return doctor(args[1:])
	}
	if args[0] == "memory" {
		return runMemory(args[1:])
	}
	if args[0] == "changes" {
		return changes(args[1:])
	}
	if args[0] == "sessions" {
		return sessions(args[1:])
	}
	command := args[0]
	args = args[1:]
	resumeID := ""
	if command == "resume" || command == "compact" {
		if len(args) == 0 {
			return fmt.Errorf("%s requires session id", command)
		}
		if args[0] != "--help" && args[0] != "-h" {
			resumeID = args[0]
			args = args[1:]
		}
	} else if command != "run" && command != "chat" {
		return fmt.Errorf("unknown command %q", command)
	}
	f := flag.NewFlagSet(command, flag.ContinueOnError)
	configPath := f.String("config", "", "explicit user JSON configuration")
	profileName := f.String("profile", "", "configured model profile")
	agentName := f.String("agent", "", "build, plan, or configured custom agent")
	project := f.String("project", ".", "project directory")
	prompt := f.String("prompt", "", "user request")
	format := f.String("format", "text", "text or json events")
	protocol := f.String("protocol", "", "chat_completions or anthropic_messages")
	endpoint := f.String("endpoint", "", "exact provider API URL")
	baseURL := f.String("base-url", "", "provider API base URL including its version path")
	apiKeyEnv := f.String("api-key-env", "", "environment variable containing the provider API key")
	model := f.String("model", "", "model name")
	window := f.Int("context-window", 0, "configured context window")
	output := f.Int("output-budget", 0, "response token reserve")
	sessionDir := f.String("session-dir", "", "private session storage")
	allow := f.String("allow", "", "comma-separated run grants: write_file,create_file,edit_file,apply_patch,todowrite,skill,shell,task,child:TOOL,format:NAME,snapshot_undo,snapshot_redo,webfetch,lsp,lsp_start:NAME,mcp_start:NAME,mcp__NAME__TOOL")
	snapshotsEnabled := f.Bool("snapshots", false, "retain before/after file checkpoints outside project")
	selectedSession := f.String("session", "", "saved session ID for chat")
	ack := f.Bool("acknowledge-uncertain", false, "acknowledge interrupted effects after inspecting project")
	if e := f.Parse(args); e != nil {
		if errors.Is(e, flag.ErrHelp) {
			return nil
		}
		return e
	}
	if f.NArg() > 0 {
		return errors.New("unexpected positional arguments")
	}
	if *format != "text" && *format != "json" {
		return errors.New("format must be text or json")
	}
	c, e := core.LoadConfig(*configPath)
	if e != nil {
		return e
	}
	f.Visit(func(v *flag.Flag) {
		switch v.Name {
		case "profile":
			c.Profile = *profileName
		case "agent":
			c.Agent = *agentName
		}
	})
	c, e = c.Resolve(*profileName, *agentName)
	if e != nil {
		return e
	}
	if *protocol != "" {
		c.Protocol = *protocol
	}
	if *endpoint != "" || *baseURL != "" {
		c.Endpoint, c.BaseURL = *endpoint, *baseURL
	}
	if *model != "" {
		c.Model = *model
	}
	f.Visit(func(v *flag.Flag) {
		switch v.Name {
		case "api-key-env":
			c.APIKeyEnv = *apiKeyEnv
		case "snapshots":
			c.Snapshots = *snapshotsEnabled
		case "context-window":
			c.ContextWindow = *window
		case "output-budget":
			c.OutputBudget = *output
		}
	})

	if *sessionDir != "" {
		c.SessionDir = *sessionDir
	}
	if e = c.Validate(); e != nil {
		return e
	}
	if command == "chat" {
		resumeID = *selectedSession
		if resumeID == "" && !skipChooser {
			infos, x := session.List(c.SessionDir)
			if x != nil {
				return x
			}
			if len(infos) > 0 {
				id, ok, x := tui.SelectSession(infos)
				if x != nil {
					return x
				}
				if !ok {
					return nil
				}
				resumeID = id
			}
		}
	}
	var store *session.Store
	if resumeID != "" {
		store, e = session.Open(c.SessionDir, resumeID)
	} else {
		root, x := filepath.Abs(*project)
		if x != nil {
			return x
		}
		root, x = filepath.EvalSymlinks(root)
		if x != nil {
			return x
		}
		if x := core.CheckProjectCredentialBoundary(root); x != nil {
			return x
		}
		store, e = session.New(c.SessionDir, root)
	}
	if e != nil {
		return e
	}
	defer store.Close()
	a, e := agent.New(c, store)
	if e != nil {
		return e
	}
	if len(store.Pending()) > 0 && !*ack {
		return errors.New("uncertain tool effects: inspect session/project, then resume with --acknowledge-uncertain; no tools replayed")
	}
	if *ack {
		if e = store.Acknowledge(); e != nil {
			return e
		}
	}
	if e = a.RepairInterruptedHistory(); e != nil {
		return e
	}
	for _, v := range strings.Split(*allow, ",") {
		if v != "" {
			a.Grants[v] = true
		}
	}
	a.ExternalFactory = func() agent.External { return mcpclient.New(c.MCP, store.Project, a.Permission) }
	if command == "chat" {
		return tui.Run(a)
	}
	enc := json.NewEncoder(os.Stdout)
	var emitErr error
	a.Emit = func(e core.Event) {
		if *format == "json" {
			if emitErr == nil {
				emitErr = enc.Encode(e)
			}
		} else {
			switch e.Type {
			case "assistant_updated":
				if d, ok := e.Data.(map[string]any); ok {
					fmt.Print(tui.Safe(fmt.Sprint(d["text"])))
				}
			case "permission_requested":
				p, _ := e.Data.(core.Permission)
				fmt.Fprintln(os.Stderr, "Approval required:", tui.Safe(p.Tool), tui.Safe(p.Preview))
			case "run_finished":
				fmt.Println()
			}
		}
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	var status string
	if command == "compact" {
		status, e = a.Compact(ctx)
	} else {
		status, e = a.Run(ctx, *prompt)
	}
	if emitErr != nil {
		return emitErr
	}
	if e != nil {
		code := 1
		if status == "needs_permission" {
			code = 3
		}
		if status == "needs_input" {
			code = 4
		}
		if status == "cancelled" {
			code = 130
		}
		return &exitError{code: code, status: status + ": " + e.Error()}
	}
	return nil
}

func doctor(args []string) error {
	f := flag.NewFlagSet("doctor", flag.ContinueOnError)
	path := f.String("config", "", "user JSON configuration")
	if e := f.Parse(args); e != nil {
		if errors.Is(e, flag.ErrHelp) {
			return nil
		}
		return e
	}
	c, e := core.LoadConfig(*path)
	if e != nil {
		return e
	}
	c, e = c.Resolve("", "")
	if e != nil {
		return e
	}
	fmt.Printf("GoHarness %s\nplatform: %s/%s\n", version, goruntime.GOOS, goruntime.GOARCH)
	for _, name := range []string{"/bin/sh", "git"} {
		p, e := exec.LookPath(name)
		if e != nil {
			fmt.Printf("%s: unavailable\n", name)
		} else {
			fmt.Printf("%s: %s\n", name, p)
		}
	}
	if e := c.Validate(); e != nil {
		fmt.Println("provider configuration:", e)
	} else {
		fmt.Printf("provider configuration: valid (model %s, context %d)\n", tui.Safe(c.Model), c.ContextWindow)
	}
	fmt.Printf("session directory: %s\nconfigured MCP servers: %d\nconfigured LSP servers: %d\n", tui.Safe(c.SessionDir), len(c.MCP), len(c.LSP))
	entries, formatErr := formatters.Discover(c.Formatters, ".")
	if formatErr != nil {
		fmt.Println("formatter configuration:", formatErr)
	} else {
		for _, entry := range entries {
			fmt.Printf("formatter %s: enabled=%t available=%t\n", entry.Name, entry.Enabled, entry.Available)
		}
	}
	fmt.Printf("file checkpoints enabled: %t\n", c.Snapshots)
	credential := c.CredentialEnvironment()
	fmt.Printf("protocol: %s\ncredential variable %s set: %t\n", c.EffectiveProtocol(), tui.Safe(credential), os.Getenv(credential) != "")
	if key, err := c.ProviderKey(); err == nil {
		fmt.Printf("provider key available: %t\n", key != "")
	} else {
		fmt.Println("provider key:", err)
	}
	for role, profile := range c.ModelRoles {
		fmt.Printf("model role %s: profile %s\n", tui.Safe(role), tui.Safe(profile))
	}
	fmt.Println("No network requests or server starts performed.")
	return nil
}
