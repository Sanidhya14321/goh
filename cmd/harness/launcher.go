package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Sanidhya14321/goh/internal/core"
	"github.com/Sanidhya14321/goh/internal/rewind"
	"github.com/Sanidhya14321/goh/internal/session"
	"github.com/Sanidhya14321/goh/internal/tui"
)

func settingsText(path string) (string, error) {
	raw, err := core.ReadConfigText(path)
	if os.IsNotExist(err) {
		return "{}\n", nil
	}
	return raw, err
}

func editConfiguration() error {
	path, err := core.UserConfigPath()
	if err != nil {
		return err
	}
	raw, err := settingsText(path)
	if err != nil {
		return err
	}
	_, err = tui.EditSettings("Advanced configuration", "Edit GoHarness JSON settings. Providers and their keys are easier to manage in /providers.", []tui.SettingsField{{Label: path, Value: raw}}, true, func(fields []tui.SettingsField) error {
		return core.WriteUserConfig(path, fields[0].Value)
	})
	return err
}

// currentProject is the folder the next chat opens in: the chosen
// workspace, or the directory goh was started from.
func currentProject(project string) string {
	if project == "" {
		project = "."
	}
	abs, err := filepath.Abs(project)
	if err != nil {
		return project
	}
	return abs
}

// displayPath writes paths under the home folder with ~.
func displayPath(path string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	if path == home {
		return "your home folder (~)"
	}
	if rel, err := filepath.Rel(home, path); err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "~" + string(filepath.Separator) + rel
	}
	return path
}

// chooseWorkspace opens the project folder browser at start. The chosen
// folder is trusted and recorded as recently opened.
func chooseWorkspace(start string) (string, bool, error) {
	var recent []string
	for _, w := range core.Workspaces() {
		if len(recent) == 8 {
			break
		}
		recent = append(recent, w.Path)
	}
	path, ok, err := tui.ChooseWorkspace(tui.WorkspaceChooser{
		Start:  start,
		Recent: recent,
		Check: func(path string) error {
			if core.CheckProjectCredentialBoundary(path) != nil {
				return errors.New("goh's saved keys live in this folder, so it can't be a project; open a folder inside it")
			}
			return nil
		},
		Trusted: core.IsTrustedWorkspace,
		Kind:    core.ProjectKind,
	})
	if err != nil || !ok {
		return "", false, err
	}
	return path, true, core.OpenWorkspace(path)
}

// picksWorkspace reports whether chat should ask for a project folder at
// start: no folder or session was named on the command line.
func picksWorkspace(args []string) bool {
	for _, a := range args {
		if a == "--" {
			break
		}
		name, _, _ := strings.Cut(strings.TrimLeft(a, "-"), "=")
		if strings.HasPrefix(a, "-") && (name == "project" || name == "session" || name == "continue" || name == "fork") {
			return false
		}
	}
	return true
}

// startupFlags are chat flags that act on the first session only; the value
// says whether the flag takes a separate argument.
var startupFlags = map[string]bool{"session": true, "title": true, "continue": false, "fork": false, "acknowledge-uncertain": false}

// withoutStartupFlags drops startupFlags from args, in any of the forms the
// flag package accepts (-x, --x, -x=v, --x v).
func withoutStartupFlags(args []string) []string {
	out := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--" {
			return append(out, args[i:]...)
		}
		name := strings.TrimLeft(a, "-")
		if !strings.HasPrefix(a, "-") || name == "" {
			out = append(out, a)
			continue
		}
		name, _, hasValue := strings.Cut(name, "=")
		takesValue, ok := startupFlags[name]
		if !ok {
			out = append(out, a)
			continue
		}
		if takesValue && !hasValue {
			i++ // skip the value
		}
	}
	return out
}

// keepsStartupFlags reports whether the startup session flags still apply
// after a navigation: only when no session has opened yet and the user is
// setting up a provider, after which the requested session should open.
func keepsStartupFlags(opened bool, action string) bool {
	return !opened && (action == "connect" || action == "provider")
}

