package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/Sanidhya14321/goh/internal/sandbox"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	goruntime "runtime"
	"slices"
	"strings"
	"syscall"

	"github.com/Sanidhya14321/goh/internal/core"
	"github.com/Sanidhya14321/goh/internal/formatters"
	"github.com/Sanidhya14321/goh/internal/mcpclient"
	agent "github.com/Sanidhya14321/goh/internal/runtime"
	"github.com/Sanidhya14321/goh/internal/session"
	"github.com/Sanidhya14321/goh/internal/tools"
	"github.com/Sanidhya14321/goh/internal/tui"
)

var version = "0.1.0-dev"

// commandHelp lists every top-level command; TestUsageListsDispatchedCommands
// keeps it in step with runCommand.
var commandHelp = [][2]string{
	{"chat", "open the terminal UI (the default with no command)"},
	{"run", "run one request headlessly: goh run fix the tests"},
	{"resume", "continue a saved session by ID"},
	{"compact", "compact a saved session by ID"},
	{"sessions", "list, rename, export, import, delete or salvage sessions"},
	{"export", "export a session (default: this project's latest) as JSON or Markdown"},
	{"import", "import a session exported with goh export"},
	{"changes", "show files changed in a session"},
	{"snapshots", "whole-project snapshots per run: list, diff, restore, undo"},
	{"models", "list models available from the configured provider"},
	{"pr", "check out a GitHub pull request with gh, then open the UI"},
	{"worktree", "open the UI in a separate git worktree: goh worktree NAME|list"},
	{"mcp", "list configured MCP servers: goh mcp list"},
	{"agents", "list built-in, configured and project agents"},
	{"stats", "token usage and activity over recent days"},
	{"auth", "sign in with ChatGPT: login, status, models, logout; list credential sources; store file|keychain"},
	{"setup", "configure a provider"},
	{"settings", "open settings"},
	{"memory", "opt-in project notes: add, list, recall, delete"},
	{"serve", "local HTTP API"},
	{"attach", "terminal UI for a running goh serve: goh attach [ADDRESS]"},
	{"acp", "Agent Client Protocol over stdio for editors"},
	{"config", "print the effective configuration, secrets removed"},
	{"doctor", "check configuration and integrations"},
	{"version", "print the version"},
	{"uninstall", "remove goh and its data (lists first; --yes to remove)"},
	{"completion", "print a shell completion script: bash, zsh or fish"},
}

var usage = func() string {
	var b strings.Builder
	b.WriteString("usage: goh [command] [flags]\n\nCommands:\n")
	for _, c := range commandHelp {
		fmt.Fprintf(&b, "  %-10s %s\n", c[0], c[1])
	}
	b.WriteString("\nUse --help after a command for its flags.")
	return b.String()
}()

func main() {
	sandbox.HelperMain() // never returns when started as the Linux sandbox helper
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
	return runCommand(args, false, "", "")
}

