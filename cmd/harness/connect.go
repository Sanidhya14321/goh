package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strconv"
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

const defaultWindow = 32768

var protocolChoices = []string{"chat_completions", "openai_responses", "anthropic_messages"}

var protocolHelp = map[string]string{
	"chat_completions":   "OpenAI-compatible Chat Completions: most gateways and local servers (OpenRouter, vLLM, LM Studio, LiteLLM, Together, Fireworks…).",
	"openai_responses":   "The OpenAI Responses API (api.openai.com or a gateway that implements /responses).",
	"anthropic_messages": "The Anthropic Messages API (api.anthropic.com or a gateway that implements /messages).",
}

// connectOptions are the "add a provider" choices: ChatGPT sign-in, the
// provider catalog, then a custom URL.
func connectOptions(signedIn bool) []tui.Option {
	options := []tui.Option{{Value: "chatgpt", Label: "Continue with ChatGPT", Description: "Sign in with your ChatGPT Plus, Pro or Team plan in the browser"}}
	if signedIn {
		options = append(options,
			tui.Option{Value: "reauthorize", Label: "Reauthorize ChatGPT", Description: "Refresh sign-in for the same account and workspace"},
			tui.Option{Value: "switch", Label: "Use a different ChatGPT account", Description: "Sign in to another account or workspace; replaces the saved one"})
	}
	for _, p := range core.ProviderCatalog {
		options = append(options, tui.Option{Value: "preset:" + p.ID, Label: p.Name, Description: p.Description})
	}
	return append(options, tui.Option{Value: "custom", Label: "Custom provider", Description: "Any OpenAI-, Responses- or Anthropic-compatible URL"})
}

// userSettings reads the user settings file.
func userSettings() (path, raw string, cfg core.Config, err error) {
	if path, err = core.UserConfigPath(); err != nil {
		return
	}
	if raw, err = settingsText(path); err != nil {
		return
	}
	cfg, err = core.DecodeConfig(raw)
	return
}

// checkRoute lists route's models with key to prove the key and URL work.
// p, when it has a name, supplies a key-check endpoint and the filter for
// non-chat models. It is a variable so tests can stand in for the network.
var checkRoute = func(ctx context.Context, route core.Config, key string, p core.ProviderPreset) ([]provider.ModelInfo, error) {
	name := p.Name
	if name == "" {
		u, _ := url.Parse(route.BaseURL + route.Endpoint)
		name = "The provider"
		if u != nil && u.Host != "" {
			name = u.Host
		}
	}
	var infos []provider.ModelInfo
	var err error
	if p.KeyCheckURL != "" {
		err = provider.CheckKey(ctx, route, p.KeyCheckURL, key, &http.Client{})
	}
	if err == nil {
		infos, err = provider.ListModelInfo(ctx, route, key, &http.Client{})
	}
	if errors.Is(err, provider.ErrKeyRejected) {
		return nil, fmt.Errorf("%s did not accept this key; check that it was copied whole and is active", name)
	}
	if err != nil && p.ID != "" && !p.NeedsKey() {
		return nil, fmt.Errorf("%s is not reachable at %s; start it and try again (%v)", name, p.BaseURL, err)
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
		return nil, fmt.Errorf("%s answered but lists no chat models for this key", name)
	}
	return chat, nil
}

// pickModel shows models with the current one highlighted. It returns
// the choice and its reported window (0 if unknown).
func pickModel(title, current string, models []provider.ModelInfo, backdrop string) (string, int, bool, error) {
	options := make([]tui.Option, len(models))
	for i, m := range models {
		description := ""
		if m.Window > 0 {
			description = fmt.Sprintf("%dK context", m.Window/1000)
		}
		options[i] = tui.Option{Value: m.ID, Label: m.ID, Description: description}
	}
	model, chosen, err := tui.SelectOption(title+" (type to search)", current, options, backdrop)
	for _, m := range models {
		if m.ID == model {
			return model, m.Window, chosen, err
		}
	}
	return model, 0, chosen, err
}

