package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"example.com/independent-harness/internal/core"
	"example.com/independent-harness/internal/session"
	"example.com/independent-harness/internal/tui"
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

func chatFirst(args []string) error {
	project, sessionID, draft := "", "", ""
	var profile, agent *string
	sessionOverride := false
	skipChooser := true
	var actionErr error
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
		if sessionOverride || sessionID != "" || !skipChooser {
			command = append(command, "--session="+sessionID)
		}
		notice := ""
		if actionErr != nil {
			notice = actionErr.Error()
		}
		err := runCommand(command, skipChooser, notice, draft)
		var nav *tui.ChatNavigation
		if err == nil {
			return nil
		}
		if !errors.As(err, &nav) {
			reason := "Chat is waiting for setup.\n" + err.Error() + "\n\nPress Ctrl+P or type /provider to configure your API URL, model and key.\nUse /workspace to select your project, or /settings for other options."
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
		case "provider":
			draft = nav.Draft
			_, actionErr = configureProvider()
		case "settings":
			draft = nav.Draft
			actionErr = launch()
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
			infos, actionErr = session.List(cfg.SessionDir)
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
			title, current, options := "Select model profile", cfg.Profile, modelOptions(cfg)
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
				default:
					profile = &selected
				}
			}
		}
	}
}

func launch() error {
	var status string
	project := "."
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
			saved, err = editWorkspace(&project)
			if saved {
				status = "Workspace: " + project
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
