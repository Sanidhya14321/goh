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
		next.Endpoint, next.BaseURL = "", ""
		address := strings.TrimSpace(fields[0].Value)
		switch strings.TrimSpace(fields[1].Value) {
		case "base":
			next.BaseURL = address
		case "endpoint":
			next.Endpoint = address
		default:
			return errors.New("URL type must be base or endpoint")
		}
		next.Model = strings.TrimSpace(fields[2].Value)
		next.ContextWindow, err = strconv.Atoi(strings.TrimSpace(fields[3].Value))
		if err != nil {
			return errors.New("Context window must be a whole number")
		}
		next.Protocol = strings.TrimSpace(fields[4].Value)
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
	project, sessionID := "", ""
	skipChooser := true
	var actionErr error
	for {
		command := append([]string(nil), args...)
		if project != "" {
			command = append(command, "--project", project)
		}
		if sessionID != "" || !skipChooser {
			command = append(command, "--session="+sessionID)
		}
		err := runCommand(command, skipChooser)
		var nav *tui.ChatNavigation
		if err == nil {
			return nil
		}
		if !errors.As(err, &nav) {
			reason := "Chat is waiting for setup.\n" + err.Error() + "\n\nPress Ctrl+P or type /provider to configure your API URL, model and key.\nUse /workspace to select your project, or /settings for other options."
			if actionErr != nil {
				reason = actionErr.Error() + "\n" + reason
			}
			err = tui.RunBlocked("Chat setup required", reason)
			if !errors.As(err, &nav) {
				return err
			}
		}
		sessionID = nav.Session
		skipChooser = true
		actionErr = nil
		switch nav.Action {
		case "provider":
			_, actionErr = configureProvider()
		case "settings":
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
			sessionID, skipChooser = "", false
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