// saveProvider checks and writes a provider (and its key, when one was
// entered), optionally making it the one new chats use.
func saveProvider(name string, s core.ProviderSettings, key string, activate bool) error {
	path, raw, _, err := userSettings()
	if err != nil {
		return err
	}
	next, err := core.SaveProvider(raw, name, s, activate)
	if err != nil {
		return err
	}
	route := core.ProviderEntry{Protocol: s.Protocol, Address: s.Address, IsEndpoint: s.IsEndpoint, Model: s.Model, Window: s.Window, KeyEnv: s.KeyEnv}.Route(core.Defaults())
	check := route
	if check.EffectiveProtocol() == core.ChatGPTProtocol && check.ContextWindow == 0 {
		check.ContextWindow = core.DefaultChatGPTWindow // looked up from the account at chat start
	}
	if err := check.Validate(); err != nil {
		return err
	}
	if key != "" {
		if err := route.SaveProviderKey(key); err != nil {
			return err
		}
	}
	return core.WriteUserConfig(path, next)
}

// keyState describes where a provider's key comes from, without the key.
func keyState(e core.ProviderEntry, cfg core.Config) string {
	if e.Protocol == core.ChatGPTProtocol {
		if core.ChatGPTSignedIn() {
			return "signed in"
		}
		return "not signed in"
	}
	switch source := e.Route(cfg).CredentialSource(); {
	case source == "none":
		return "no key"
	case strings.HasPrefix(source, "unavailable"):
		return "key unreadable"
	default:
		return source
	}
}

func providerRows(cfg core.Config) []tui.ProviderRow {
	var rows []tui.ProviderRow
	for _, e := range cfg.Providers() {
		host := e.Address
		if u, err := url.Parse(e.Address); err == nil && u.Host != "" {
			host = u.Host
		}
		model := e.Model
		if model == "" {
			model = "no model chosen"
		}
		title := e.Title()
		if e.Unsaved {
			title += " (from settings, not saved under a name yet)"
		}
		rows = append(rows, tui.ProviderRow{Name: e.Name, Title: title, Detail: model + " · " + host + " · " + keyState(e, cfg), Active: e.Active})
	}
	return rows
}

// manageProviders runs the provider manager until it is closed. It reports
// whether the provider new chats use changed, and a notice for the chat.
func manageProviders(backdrop string) (changed bool, notice string, err error) {
	selected := ""
	for {
		_, _, cfg, err := userSettings()
		if err != nil {
			return changed, "", err
		}
		action, name, err := tui.ManageProviders(providerRows(cfg), selected, notice, backdrop)
		if err != nil || action == "" {
			return changed, "", err
		}
		selected, notice = name, ""
		var actionErr error
		switch action {
		case tui.ProviderUse:
			var used bool
			used, notice, actionErr = useProvider(name)
			if actionErr == nil {
				return changed || used, notice, nil
			}
		case tui.ProviderAdd:
			var added string
			added, notice, actionErr = addProvider(backdrop)
			if actionErr == nil && added != "" {
				return true, notice, nil
			}
		case tui.ProviderModel:
			var active bool
			active, notice, actionErr = changeProviderModel(name, backdrop)
			changed = changed || active
		case tui.ProviderKey:
			notice, actionErr = changeProviderKey(name, backdrop)
		case tui.ProviderTest:
			notice = testProvider(name)
		case tui.ProviderEdit:
			var renamed string
			renamed, notice, actionErr = editProvider(name, backdrop)
			if renamed != "" {
				selected, changed = renamed, true
			}
		case tui.ProviderRemove:
			var active bool
			active, notice, actionErr = removeProvider(name)
			changed = changed || active
		}
		if actionErr != nil {
			notice = actionErr.Error()
		}
	}
}

// adopt saves the hand-configured top-level provider under a name, so it can
// be managed like the others.
func adopt(e core.ProviderEntry, cfg core.Config) (core.ProviderEntry, error) {
	if !e.Unsaved {
		return e, nil
	}
	e.Name, _ = adoptName(e, cfg)
	e.Unsaved = false
	return e, saveProvider(e.Name, settingsOf(e), "", true)
}

