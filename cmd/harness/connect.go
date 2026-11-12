package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/Sanidhya14321/goh/internal/core"
	"github.com/Sanidhya14321/goh/internal/provider"
	"github.com/Sanidhya14321/goh/internal/tui"
)

// openBrowser shows address in the default browser, best effort. It
// reports whether a browser was started.
// GOH_NO_BROWSER=1, or an SSH session, only shows the link.
func openBrowser(ctx context.Context, address string) bool {
	if os.Getenv("GOH_NO_BROWSER") != "" || os.Getenv("SSH_CONNECTION") != "" {
		return false
	}
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.CommandContext(ctx, "open", address)
	case "linux":
		cmd = exec.CommandContext(ctx, "xdg-open", address)
	case "windows":
		// rundll32 takes the URL as one argument; cmd /c start would
		// reinterpret & and other metacharacters in it.
		cmd = exec.CommandContext(ctx, "rundll32", "url.dll,FileProtocolHandler", address)
	default:
		return false
	}
	return cmd.Run() == nil
}

// connectOptions are the /connect choices: ChatGPT sign-in, the provider
// catalog, then a custom URL.
func connectOptions(signedIn bool) []tui.Option {
	options := []tui.Option{{Value: "chatgpt", Label: "Continue with ChatGPT", Description: "Sign in with your ChatGPT Plus, Pro or Team plan in the browser"}}
	if signedIn {
		options = append(options, tui.Option{Value: "reauthorize", Label: "Reauthorize ChatGPT", Description: "Refresh browser authorization"})
	}
	for _, p := range core.ProviderCatalog {
		options = append(options, tui.Option{Value: "preset:" + p.ID, Label: p.Name, Description: p.Description})
	}
	return append(options, tui.Option{Value: "api", Label: "API key or compatible gateway", Description: "Any other URL: edit the address, protocol and model by hand"})
}

// verifyKey lists the preset's models with key, keeping only chat models.
// It is a variable so tests can stand in for the network.
var verifyKey = func(ctx context.Context, p core.ProviderPreset, c core.Config, key string) ([]provider.ModelInfo, error) {
	var infos []provider.ModelInfo
	var err error
	if p.KeyCheckURL != "" {
		err = provider.CheckKey(ctx, c, p.KeyCheckURL, key, &http.Client{})
	}
	if err == nil {
		infos, err = provider.ListModelInfo(ctx, c, key, &http.Client{})
	}
	if errors.Is(err, provider.ErrKeyRejected) {
		return nil, fmt.Errorf("%s did not accept this key; check that it was copied whole and is active", p.Name)
	}
	if err != nil && !p.NeedsKey() {
		return nil, fmt.Errorf("%s is not reachable at %s; start it and try again (%v)", p.Name, p.BaseURL, err)
	}
	if err != nil {
		return nil, err
	}
	var chat []provider.ModelInfo
	for _, m := range infos {
		if id, ok := p.ChatModel(m.ID); ok {
			chat = append(chat, provider.ModelInfo{ID: id, Window: m.Window})
		}
	}
	if len(chat) == 0 {
		return nil, fmt.Errorf("%s accepted the key but lists no chat models for it", p.Name)
	}
	return chat, nil
}

// connectPreset sets up a catalog provider: it opens the key page, takes the
// key, checks it against the provider's model list before saving anything,
// and lets the user pick one of the models that key can use.
func connectPreset(id, backdrop string) (bool, error) {
	p, ok := core.FindPreset(id)
	if !ok {
		return false, fmt.Errorf("unknown provider %q", id)
	}
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
	next := p.Apply(cfg)
	saved, _ := next.ProviderKey()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	var models []provider.ModelInfo
	key := ""
	if !p.NeedsKey() {
		if models, err = verifyKey(ctx, p, next, ""); err != nil {
			return false, err
		}
	} else {
		opened := openBrowser(ctx, p.KeyURL)
		hint := "Create an API key at " + p.KeyURL
		if opened {
			hint += " (opened in your browser)"
		}
		hint += ", paste it below and press Ctrl+S. GoHarness checks it with " + p.Name + " before saving anything."
		label := "API key"
		switch source := next.CredentialSource(); {
		case saved == "":
		case strings.HasPrefix(source, "environment "):
			label = "API key (blank uses $" + strings.TrimPrefix(source, "environment ") + ")"
		default:
			label = "API key (blank keeps the saved key)"
		}
		ok, err := tui.EditSettings("Connect "+p.Name, hint, []tui.SettingsField{{Label: label, Secret: true}}, false, func(fields []tui.SettingsField) error {
			candidate := strings.TrimSpace(fields[0].Value)
			if candidate == "" {
				candidate = saved
			}
			if candidate == "" {
				return errors.New("paste an API key first")
			}
			if len(candidate) > 4096 || strings.ContainsAny(candidate, "\r\n\x00 ") {
				return errors.New("that does not look like an API key; paste just the key")
			}
			found, err := verifyKey(ctx, p, next, candidate)
			if err != nil {
				return err
			}
			key, models = candidate, found
			return nil
		})
		if err != nil || !ok {
			return false, err
		}
	}

	options := make([]tui.Option, len(models))
	current := ""
	for i, m := range models {
		description := ""
		if m.Window > 0 {
			description = fmt.Sprintf("%dK context", m.Window/1000)
		}
		options[i] = tui.Option{Value: m.ID, Label: m.ID, Description: description}
		if m.ID == cfg.Model {
			current = m.ID
		}
	}
	model, chosen, err := tui.SelectOption("Select "+p.Name+" model (type to search)", current, options, backdrop)
	if err != nil || !chosen {
		return false, err
	}
	next.Model = model
	next.ContextWindow = p.Window
	for _, m := range models {
		if m.ID == model && m.Window > 0 {
			next.ContextWindow = m.Window
		}
	}
	return true, savePreset(path, raw, cfg, next, p, key)
}

// savePreset writes the verified key and the provider settings.
func savePreset(path, raw string, cfg, next core.Config, p core.ProviderPreset, key string) error {
	if err := next.Validate(); err != nil {
		return err
	}
	data, err := providerDocument(path, raw, cfg, next, p.BaseURL)
	if err != nil {
		return err
	}
	if key != "" {
		if err := next.SaveProviderKey(key); err != nil {
			return err
		}
	}
	return core.WriteUserConfig(path, data)
}