func chatFirst(args []string) error {
	defer stopParkedJobs()
	// One terminal program for the whole session: chat, pickers and setup
	// screens replace each other inside it instead of restarting.
	defer tui.StartHost()()
	project, sessionID, draft := "", "", ""
	if picksWorkspace(args) {
		// A trusted folder opens directly; any other shows the browser,
		// starting where goh was run.
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		if core.CheckProjectCredentialBoundary(cwd) == nil && core.IsTrustedWorkspace(cwd) {
			_ = core.OpenWorkspace(cwd) // only orders the recent list
		} else {
			chosen, ok, err := chooseWorkspace(cwd)
			if err != nil || !ok {
				return err
			}
			project = chosen
		}
	}
	var agent *string
	chatModel := "" // a /models choice for a chat whose route is not saved
	// The route of a provider connected from a chat whose --config file
	// does not include it; without it the chat would still have no model.
	var connected []string
	sessionOverride := false
	skipChooser := true
	var actionErr error
	info := ""        // non-error notice for the next chat screen
	welcomed := false // the first-run provider picker was offered
	for {
		command := append([]string(nil), args...)
		if project != "" {
			command = append(command, "--project", project)
		}
		if agent != nil {
			command = append(command, "--agent="+*agent)
		}
		command = append(command, connected...)
		if chatModel != "" {
			command = append(command, "--model="+chatModel)
		}
		if sessionOverride || sessionID != "" || !skipChooser {
			command = append(command, "--session="+sessionID)
			// A session-only access level follows only its own session.
			if level := tui.SessionAccessFor(sessionID); level != "" {
				command = append(command, "--access="+string(level))
			}
		}
		notice := info
		if actionErr != nil {
			notice = actionErr.Error()
		}
		info = ""
		err := runCommand(command, skipChooser, notice, draft)
		var nav *tui.ChatNavigation
		if err == nil {
			return nil
		}
		if errors.Is(err, tui.ErrHostEnded) {
			return nil // the terminal program ended (SIGTERM); nothing can be shown
		}
		opened := errors.As(err, &nav) // chat ran, so its session was opened
		if !opened {
			fresh := noProviders()
			if fresh && !welcomed && actionErr == nil {
				// First run: go straight to connecting a provider instead of
				// a blocked chat. "Not now" falls through to the chat screen.
				welcomed = true
				added, notice, err := addProviderFlow("", true)
				if err == nil && added != "" {
					info, chatModel = notice, ""
					connected = connectedRoute(args, added)
					continue
				}
				actionErr = err
			}
			headline, problem, fix := "Chat is waiting for setup.", err.Error(), tui.FixConnect
			help := "Press Enter to add a provider: ChatGPT sign-in, Claude API, OpenAI, OpenRouter, Gemini, Groq, Ollama or any compatible URL. /providers manages them, /workspace picks the project and /settings has the rest."
			switch {
			case errors.Is(err, core.ErrProjectHoldsKeys):
				// A provider cannot fix this: the folder has to change.
				headline, fix = "Chat can't open this folder.", tui.FixWorkspace
				problem = "goh's saved keys live inside " + displayPath(currentProject(project)) + ", so it can't be the project."
				help = "Press Enter to choose a project folder. Next time, run goh from inside a project (cd ~/code/app && goh)."
			case errors.Is(err, session.ErrNotFound):
				headline, fix = "This session no longer exists.", tui.FixSessions
				help = "Press Enter to choose another session or start a new one (the picker's first row)."
			case errors.Is(err, session.ErrBusy) || errors.Is(err, session.ErrSessionOpen):
				headline, fix = "This session is open in another GoHarness.", tui.FixSessions
				help = "Close the other window, or press Enter to choose another session. Nothing was changed."
			case errors.Is(err, session.ErrDamaged):
				headline, fix = "This session's journal is damaged.", tui.FixSessions
				help = "Nothing was changed. Recover its readable part with goh sessions salvage ID, or press Enter to choose another session."
			case strings.Contains(err.Error(), "uncertain tool effects"):
				headline, fix = "This session has tool effects whose outcome is unknown.", tui.FixSessions
				help = "Inspect the project, then resume with goh chat --session ID --acknowledge-uncertain, or press Enter to choose another session."
			case strings.Contains(err.Error(), "belongs to project"):
				headline, fix = "This session belongs to another project.", tui.FixSessions
				help = "Press Enter to choose a session; it opens in its own project."
			case fresh:
				problem = "No model provider is connected yet."
			}
			if actionErr != nil {
				// Put the failure of the last setup step ahead of the state.
				problem = actionErr.Error() + " · " + problem
			}
			err = tui.RunBlocked("Chat setup required", headline+"\n"+problem+"\n\n"+help, fix, draft)
			if !errors.As(err, &nav) {
				return err
			}
		}
		// Flags that pick, copy, name or acknowledge the starting session
		// apply once. After navigation the launcher passes the session
		// itself, so they must not reach the next session too. Setup that
		// ran before any session opened (connecting a provider) keeps them,
		// so the requested session still opens afterwards.
		if !keepsStartupFlags(opened, nav.Action) {
			args = withoutStartupFlags(args)
			if nav.Config != nil || nav.Session != "" {
				sessionID = nav.Session
			}
			sessionOverride = true
		}
		skipChooser = true
		actionErr = nil
		draft = ""
		switch nav.Action {
		case "connect":
			// Add a provider; it becomes the one new chats use.
			draft = nav.Draft
			var added string
			added, info, actionErr = addProvider(nav.Backdrop)
			if added != "" {
				chatModel = ""
				connected = connectedRoute(args, added)
			}
		case "plan_exit":
			// The plan was approved in chat: same session, build agent.
			draft = nav.Draft
			build := "build"
			agent = &build
			info = "Switched to Build to carry out the approved plan. Press Enter to start."
		case "provider":
			draft = nav.Draft
			var changed bool
			changed, info, actionErr = manageProviders(nav.Backdrop)
			if changed {
				chatModel = "" // the provider's own model applies again
				connected = connectedRoute(args, "")
			}
		case "settings":
			draft = nav.Draft
			workspace := project
			actionErr = launch(&workspace)
			if workspace != "." && workspace != project {
				project, sessionID = workspace, ""
			}
		case "access":
			draft = nav.Draft
			workspace := nav.Project
			if workspace == "" {
				workspace = currentProject(project)
			}
			info, actionErr = chooseAccess(workspace, nav.Session, nav.Backdrop)
		case "workspace":
			draft = nav.Draft
			var chosen string
			var ok bool
			chosen, ok, actionErr = chooseWorkspace(currentProject(project))
			if ok {
				project, sessionID = chosen, ""
				info = "Project: " + displayPath(chosen)
			}
		case "sessions":
			draft = nav.Draft
			cfg := core.Defaults()
			if nav.Config != nil {
				cfg = *nav.Config
			} else if loaded, err := core.LoadConfig(""); err == nil {
				cfg = loaded
			}
			var infos []session.Info
			infos, actionErr = session.ListForPicker(cfg.SessionDir)
			if actionErr != nil {
				break
			}
			workspace := nav.Project
			if workspace == "" {
				workspace = project
			}
			if workspace == "" {
				workspace, _ = os.Getwd()
			}
			var selected string
			var chosen bool
			selected, chosen, actionErr = tui.SelectSession(infos, cfg.SessionDir, sessionID, workspace)
			if chosen {
				sessionID = selected
				if selected == "" {
					project = workspace
				} else if recorded, err := session.ProjectOf(cfg.SessionDir, selected); err == nil {
					// The session runs in the project it recorded, which
					// may be another workspace than the current one.
					project = recorded
				}
			}
		case "undo", "redo", "revert":
			draft = nav.Draft
			var result rewind.Result
			if nav.Action == "undo" {
				result, actionErr = rewind.Undo(context.Background(), nav.Config.SessionDir, nav.Session)
			} else if nav.Action == "revert" {
				result, actionErr = rewind.RevertTo(context.Background(), nav.Config.SessionDir, nav.Session, nav.RevertPrompt, nav.RevertOccurrence)
			} else {
				result, actionErr = rewind.Redo(context.Background(), nav.Config.SessionDir, nav.Session)
			}
			if actionErr != nil {
				break
			}
			sessionID = result.Session
			info = result.Summary(nav.Action != "redo")
			if nav.Action != "redo" {
				draft = result.Prompt
			}
		case "fork":
			draft = nav.Draft
			var branch *session.Store
			var prompt string
			branch, prompt, actionErr = session.ForkBefore(nav.Config.SessionDir, nav.Session, nav.BeforeTurn)
			if actionErr != nil {
				break
			}
			sessionID = branch.ID
			branch.Close()
			if nav.BeforeTurn > 0 {
				draft = prompt
			}
		case "new":
			sessionID = ""
			if nav.Project != "" {
				project = nav.Project
			}
		case "models":
			// The choice is saved on the provider, so it stays for new chats.
			draft = nav.Draft
			var override string
			override, info, actionErr = chooseModel(nav.Config, nav.Backdrop)
			if override != "" {
				chatModel = override
			} else if actionErr == nil && info != "" {
				chatModel = "" // a saved choice replaces a chat-only one
			}
		case "agents", "cycle_agent", "previous_agent":
			draft = nav.Draft
			cfg := core.Defaults()
			if nav.Config != nil {
				cfg = *nav.Config
			} else if loaded, err := core.LoadConfig(""); err == nil {
				cfg = loaded
			}
			if nav.Action != "agents" {
				selected := nextAgent(cfg, nav.Action == "previous_agent")
				agent = &selected
				break
			}
			current := cfg.Agent
			if current == "" {
				current = "build"
			}
			selected, chosen, err := tui.SelectOption("Select agent", current, agentOptions(cfg), nav.Backdrop)
			actionErr = err
			if chosen {
				agent = &selected
			}
		}
	}
}