func settingsOf(e core.ProviderEntry) core.ProviderSettings {
	return core.ProviderSettings{Protocol: e.Protocol, Address: e.Address, IsEndpoint: e.IsEndpoint, Model: e.Model, Window: e.Window, KeyEnv: e.KeyEnv}
}

func findEntry(name string) (core.ProviderEntry, core.Config, error) {
	_, _, cfg, err := userSettings()
	if err != nil {
		return core.ProviderEntry{}, cfg, err
	}
	e, ok := cfg.FindProvider(name)
	if !ok {
		return e, cfg, fmt.Errorf("no provider named %q", name)
	}
	return e, cfg, nil
}

func useProvider(name string) (bool, string, error) {
	e, _, err := findEntry(name)
	if err != nil {
		return false, "", err
	}
	if e.Active {
		return false, "New chats already use " + name + ".", nil
	}
	path, raw, _, err := userSettings()
	if err != nil {
		return false, "", err
	}
	next, err := core.ActivateProvider(raw, name)
	if err != nil {
		return false, "", err
	}
	if err := core.WriteUserConfig(path, next); err != nil {
		return false, "", err
	}
	return true, "New chats use " + name + " (" + e.Title() + ", " + e.Model + ").", nil
}

// addProvider asks which kind of provider to add and connects it. It
// returns the new provider's name ("" when cancelled).
func addProvider(backdrop string) (string, string, error) {
	choice, chosen, err := tui.SelectOption("Add a provider", "", connectOptions(core.ChatGPTSignedIn()), backdrop)
	if err != nil || !chosen {
		return "", "", err
	}
	switch {
	case choice == "chatgpt" || choice == "reauthorize" || choice == "switch":
		return connectChatGPT(choice, backdrop)
	case strings.HasPrefix(choice, "preset:"):
		return connectPreset(strings.TrimPrefix(choice, "preset:"), backdrop)
	default:
		return customProvider(nil, backdrop)
	}
}

// connectChatGPT signs in (mode "chatgpt" only when not signed in yet,
// "reauthorize" or "switch") and saves ChatGPT as a provider.
func connectChatGPT(mode, backdrop string) (string, string, error) {
	switch {
	case mode == "switch":
		if err := chatGPTAuth([]string{"switch"}); err != nil {
			return "", "", err
		}
	case mode == "reauthorize" || !core.ChatGPTSignedIn():
		if err := chatGPTAuth([]string{"login"}); err != nil {
			return "", "", err
		}
	}
	current, _ := core.ChatGPTModel()
	model, chosen, err := selectChatGPTModel(current, backdrop)
	notice := ""
	if err != nil {
		if current == "" {
			return "", "", err
		}
		model, notice = current, "Could not refresh the ChatGPT model list; using "+current+". "
	} else if !chosen {
		if current == "" {
			return "", "", nil
		}
		model = current
	}
	_, _, cfg, err := userSettings()
	if err != nil {
		return "", "", err
	}
	name := cfg.SuggestProviderName("chatgpt")
	for _, e := range cfg.Providers() {
		if e.Protocol == core.ChatGPTProtocol && !e.Unsaved {
			name = e.Name
		}
	}
	if err := saveProvider(name, core.ProviderSettings{Protocol: core.ChatGPTProtocol, Address: "https://api.openai.com/v1", Model: model}, "", true); err != nil {
		return "", "", err
	}
	return name, notice + "Connected ChatGPT as " + name + "; new chats use " + model + ".", nil
}

