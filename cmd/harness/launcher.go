package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
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

func configureProvider() (bool, error) {
	path, err := core.UserConfigPath()
	if err != nil {
		return false, err
	}
	raw, err := settingsText(path)
	if err != nil {
		return false, err
	}
	cfg, err := core.DecodeConfig(raw)
	if err != nil {
		return false, err
	}
	address := cfg.BaseURL
	kind := "base"
	if cfg.Endpoint != "" {
		address, kind = cfg.Endpoint, "endpoint"
	}
	window := cfg.ContextWindow
	if window == 0 {
		window = 32768
	}
	fields := []tui.SettingsField{
		{Label: "API URL (include the API version path)", Value: address},
		{Label: "URL type (base or endpoint)", Value: kind},
		{Label: "Model ID", Value: cfg.Model},
		{Label: "Context window", Value: strconv.Itoa(window)},
		{Label: "Protocol", Value: cfg.EffectiveProtocol()},
		{Label: "API key (blank keeps saved key, REMOVE deletes it)", Secret: true},
	}
	return tui.EditSettings("Provider setup", "Enter the details from your model provider. Keys are saved separately with private file permissions.", fields, false, func(fields []tui.SettingsField) error {
		next := cfg
		endpoint, baseURL := "", ""
		address := strings.TrimSpace(fields[0].Value)
		switch strings.TrimSpace(fields[1].Value) {
		case "base":
			baseURL = address
		case "endpoint":
			endpoint = address
		default:
			return errors.New("URL type must be base or endpoint")
		}
		next.Model = strings.TrimSpace(fields[2].Value)
		next.ContextWindow, err = strconv.Atoi(strings.TrimSpace(fields[3].Value))
		if err != nil {
			return errors.New("Context window must be a whole number")
		}
		next.SetProviderRoute(strings.TrimSpace(fields[4].Value), endpoint, baseURL)
		if err := next.Validate(); err != nil {
			return err
		}
		// Edit only these fields, retaining other settings as written.
		if core.HasJSONCSyntax(raw) {
			// Rewriting would silently drop the user's comments.
			return fmt.Errorf("%s has comments or trailing commas, which provider setup would erase; edit it in Advanced configuration (or by hand) instead", path)
		}
		var document map[string]json.RawMessage
		if err := json.Unmarshal([]byte(raw), &document); err != nil {
			return err
		}
		if document == nil {
			document = map[string]json.RawMessage{}
		}
		delete(document, "endpoint")
		delete(document, "base_url")
		if next.APIKeyEnv != cfg.APIKeyEnv {
			delete(document, "api_key_env")
		}
		for name, value := range map[string]any{"model": next.Model, "context_window": next.ContextWindow, "protocol": next.Protocol} {
			document[name], _ = json.Marshal(value)
		}
		addressField := "base_url"
		if next.Endpoint != "" {
			addressField = "endpoint"
		}
		document[addressField], _ = json.Marshal(address)
		data, err := json.MarshalIndent(document, "", "  ")
		if err != nil {
			return err
		}
		if key := strings.TrimSpace(fields[5].Value); key != "" {
			if key == "REMOVE" {
				key = ""
			}
			if err := next.SaveProviderKey(key); err != nil {
				return err
			}
		}
		return core.WriteUserConfig(path, string(data)+"\n")
	})
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
	_, err = tui.EditSettings("Advanced configuration", "Edit GoHarness JSON settings. Use Provider setup to store API keys.", []tui.SettingsField{{Label: path, Value: raw}}, true, func(fields []tui.SettingsField) error {
		return core.WriteUserConfig(path, fields[0].Value)
	})
	return err
}