func selectChatGPTModel(current, backdrop string) (string, bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 18*time.Second)
	defer cancel()
	models, err := core.ListChatGPTModels(ctx)
	if err != nil {
		return "", false, err
	}
	options := make([]tui.Option, 0, len(models))
	for _, model := range models {
		options = append(options, tui.Option{Value: model.Slug, Label: model.Slug, Description: model.Name})
	}
	return tui.SelectOption("Select ChatGPT model", current, options, backdrop)
}

// launch runs the settings menu. A workspace chosen there is written to
// *project, so the chat that opens next uses it.
func launch(project *string) error {
	var status string
	if *project == "" {
		*project = "."
	}
	for {
		action, err := tui.LaunchMenu(status)
		if err != nil {
			return err
		}
		status = ""
		switch action {
		case "":
			return nil
		case "provider":
			_, status, err = manageProviders("")
		case "config":
			err = editConfiguration()
		case "project":
			var chosen string
			var ok bool
			chosen, ok, err = chooseWorkspace(currentProject(*project))
			if ok {
				*project = chosen
				status = "Workspace: " + displayPath(chosen)
			}
		case "chat":
			return nil
		}
		if err != nil {
			status = err.Error()
		}
	}
}

// noProviders reports a first run: no provider saved or configured by hand.
func noProviders() bool {
	_, _, cfg, err := userSettings()
	return err == nil && len(cfg.Providers()) == 0
}

// connectedRoute is the route flags that make a just-connected provider
// apply to a chat started with --config, or nil when the chat reads the
// user settings and sees it anyway.
func connectedRoute(args []string, name string) []string {
	if !explicitConfig(args) {
		return nil
	}
	return providerFlags(name)
}