// connectPreset sets up a catalog provider: it opens the key page, takes the
// key, checks it with the provider before saving anything, and lets the
// user pick one of the models that key can use.
func connectPreset(id, backdrop string) (string, string, error) {
	p, ok := core.FindPreset(id)
	if !ok {
		return "", "", fmt.Errorf("unknown provider %q", id)
	}
	_, _, cfg, err := userSettings()
	if err != nil {
		return "", "", err
	}
	name := cfg.SuggestProviderName(p.ID)
	current := ""
	for _, e := range cfg.Providers() {
		if e.Preset == p.ID && !e.Unsaved {
			name, current = e.Name, e.Model
		}
	}
	route := p.Apply(core.Defaults())
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	key, models, ok, err := askKey(ctx, "Connect "+p.Name, route, p, true)
	if err != nil || !ok {
		return "", "", err
	}
	model, window, chosen, err := pickModel("Select "+p.Name+" model", current, models, backdrop)
	if err != nil || !chosen {
		return "", "", err
	}
	if window == 0 {
		window = p.Window
	}
	s := core.ProviderSettings{Protocol: p.Protocol, Address: p.BaseURL, Model: model, Window: window}
	if err := saveProvider(name, s, key, true); err != nil {
		return "", "", err
	}
	return name, "Connected " + p.Name + " as " + name + "; new chats use " + model + ".", nil
}

// askKey shows a key form for route and checks the key before accepting it.
// A blank key keeps the one already available (saved or from the
// environment). openPage opens the preset's key page first.
func askKey(ctx context.Context, title string, route core.Config, p core.ProviderPreset, openPage bool) (string, []provider.ModelInfo, bool, error) {
	existing, _ := route.ProviderKey()
	if !p.NeedsKey() && p.ID != "" {
		models, err := checkRoute(ctx, route, "", p)
		return "", models, err == nil, err
	}
	hint := "Paste the API key and press Ctrl+S. GoHarness checks it with the provider before saving anything."
	if p.KeyURL != "" {
		hint = "Create an API key at " + p.KeyURL
		if openPage && openBrowser(ctx, p.KeyURL) {
			hint += " (opened in your browser)"
		}
		hint += ", paste it below and press Ctrl+S. GoHarness checks it with " + p.Name + " before saving anything."
	}
	label := "API key"
	switch source := route.CredentialSource(); {
	case existing == "":
	case strings.HasPrefix(source, "environment "):
		label = "API key (blank uses $" + strings.TrimPrefix(source, "environment ") + ")"
	default:
		label = "API key (blank keeps the saved key)"
	}
	var key string
	var models []provider.ModelInfo
	ok, err := tui.EditSettings(title, hint, []tui.SettingsField{{Label: label, Secret: true}}, false, func(fields []tui.SettingsField) error {
		candidate := strings.TrimSpace(fields[0].Value)
		use := candidate
		if use == "" {
			use = existing
		}
		if use == "" {
			return errors.New("paste an API key first")
		}
		if len(use) > 4096 || strings.ContainsAny(use, "\r\n\x00 ") {
			return errors.New("that does not look like an API key; paste just the key")
		}
		found, err := checkRoute(ctx, route, use, p)
		if err != nil {
			return err
		}
		key, models = candidate, found
		return nil
	})
	return key, models, ok, err
}