func editWorkspace(project *string) (bool, error) {
	return tui.EditSettings("Workspace", "Select the directory for your next new chat.", []tui.SettingsField{{Label: "Project directory", Value: *project}}, false, func(fields []tui.SettingsField) error {
		path, err := filepath.Abs(strings.TrimSpace(fields[0].Value))
		if err != nil {
			return err
		}
		info, err := os.Stat(path)
		if err != nil || !info.IsDir() {
			return fmt.Errorf("project directory must exist")
		}
		if err := core.CheckProjectCredentialBoundary(path); err != nil {
			return err
		}
		*project = path
		return nil
	})
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

func chatFirst(args []string) error {
	defer stopParkedJobs()
	project, sessionID, draft := "", "", ""
	var profile, agent *string
	chatGPTModel := ""
	providerModel := "" // a model from the provider's list, chosen in /models
	sessionOverride := false
	skipChooser := true
	var actionErr error
	info := "" // non-error notice for the next chat screen
	for {
		command := append([]string(nil), args...)
		if project != "" {
			command = append(command, "--project", project)
		}
		if profile != nil {
			command = append(command, "--profile="+*profile)
		}
		if agent != nil {
			command = append(command, "--agent="+*agent)
		}
		if chatGPTModel != "" {
			command = append(command, "--protocol="+core.ChatGPTProtocol, "--model="+chatGPTModel,
				"--base-url=https://api.openai.com/v1", "--endpoint=")
		} else if providerModel != "" {
			command = append(command, "--model="+providerModel)
		}
		if sessionOverride || sessionID != "" || !skipChooser {
			command = append(command, "--session="+sessionID)
		}
		notice := info
		if actionErr != nil {
			notice = actionErr.Error()
		}
		info = ""
		err := runCommand(command, skipChooser, notice, draft)
		// Flags that pick, copy, name or acknowledge the starting session
		// apply once. After any navigation the launcher passes the session
		// itself, so they must not reach the next session too.
		args = withoutStartupFlags(args)
		var nav *tui.ChatNavigation
		if err == nil {
			return nil
		}
		if !errors.As(err, &nav) {
			reason := "Chat is waiting for setup.\n" + err.Error() + "\n\nType /connect to use ChatGPT or configure an API key. Ctrl+P opens commands.\nUse /workspace to select your project, or /settings for other options."
			if actionErr != nil {
				reason = actionErr.Error() + "\n" + reason
			}
			err = tui.RunBlocked("Chat setup required", reason, draft)
			if !errors.As(err, &nav) {
				return err
			}
		}
		if nav.Config != nil || nav.Session != "" {
			sessionID = nav.Session
		}
		sessionOverride = true
		skipChooser = true
		actionErr = nil
		draft = ""
		switch nav.Action {
		case "connect":
			draft = nav.Draft
			connections := []tui.Option{
				{Value: "chatgpt", Label: "Continue with ChatGPT", Description: "Use your eligible ChatGPT plan"},
				{Value: "api", Label: "API key or compatible gateway", Description: "Edit the provider URL and model"},
			}
			_, authErr := core.ChatGPTModel()
			if authErr == nil {
				connections = append(connections, tui.Option{Value: "reauthorize", Label: "Reauthorize ChatGPT", Description: "Refresh browser authorization"})
			}
			connection, chosen, err := tui.SelectOption("Connect a provider", "", connections, nav.Backdrop)
			actionErr = err
			if err != nil || !chosen {
				break
			}
			if connection == "api" {
				var saved bool
				saved, actionErr = configureProvider()
				if saved {
					chatGPTModel, providerModel = "", ""
				}
				break
			}
			if authErr != nil || connection == "reauthorize" {
				if actionErr = chatGPTAuth([]string{"login"}); actionErr != nil {
					break
				}
			}
			current, _ := core.ChatGPTModel()
			selected, chosen, err := selectChatGPTModel(current, nav.Backdrop)
			if err != nil && current != "" {
				chatGPTModel = current
				actionErr = fmt.Errorf("could not refresh the model catalog; using the saved model: %w", err)
			} else {
				actionErr = err
				if chosen {
					chatGPTModel = selected
				}
			}
		case "plan_exit":
			// The plan was approved in chat: same session, build agent.
			draft = nav.Draft
			build := "build"
			agent = &build
			info = "Switched to Build to carry out the approved plan. Press Enter to start."
		case "provider":
			draft = nav.Draft
			var saved bool
			saved, actionErr = configureProvider()
			if saved {
				// The saved route replaces any model chosen in chat.
				chatGPTModel, providerModel = "", ""
			}
		case "settings":
			draft = nav.Draft
			workspace := project
			actionErr = launch(&workspace)
			if workspace != "." && workspace != project {
				project, sessionID = workspace, ""
			}
		case "workspace":
			next := project
			if next == "" {
				next = "."
			}
			var saved bool
			saved, actionErr = editWorkspace(&next)
			if saved {
				project, sessionID = next, ""
			}
		case "sessions":
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
		case "models", "agents", "cycle_agent", "previous_agent":
			draft = nav.Draft
			if nav.Action == "models" && nav.Config != nil && nav.Config.EffectiveProtocol() == core.ChatGPTProtocol {
				var selected string
				var chosen bool
				selected, chosen, actionErr = selectChatGPTModel(nav.Config.Model, nav.Backdrop)
				if chosen {
					chatGPTModel = selected
				}
				break
			}
			cfg := core.Defaults()
			if nav.Config != nil {
				cfg = *nav.Config
			} else if loaded, err := core.LoadConfig(""); err == nil {
				cfg = loaded
			}
			if nav.Action == "cycle_agent" || nav.Action == "previous_agent" {
				selected := nextAgent(cfg, nav.Action == "previous_agent")
				agent = &selected
				break
			}
			title, current, options := "Select model", cfg.Profile, modelOptions(cfg)
			if nav.Action == "models" {
				options = append(options, providerModelOptions(cfg)...)
				if providerModel != "" {
					current = "@model:" + providerModel
				}
			}
			if nav.Action == "agents" {
				title, current, options = "Select agent", cfg.Agent, agentOptions(cfg)
				if current == "" {
					current = "build"
				}
			}
			var selected string
			var chosen bool
			selected, chosen, actionErr = tui.SelectOption(title, current, options, nav.Backdrop)
			if chosen {
				switch {
				case nav.Action == "agents":
					agent = &selected
				case selected == "@provider":
					_, actionErr = configureProvider()
					providerModel = ""
				case strings.HasPrefix(selected, "@model:"):
					providerModel = strings.TrimPrefix(selected, "@model:")
					info = "Model for this chat: " + providerModel + " (set \"model\" in config to keep it)"
				case selected == "@unavailable":
				default:
					profile = &selected
					providerModel = ""
				}
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
			var saved bool
			saved, err = configureProvider()
			if saved {
				status = "Provider settings saved."
			}
		case "config":
			err = editConfiguration()
		case "project":
			var saved bool
			saved, err = editWorkspace(project)
			if saved {
				status = "Workspace: " + *project
			}
		case "chat":
			return nil
		case "command":
			var args []string
			hint := "Enter CLI arguments as a JSON array. Use a command with --help to see its options.\n" + usage
			saved, editErr := tui.EditSettings("Run a command", hint, []tui.SettingsField{{Label: "Arguments", Value: `["sessions", "list"]`}}, false, func(fields []tui.SettingsField) error {
				if err := json.Unmarshal([]byte(fields[0].Value), &args); err != nil || len(args) == 0 {
					return fmt.Errorf("enter a nonempty JSON array of argument strings")
				}
				if args[0] == "setup" || args[0] == "settings" {
					return fmt.Errorf("use the provider or advanced settings menu")
				}
				return nil
			})
			if editErr != nil {
				err = editErr
			} else if saved {
				// Leave the alternate screen so streaming or interactive command
				// output remains usable and visible in the terminal.
				return run(args)
			}
		}
		if err != nil {
			status = err.Error()
		}
	}
}