func runCommand(args []string, skipChooser bool, notice, draft string) error {
	if args[0] == "--help" || args[0] == "-h" {
		fmt.Println(usage)
		return nil
	}
	if args[0] == "version" {
		fmt.Println(version)
		return nil
	}
	if args[0] == "completion" {
		shell := ""
		if len(args) > 1 {
			shell = args[1]
		}
		return writeCompletion(os.Stdout, shell)
	}
	if (args[0] == "setup" || args[0] == "settings") && len(args) == 1 {
		if args[0] == "setup" {
			_, err := configureProvider()
			return err
		}
		project := "."
		return launch(&project)
	}
	if args[0] == "auth" {
		return chatGPTAuth(args[1:])
	}
	if args[0] == "acp" {
		return runACP(args[1:])
	}
	if args[0] == "serve" {
		return serve(args[1:])
	}
	if args[0] == "attach" {
		return runAttach(args[1:])
	}
	if args[0] == "doctor" {
		return doctor(args[1:])
	}
	if args[0] == "models" {
		return listModels(args[1:])
	}
	if args[0] == "export" || args[0] == "import" {
		return exportAlias(args[0], args[1:])
	}
	if args[0] == "config" {
		return runConfigShow(args[1:], os.Stdout)
	}
	if args[0] == "uninstall" {
		return runUninstall(args[1:], os.Stdout)
	}
	if args[0] == "pr" {
		return runPR(args[1:], func() error { return chatFirst([]string{"chat"}) })
	}
	if args[0] == "worktree" {
		return runWorktree(args[1:], func() error { return chatFirst([]string{"chat"}) })
	}
	if args[0] == "mcp" {
		return runMCP(args[1:])
	}
	if args[0] == "agents" {
		return listAgents(args[1:])
	}
	if args[0] == "stats" {
		return stats(args[1:])
	}
	if args[0] == "memory" {
		return runMemory(args[1:])
	}
	if args[0] == "snapshots" {
		return runSnapshots(args[1:], os.Stdout)
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
	protocol := f.String("protocol", "", "chat_completions, anthropic_messages, openai_responses or openai_chatgpt")
	endpoint := f.String("endpoint", "", "exact provider API URL")
	baseURL := f.String("base-url", "", "provider API base URL including its version path")
	apiKeyEnv := f.String("api-key-env", "", "environment variable containing the provider API key")
	model := f.String("model", "", "model name")
	window := f.Int("context-window", 0, "configured context window")
	output := f.Int("output-budget", 0, "response token reserve")
	effort := f.String("reasoning-effort", "", "OpenAI: none|minimal|low|medium|high|xhigh; Anthropic: low|medium|high|xhigh|max; off disables")
	verifyCmd := f.String("verify", "", "run: check command after file changes, with bounded repair (overrides config)")
	maxToolCalls := f.Int("max-tool-calls", 0, "run: stop after this many tool calls, child tasks included")
	maxRunSeconds := f.Int("max-run-seconds", 0, "run: stop after this many seconds, child tasks included")
	sandboxContainer := f.String("sandbox-container", "", "run: run shell commands inside this running Docker container (project mounted at the same path)")
	maxRunTokens := f.Int("max-run-tokens", 0, "run: stop before the next model turn once this many tokens are used, child tasks included")
	lineEdits := f.Bool("line-edits", false, "run: let edit_file replace numbered line ranges as well as exact text")
	retryInterrupted := f.Int("retry-interrupted", 0, "run: re-send a model request whose stream broke, up to N times (0-2); partial replies are discarded")
	completionCheck := f.Bool("completion-check", true, "remind once to check unverified file changes before finishing")
	sessionDir := f.String("session-dir", "", "private session storage")
	allow := f.String("allow", "", "comma-separated run grants (all = every built-in local tool: file edits, shell, todos, skills, tasks): write_file,create_file,edit_file,apply_patch,todowrite,skill,shell,task,child:TOOL,format:NAME,snapshot_undo,snapshot_redo,webfetch,lsp,lsp_start:NAME,mcp_start:NAME,mcp__NAME__TOOL")
	snapshotsEnabled := f.Bool("snapshots", false, "retain before/after file checkpoints outside project")
	selectedSession := f.String("session", "", "saved session ID to continue (chat and run)")
	continueLast := f.Bool("continue", false, "continue this project's most recent session (chat and run)")
	compactFocus := f.String("focus", "", "compact: what the summary should keep (up to 500 characters)")
	commandName := f.String("command", "", "run a custom command; message words become its arguments")
	forkSession := f.Bool("fork", false, "continue a copy of --session or --continue; the original is unchanged")
	title := f.String("title", "", "title for the session (shown in pickers and lists)")
	var files []string
	f.Func("file", "attach a project file to the run message (repeatable; images become @path)", func(v string) error {
		files = append(files, v)
		return nil
	})
	ack := f.Bool("acknowledge-uncertain", false, "acknowledge interrupted effects after inspecting project")
	// Flags and message words may be interleaved: goh run fix the tests --model x
	var words []string
	for rest := args; ; {
		if e := f.Parse(rest); e != nil {
			if errors.Is(e, flag.ErrHelp) {
				return nil
			}
			return e
		}
		if f.NArg() == 0 {
			break
		}
		words = append(words, f.Arg(0))
		rest = f.Args()[1:]
	}
	if command == "run" && *prompt == "" && !slices.Contains(words, "-") && pipedInput(os.Stdin) {
		words = append(words, "-") // cat log | goh run explain this
	}
	if len(words) > 0 {
		if command != "run" {
			return errors.New("unexpected positional arguments")
		}
		if *prompt != "" {
			return errors.New("give the message either as words or with --prompt, not both")
		}
		message, err := runMessage(words, os.Stdin)
		if err != nil {
			return err
		}
		*prompt = message
	} else if *prompt == "-" {
		message, err := runMessage([]string{"-"}, os.Stdin)
		if err != nil {
			return err
		}
		*prompt = message
	}
	if *commandName != "" {
		if command != "run" {
			return errors.New("--command is only for run")
		}
		message, err := commandPrompt(*commandName, *prompt)
		if err != nil {
			return err
		}
		*prompt = message
	}
	if len(files) > 0 {
		if command != "run" {
			return errors.New("--file is only for run")
		}
		if strings.TrimSpace(*prompt) == "" {
			return errors.New("--file needs a message")
		}
	}
	if *format != "text" && *format != "json" {
		return errors.New("format must be text or json")
	}
	c, e := core.LoadConfig(*configPath)
	if e != nil {
		return e
	}
	if *selectedSession != "" {
		// A resumed session runs in its own project: project agents and
		// --file attachments come from there, and a different --project is
		// refused rather than mixed in.
		dir := c.SessionDir
		if *sessionDir != "" {
			dir = *sessionDir
		}
		if sessionProject, err := session.ProjectOf(dir, *selectedSession); err == nil {
			projectSet := false
			f.Visit(func(v *flag.Flag) { projectSet = projectSet || v.Name == "project" })
			if projectSet && !sameDir(*project, sessionProject) {
				return fmt.Errorf("session %s belongs to project %s, not --project %s", *selectedSession, sessionProject, *project)
			}
			*project = sessionProject
		}
	}
	if len(files) > 0 {
		attached, err := attachFiles(*project, files, func(rel string, fold bool) bool { return c.FileDecisionFold([]string{rel}, fold) == "deny" })
		if err != nil {
			return err
		}
		*prompt += attached
	}
	f.Visit(func(v *flag.Flag) {
		switch v.Name {
		case "profile":
			c.Profile = *profileName
		case "agent":
			c.Agent = *agentName
		}
	})
	if dir, err := filepath.Abs(*project); err == nil {
		c, _ = c.WithProjectAgents(dir)
	}
	c, e = c.Resolve(*profileName, *agentName)
	if e != nil {
		return e
	}
	if *protocol == core.ChatGPTProtocol && *endpoint == "" && *baseURL == "" {
		c.SetProviderRoute(*protocol, "", "")
	}
	if *protocol != "" || *endpoint != "" || *baseURL != "" {
		nextProtocol, nextEndpoint, nextBase := c.Protocol, c.Endpoint, c.BaseURL
		if *protocol != "" {
			nextProtocol = *protocol
		}
		if *endpoint != "" || *baseURL != "" {
			nextEndpoint, nextBase = *endpoint, *baseURL
		}
		c.SetProviderRoute(nextProtocol, nextEndpoint, nextBase)
		// A process-wide override is explicitly selected by the caller.
		if v, ok := os.LookupEnv("HARNESS_API_KEY_ENV"); ok {
			c.APIKeyEnv = v
		}
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
			c.SnapshotsStrict = *snapshotsEnabled
		case "context-window":
			c.ContextWindow = *window
		case "output-budget":
			c.OutputBudget = *output
		case "reasoning-effort":
			c.ReasoningEffort = strings.ToLower(*effort)
			if c.ReasoningEffort == "off" {
				c.ReasoningEffort = ""
			}
		case "verify":
			c.Verify = &core.VerifyConfig{Command: *verifyCmd}
		case "max-tool-calls":
			c.MaxToolCalls = *maxToolCalls
		case "max-run-seconds":
			c.MaxRunSeconds = *maxRunSeconds
		case "sandbox-container":
			if c.Sandbox == nil {
				c.Sandbox = &core.SandboxConfig{}
			}
			c.Sandbox.Container = *sandboxContainer
		case "max-run-tokens":
			c.MaxRunTokens = *maxRunTokens
		case "line-edits":
			c.LineEdits = *lineEdits
		case "retry-interrupted":
			c.RetryInterruptedStreams = *retryInterrupted
		case "completion-check":
			c.CompletionCheck = completionCheck
		}
	})

	if *sessionDir != "" {
		c.SessionDir = *sessionDir
	}
	if *configPath == "" && *profileName == "" {
		c = withChatGPTDefault(c)
	}
	c = c.WithDetectedLSP(nil)
	c = c.WithDetectedFormatters(nil)
	if c.EffectiveProtocol() == core.ChatGPTProtocol {
		if c.Model == "" || (*protocol == core.ChatGPTProtocol && *model == "") {
			if chosen, err := core.ChatGPTModel(); err == nil && chosen != "" {
				c.Model = chosen
			}
		}
		if c.ContextWindow == 0 {
			c.ContextWindow = core.ChatGPTContextWindow(context.Background(), c.Model)
		}
	}
	if e = c.Validate(); e != nil {
		return e
	}
	if (command == "chat" || command == "run") && (*continueLast || *selectedSession != "") {
		if *continueLast && *selectedSession != "" {
			return errors.New("use either --continue or --session, not both")
		}
		resumeID = *selectedSession
		if *continueLast {
			root, x := filepath.Abs(*project)
			if x == nil {
				root, x = filepath.EvalSymlinks(root)
			}
			if x != nil {
				return x
			}
			if resumeID, x = session.Latest(c.SessionDir, root); x != nil {
				return x
			}
		}
	} else if command == "chat" {
		resumeID = *selectedSession
		if resumeID == "" && !skipChooser {
			infos, x := session.ListForPicker(c.SessionDir)
			if x != nil {
				return x
			}
			if len(infos) > 0 {
				id, ok, x := tui.SelectSession(infos, c.SessionDir, "", "")
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
	if resumeID != "" && *forkSession {
		store, e = session.Fork(c.SessionDir, resumeID)
		if e == nil {
			fmt.Fprintf(os.Stderr, "Forked session %s into %s\n", resumeID, store.ID)
		}
	} else if *forkSession {
		return errors.New("--fork needs --session ID or --continue")
	} else if resumeID != "" {
		store, e = session.Open(c.SessionDir, resumeID)
		if errors.Is(e, session.ErrDamaged) {
			if command != "run" && interactive() {
				store, e = offerSalvage(os.Stdin, os.Stderr, c.SessionDir, resumeID, e)
			} else {
				e = fmt.Errorf("%w\nThe damaged journal was not changed. Recover its readable conversation into a new session with:\n  goh sessions salvage %s", e, resumeID)
			}
		}
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
	if t := session.CleanTitle(*title); t != "" {
		if e := session.SetTitle(c.SessionDir, store.ID, t); e != nil {
			return e
		}
	}
	a, e := agent.New(c, store)
	if e != nil {
		return e
	}
	// Background jobs belong to this chat and never outlive it. Opening a
	// picker or settings rebuilds the chat; jobs carry over when it reopens
	// on the same session (see parkedJobs).
	if command == "chat" {
		adoptParkedJobs(store.ID, a.Tools)
	}
	keepJobs := false
	defer func() {
		if !keepJobs {
			a.Tools.StopJobs()
		}
	}()
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
		if v == "all" {
			// Every built-in local tool; web access and MCP stay explicit.
			for _, g := range allLocalGrants {
				a.Grants[g] = true
			}
			continue
		}
		if v != "" {
			a.Grants[v] = true
		}
	}
	a.ExternalFactory = func() agent.External { return mcpclient.New(c.MCP, store.Project, a.Permission) }
	if command == "chat" {
		tui.Version = version
		err := tui.Run(a, notice, draft)
		if nav := (*tui.ChatNavigation)(nil); errors.As(err, &nav) {
			parkedJobs = &parkedEngine{session: store.ID, tools: a.Tools}
			keepJobs = true
		}
		return err
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
			case "stream_retry":
				fmt.Fprintln(os.Stderr, "\nStream broke; the partial reply was discarded and the request re-sent")
			case "assistant_reset":
				fmt.Fprintln(os.Stderr, "\nThe final reply differs from the streamed text above; it follows in full")
			case "provider_retry":
				if n, ok := e.Data.(core.RetryNotice); ok {
					fmt.Fprintf(os.Stderr, "Provider busy (%s); retrying in %.1fs (attempt %d of %d)\n", tui.Safe(n.Reason), float64(n.DelayMS)/1000, n.Attempt, n.Max)
				}
			case "permission_requested":
				p, _ := e.Data.(core.Permission)
				fmt.Fprintln(os.Stderr, "Approval required:", tui.Safe(p.Tool), tui.Safe(p.Preview))
			case "run_finished":
				fmt.Println()
				if d, ok := e.Data.(map[string]any); ok && d["outcome"] == agent.OutcomeUnverified {
					fmt.Fprintln(os.Stderr, "Note: files were changed but no check (test or build command) succeeded afterwards.")
				}
			}
		}
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	var status string
	if command == "compact" {
		status, e = a.CompactWith(ctx, *compactFocus)
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
		if status == "budget_exhausted" {
			code = 5
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
	if _, err := core.ChatGPTModel(); err == nil {
		fmt.Println("ChatGPT sign-in: saved")
	} else {
		fmt.Println("ChatGPT sign-in: none")
	}
	c = withChatGPTDefault(c)
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
	if c.EffectiveProtocol() == core.ChatGPTProtocol {
		_, err := core.ChatGPTModel()
		fmt.Printf("ChatGPT authorization saved: %t\n", err == nil)
	} else if key, err := c.ProviderKey(); err == nil {
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

// runMessage joins message words; a "-" word is replaced by standard input
// (at most 1 MiB), so `git diff | goh run - review this` works. Standard
// input is never read unless asked, so scripts with open pipes never hang.
// pipedInput reports whether stdin is a pipe or a regular file, so piped
// text joins the run message. Terminals and devices such as /dev/null are
// never read.
func pipedInput(f *os.File) bool {
	info, err := f.Stat()
	if err != nil {
		return false
	}
	m := info.Mode()
	return m&os.ModeNamedPipe != 0 || m.IsRegular()
}

func runMessage(words []string, stdin io.Reader) (string, error) {
	var parts []string
	read := false
	for _, w := range words {
		if w != "-" {
			parts = append(parts, w)
			continue
		}
		if read {
			return "", errors.New("standard input (-) can be used once")
		}
		read = true
		b, err := io.ReadAll(io.LimitReader(stdin, 1<<20+1))
		if err != nil {
			return "", err
		}
		if len(b) > 1<<20 {
			return "", errors.New("standard input exceeds 1 MiB")
		}
		parts = append(parts, "\n\n"+strings.TrimRight(string(b), "\n")+"\n\n")
	}
	message := strings.TrimSpace(strings.NewReplacer("\n\n ", "\n\n", " \n\n", "\n\n").Replace(strings.Join(parts, " ")))
	if message == "" {
		return "", errors.New("empty message")
	}
	return message, nil
}

// withChatGPTDefault uses a saved ChatGPT sign-in when no provider, endpoint
// or model is configured at all. It never replaces an explicit provider, so
// prompts only go where the user set things up.
func withChatGPTDefault(c core.Config) core.Config {
	if c.Protocol != "" || c.Endpoint != "" || c.BaseURL != "" || c.Model != "" {
		return c
	}
	if chosen, err := core.ChatGPTModel(); err == nil && chosen != "" {
		c.SetProviderRoute(core.ChatGPTProtocol, "", "")
		c.Model = chosen
		if c.ContextWindow == 0 {
			c.ContextWindow = core.ChatGPTContextWindow(context.Background(), c.Model)
		}
	}
	return c
}

// allLocalGrants is what --allow all grants: built-in tools that act on the
// local project. Web access, language servers and MCP must still be named.
var allLocalGrants = []string{"write_file", "create_file", "edit_file", "apply_patch", "todowrite", "skill", "shell", "task",
	"child:write_file", "child:create_file", "child:edit_file", "child:apply_patch", "child:shell", "snapshot_undo", "snapshot_redo"}

// parkedEngine holds the tool engine of a chat that closed for navigation, so
// its background jobs survive until the chat reopens.
type parkedEngine struct {
	session string
	tools   *tools.Engine
}

var parkedJobs *parkedEngine

// adoptParkedJobs moves parked jobs to next when the chat reopens on the same
// session and project, and stops them otherwise.
func adoptParkedJobs(session string, next *tools.Engine) {
	p := parkedJobs
	parkedJobs = nil
	if p == nil {
		return
	}
	if p.session != session || p.tools.ShareJobs(next) != nil {
		p.tools.StopJobs()
	}
}

// stopParkedJobs stops jobs left by a chat that will not reopen.
func stopParkedJobs() {
	if p := parkedJobs; p != nil {
		parkedJobs = nil
		p.tools.StopJobs()
	}
}