// customProvider adds (existing == nil) or edits a provider through one
// form: name, URL, protocol, key, model, context window and an optional key
// variable. The URL and key are checked with the provider before saving;
// a provider without a model list can still be saved with a typed model.
func customProvider(existing *core.ProviderEntry, backdrop string) (string, string, error) {
	_, _, cfg, err := userSettings()
	if err != nil {
		return "", "", err
	}
	name, address, protocol, model, window, keyEnv := cfg.SuggestProviderName("custom"), "", "chat_completions", "", "", ""
	activate := true
	if existing != nil {
		name, address, protocol, model, keyEnv = existing.Name, existing.Address, existing.Protocol, existing.Model, existing.KeyEnv
		if existing.Window > 0 {
			window = strconv.Itoa(existing.Window)
		}
		activate = existing.Active
		if existing.Unsaved {
			adopted, _ := adoptName(*existing, cfg)
			name = adopted
		}
	}
	if !contains(protocolChoices, protocol) {
		protocol = protocolChoices[0]
	}
	fields := []tui.SettingsField{
		{Label: "Name", Value: name, Help: "A short name for this provider, shown in /providers and /models."},
		{Label: "API URL", Value: address, Help: "The API's base URL, e.g. https://llm.example.com/v1 or http://localhost:8000/v1. A full …/chat/completions, …/responses or …/messages URL is used exactly as written."},
		{Label: "Protocol", Value: protocol, Choices: protocolChoices, ChoiceHelp: protocolHelp},
		{Label: "API key", Secret: true, Help: "Blank keeps a saved key. Local servers usually need none."},
		{Label: "Model", Value: model, Help: "Blank: choose from the provider's model list after saving."},
		{Label: "Context window (tokens)", Value: window, Help: "Blank: taken from the provider's model list when it reports one, else 32768."},
		{Label: "Key environment variable (optional)", Value: keyEnv, Help: "Read the key from this variable instead of saving it, e.g. MY_GATEWAY_KEY."},
	}
	title := "Add a custom provider"
	if existing != nil {
		title = "Edit " + existing.Name
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	var s core.ProviderSettings
	var key, newName, warning string
	var models []provider.ModelInfo
	ok, err := tui.EditSettings(title, "Tab moves between fields, ←/→ changes the protocol, Ctrl+S checks the connection and saves.", fields, false, func(f []tui.SettingsField) error {
		newName = strings.TrimSpace(f[0].Value)
		if newName == "" || newName == core.UnsavedProviderName {
			return errors.New("give the provider a name")
		}
		if _, taken := cfg.Profiles[newName]; taken && (existing == nil || existing.Name != newName) {
			return fmt.Errorf("a profile named %q already exists; choose another name", newName)
		}
		address := strings.TrimSpace(f[1].Value)
		if address == "" {
			return errors.New("enter the API URL")
		}
		s = core.ProviderSettings{Protocol: f[2].Value, Address: address, Model: strings.TrimSpace(f[4].Value), KeyEnv: strings.TrimSpace(f[6].Value)}
		for _, suffix := range []string{"/chat/completions", "/responses", "/messages"} {
			if strings.HasSuffix(strings.TrimRight(address, "/"), suffix) {
				s.IsEndpoint = true
			}
		}
		if text := strings.TrimSpace(f[5].Value); text != "" {
			n, err := strconv.Atoi(text)
			if err != nil || n < 2048 {
				return errors.New("context window must be a whole number of at least 2048, or blank")
			}
			s.Window = n
		}
		route := core.ProviderEntry{Protocol: s.Protocol, Address: s.Address, IsEndpoint: s.IsEndpoint, Model: s.Model, KeyEnv: s.KeyEnv}.Route(core.Defaults())
		if _, err := route.ProviderEndpoint(); err != nil {
			return err
		}
		key = strings.TrimSpace(f[3].Value)
		use := key
		if use == "" {
			use, _ = route.ProviderKey()
		}
		found, err := checkRoute(ctx, route, use, core.ProviderPreset{})
		switch {
		case err == nil:
			models, warning = found, ""
		case strings.Contains(err.Error(), "did not accept"):
			return err
		case s.Model == "":
			return fmt.Errorf("could not list models (%v). Check the URL and protocol, or type a model ID to save without checking", err)
		default:
			models, warning = nil, "Saved without checking: the model list was unavailable ("+err.Error()+")."
		}
		return nil
	})
	if err != nil || !ok {
		return "", "", err
	}
	if s.Model == "" {
		model, w, chosen, err := pickModel("Select model for "+newName, "", models, backdrop)
		if err != nil || !chosen {
			return "", "", err
		}
		s.Model = model
		if s.Window == 0 {
			s.Window = w
		}
	} else if s.Window == 0 {
		for _, m := range models {
			if m.ID == s.Model {
				s.Window = m.Window
			}
		}
	}
	if s.Window == 0 {
		s.Window = defaultWindow
	}
	if models != nil && warning == "" {
		listed := false
		for _, m := range models {
			listed = listed || m.ID == s.Model
		}
		if !listed {
			warning = "Note: " + s.Model + " is not in the provider's model list. "
		}
	}
	if err := saveProvider(newName, s, key, activate); err != nil {
		return "", "", err
	}
	if existing != nil && !existing.Unsaved && existing.Name != newName {
		if path, raw, _, err := userSettings(); err == nil {
			if next, err := core.RemoveProvider(raw, existing.Name, false, ""); err == nil {
				_ = core.WriteUserConfig(path, next)
			}
		}
	}
	verb := "Added"
	if existing != nil {
		verb = "Saved"
	}
	return newName, warning + verb + " " + newName + " (" + s.Model + ").", nil
}

// adoptName suggests a name for a hand-configured provider: its catalog
// ID, else the distinctive part of its host (gateway for
// api.gateway.example), else "custom".
func adoptName(e core.ProviderEntry, cfg core.Config) (string, error) {
	base := e.Preset
	if base == "" {
		base = "custom"
		if u, err := url.Parse(e.Address); err == nil && u.Hostname() != "" {
			parts := strings.Split(u.Hostname(), ".")
			base = parts[0]
			if base == "api" && len(parts) > 1 {
				base = parts[1]
			}
		}
	}
	return cfg.SuggestProviderName(base), nil
}

func contains(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

// editProvider edits a provider: catalog providers and ChatGPT change their
// model or key (their URL is fixed); custom ones open the full form.
func editProvider(name, backdrop string) (string, string, error) {
	e, _, err := findEntry(name)
	if err != nil {
		return "", "", err
	}
	if e.Preset != "" && !e.Unsaved {
		return "", e.Title() + " has a fixed URL: use m to change its model or k to change its key.", nil
	}
	return customProvider(&e, backdrop)
}

// changeProviderModel picks another model from the provider's list.
func changeProviderModel(name, backdrop string) (bool, string, error) {
	e, cfg, err := findEntry(name)
	if err != nil {
		return false, "", err
	}
	if e, err = adopt(e, cfg); err != nil {
		return false, "", err
	}
	var model string
	var window int
	var chosen bool
	if e.Protocol == core.ChatGPTProtocol {
		model, chosen, err = selectChatGPTModel(e.Model, backdrop)
	} else {
		p, _ := core.FindPreset(e.Preset)
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()
		route := e.Route(cfg)
		key, _ := route.ProviderKey()
		var models []provider.ModelInfo
		if models, err = checkRoute(ctx, route, key, p); err != nil {
			return false, "", fmt.Errorf("could not list %s models: %w", e.Name, err)
		}
		model, window, chosen, err = pickModel("Select "+e.Name+" model", e.Model, models, backdrop)
	}
	if err != nil || !chosen {
		return false, "", err
	}
	s := settingsOf(e)
	s.Model = model
	if window > 0 {
		s.Window = window
	}
	if err := saveProvider(e.Name, s, "", e.Active); err != nil {
		return false, "", err
	}
	return e.Active, e.Name + " now uses " + model + ".", nil
}

// changeProviderKey replaces a provider's key after checking it, or for
// ChatGPT signs in again.
func changeProviderKey(name, backdrop string) (string, error) {
	e, cfg, err := findEntry(name)
	if err != nil {
		return "", err
	}
	if e.Protocol == core.ChatGPTProtocol {
		options := []tui.Option{
			{Value: "reauthorize", Label: "Reauthorize ChatGPT", Description: "Same account and workspace"},
			{Value: "switch", Label: "Use a different ChatGPT account", Description: "Another account or workspace"},
		}
		mode, chosen, err := tui.SelectOption("ChatGPT sign-in", "", options, backdrop)
		if err != nil || !chosen {
			return "", err
		}
		if err := chatGPTAuth([]string{map[string]string{"reauthorize": "login", "switch": "switch"}[mode]}); err != nil {
			return "", err
		}
		return "ChatGPT sign-in updated.", nil
	}
	if e, err = adopt(e, cfg); err != nil {
		return "", err
	}
	p, _ := core.FindPreset(e.Preset)
	if p.ID == "" {
		p = core.ProviderPreset{Name: e.Name}
	} else if !p.NeedsKey() {
		return p.Name + " needs no key.", nil
	}
	route := e.Route(cfg)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	key, _, ok, err := askKey(ctx, "API key for "+e.Name, route, p, p.KeyURL != "")
	if err != nil || !ok {
		return "", err
	}
	if key == "" {
		return "Key for " + e.Name + " unchanged; it works.", nil
	}
	if err := route.SaveProviderKey(key); err != nil {
		return "", err
	}
	return "Saved the new key for " + e.Name + "; it works.", nil
}

// testProvider checks that the provider answers with its saved key.
func testProvider(name string) string {
	e, cfg, err := findEntry(name)
	if err != nil {
		return err.Error()
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if e.Protocol == core.ChatGPTProtocol {
		models, err := core.ListChatGPTModels(ctx)
		if err != nil {
			return "✗ ChatGPT: " + err.Error()
		}
		return fmt.Sprintf("✓ ChatGPT answered: %d models available to this account.", len(models))
	}
	p, _ := core.FindPreset(e.Preset)
	route := e.Route(cfg)
	key, err := route.ProviderKey()
	if err != nil {
		return "✗ " + e.Name + ": " + err.Error()
	}
	models, err := checkRoute(ctx, route, key, p)
	if err != nil {
		return "✗ " + e.Name + ": " + err.Error()
	}
	listed := false
	for _, m := range models {
		listed = listed || m.ID == e.Model
	}
	result := fmt.Sprintf("✓ %s answered: %d models available with this key.", e.Name, len(models))
	if e.Model != "" && !listed {
		result += " " + e.Model + " is not among them."
	}
	return result
}

// removeProvider deletes a provider, and its saved key when no other
// provider shares the same URL. A removed active provider hands over to the
// first remaining one.
func removeProvider(name string) (bool, string, error) {
	e, cfg, err := findEntry(name)
	if err != nil {
		return false, "", err
	}
	next := ""
	shared := false
	for _, other := range cfg.Providers() {
		if other.Name == name {
			continue
		}
		if next == "" && !other.Unsaved {
			next = other.Name
		}
		shared = shared || other.Protocol == e.Protocol && other.Address == e.Address
	}
	path, raw, _, err := userSettings()
	if err != nil {
		return false, "", err
	}
	updated, err := core.RemoveProvider(raw, name, e.Active, next)
	if err != nil {
		return false, "", err
	}
	if err := core.WriteUserConfig(path, updated); err != nil {
		return false, "", err
	}
	notice := "Removed " + name + "."
	if e.Protocol == core.ChatGPTProtocol {
		notice += " The ChatGPT sign-in is kept; `goh auth logout` signs out."
	} else if !shared {
		if err := e.Route(cfg).SaveProviderKey(""); err == nil {
			notice += " Its saved key was deleted."
		}
	}
	if e.Active && next != "" {
		notice += " New chats use " + next + "."
	} else if e.Active {
		notice += " No provider is left; add one to chat."
	}
	return e.Active, notice, nil
}

// runProviders implements `goh providers`: without arguments it opens the
// manager; list, use NAME and remove NAME work without a terminal UI.
func runProviders(args []string, w io.Writer) error {
	usage := errors.New("usage: goh providers [list | use NAME | remove NAME]")
	if len(args) == 0 {
		_, notice, err := manageProviders("")
		if notice != "" {
			fmt.Fprintln(w, notice)
		}
		return err
	}
	switch {
	case args[0] == "list" && len(args) == 1:
		_, _, cfg, err := userSettings()
		if err != nil {
			return err
		}
		entries := cfg.Providers()
		if len(entries) == 0 {
			fmt.Fprintln(w, "No providers yet. Run `goh providers` or /connect in chat to add one.")
			return nil
		}
		for _, e := range entries {
			mark := " "
			if e.Active {
				mark = "*"
			}
			fmt.Fprintf(w, "%s %-16s %-24s %-28s %s\n", mark, tui.Safe(e.Name), tui.Safe(e.Title()), tui.Safe(e.Model), tui.Safe(keyState(e, cfg)))
		}
		fmt.Fprintln(w, "* = used by new chats")
		return nil
	case args[0] == "use" && len(args) == 2:
		_, notice, err := useProvider(args[1])
		if err == nil {
			fmt.Fprintln(w, notice)
		}
		return err
	case args[0] == "remove" && len(args) == 2:
		_, notice, err := removeProvider(args[1])
		if err == nil {
			fmt.Fprintln(w, notice)
		}
		return err
	}
	return usage
}
