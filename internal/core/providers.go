package core

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// Connected providers are model profiles that carry their own route
// (protocol plus base_url or endpoint). The active provider is also copied
// into the top-level settings, so everything that reads the configuration
// keeps working unchanged; switching providers copies another profile up.
// Keys are saved per protocol and URL, so each provider keeps its own.

// ProviderEntry is one connected provider, as the provider manager shows it.
type ProviderEntry struct {
	Name       string
	Protocol   string
	Address    string
	IsEndpoint bool
	Model      string
	Window     int
	KeyEnv     string
	Active     bool
	// Preset is the catalog ID, "chatgpt", or "" for a custom URL.
	Preset string
	// Unsaved marks a top-level provider configured before the provider
	// manager existed (or by hand) and not saved under a name yet.
	Unsaved bool
}

// ProviderSettings describes a provider to save.
type ProviderSettings struct {
	Protocol   string
	Address    string
	IsEndpoint bool
	Model      string
	Window     int
	KeyEnv     string
}

// UnsavedProviderName names the entry for a provider set only at the top
// level of the settings.
const UnsavedProviderName = "current"

// Title is a readable provider name.
func (e ProviderEntry) Title() string {
	if e.Preset == "chatgpt" {
		return "ChatGPT"
	}
	if p, ok := FindPreset(e.Preset); ok {
		return p.Name
	}
	return "Custom"
}

// Route is the provider as a configuration.
func (e ProviderEntry) Route(base Config) Config {
	c := base
	endpoint, baseURL := "", e.Address
	if e.IsEndpoint {
		endpoint, baseURL = e.Address, ""
	}
	c.SetProviderRoute(e.Protocol, endpoint, baseURL)
	c.Model, c.ContextWindow, c.APIKeyEnv = e.Model, e.Window, e.KeyEnv
	return c
}

func presetFor(protocol, address string) string {
	if protocol == ChatGPTProtocol {
		return "chatgpt"
	}
	for _, p := range ProviderCatalog {
		if p.Protocol == protocol && strings.TrimRight(p.BaseURL, "/") == strings.TrimRight(address, "/") {
			return p.ID
		}
	}
	return ""
}

func deref[T any](p *T) T {
	var zero T
	if p == nil {
		return zero
	}
	return *p
}

// Providers lists the connected providers, sorted by name, marking the one
// the top-level settings use.
func (c Config) Providers() []ProviderEntry {
	top := ProviderEntry{Protocol: c.EffectiveProtocol(), Address: c.BaseURL, Model: c.Model, Window: c.ContextWindow, KeyEnv: c.APIKeyEnv}
	if c.Endpoint != "" {
		top.Address, top.IsEndpoint = c.Endpoint, true
	}
	if top.Protocol == ChatGPTProtocol && top.Address == "" {
		top.Address = chatGPTResource
	}
	configured := c.Model != "" || c.Protocol != "" || c.BaseURL != "" || c.Endpoint != ""
	var out []ProviderEntry
	activeFound := false
	for name, p := range c.Profiles {
		if p.Protocol == nil && p.BaseURL == nil && p.Endpoint == nil {
			continue // a model-only profile, shown in /models
		}
		e := ProviderEntry{Name: name, Protocol: deref(p.Protocol), Address: deref(p.BaseURL), Model: deref(p.Model),
			Window: deref(p.ContextWindow), KeyEnv: deref(p.APIKeyEnv)}
		if e.Protocol == "" {
			e.Protocol = "chat_completions"
		}
		if p.Endpoint != nil && *p.Endpoint != "" {
			e.Address, e.IsEndpoint = *p.Endpoint, true
		}
		if e.Protocol == ChatGPTProtocol && e.Address == "" {
			e.Address = chatGPTResource
		}
		e.Preset = presetFor(e.Protocol, e.Address)
		e.Active = configured && !activeFound && e.Protocol == top.Protocol && e.Address == top.Address &&
			e.IsEndpoint == top.IsEndpoint && (e.Model == top.Model || top.Model == "")
		activeFound = activeFound || e.Active
		out = append(out, e)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	if configured && !activeFound && (top.Address != "" || top.Protocol == ChatGPTProtocol) {
		top.Name, top.Active, top.Unsaved = UnsavedProviderName, true, true
		top.Preset = presetFor(top.Protocol, top.Address)
		out = append([]ProviderEntry{top}, out...)
	}
	return out
}

// FindProvider returns the provider with this name.
func (c Config) FindProvider(name string) (ProviderEntry, bool) {
	for _, e := range c.Providers() {
		if e.Name == name {
			return e, true
		}
	}
	return ProviderEntry{}, false
}

var providerNameChars = regexp.MustCompile(`[^a-z0-9_-]+`)

// SuggestProviderName derives a free profile name from base, adding -2, -3…
// when it is taken.
func (c Config) SuggestProviderName(base string) string {
	base = strings.Trim(providerNameChars.ReplaceAllString(strings.ToLower(base), "-"), "-_")
	if base == "" || base == UnsavedProviderName {
		base = "provider"
	}
	if len(base) > 48 {
		base = base[:48]
	}
	name := base
	for i := 2; ; i++ {
		if _, taken := c.Profiles[name]; !taken {
			return name
		}
		name = fmt.Sprintf("%s-%d", base, i)
	}
}

// editDocument applies change to the user settings JSON, keeping every other
// setting as written. Settings with comments or trailing commas are refused,
// since rewriting them would drop the comments.
func editDocument(raw string, change func(doc map[string]json.RawMessage, profiles map[string]map[string]json.RawMessage) error) (string, error) {
	if HasJSONCSyntax(raw) {
		return "", errors.New("the settings file has comments or trailing commas, which the provider manager would erase; edit it in Advanced configuration (or by hand) instead")
	}
	doc := map[string]json.RawMessage{}
	if strings.TrimSpace(raw) != "" {
		if err := json.Unmarshal([]byte(raw), &doc); err != nil {
			return "", err
		}
		if doc == nil {
			doc = map[string]json.RawMessage{}
		}
	}
	profiles := map[string]map[string]json.RawMessage{}
	if b, ok := doc["profiles"]; ok {
		if err := json.Unmarshal(b, &profiles); err != nil {
			return "", fmt.Errorf("profiles: %w", err)
		}
	}
	if err := change(doc, profiles); err != nil {
		return "", err
	}
	if len(profiles) > 0 {
		doc["profiles"], _ = json.Marshal(profiles)
	} else {
		delete(doc, "profiles")
	}
	out, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return "", err
	}
	next := string(out) + "\n"
	if _, err := DecodeConfig(next); err != nil {
		return "", err
	}
	return next, nil
}

func routeFields(s ProviderSettings) map[string]json.RawMessage {
	m := map[string]json.RawMessage{}
	put := func(k string, v any) { m[k], _ = json.Marshal(v) }
	put("protocol", s.Protocol)
	if s.IsEndpoint {
		put("endpoint", s.Address)
	} else {
		put("base_url", s.Address)
	}
	if s.Model != "" {
		put("model", s.Model)
	}
	if s.Window > 0 {
		put("context_window", s.Window)
	}
	if s.KeyEnv != "" {
		put("api_key_env", s.KeyEnv)
	}
	return m
}

// setTop makes the top-level settings use route: its fields replace the
// previous provider's, and fields it does not set are removed.
func setTop(doc map[string]json.RawMessage, route map[string]json.RawMessage) {
	for _, k := range []string{"protocol", "base_url", "endpoint", "model", "context_window", "api_key_env"} {
		delete(doc, k)
		if v, ok := route[k]; ok {
			doc[k] = v
		}
	}
}

// SaveProvider saves s under name, replacing that provider's route but
// keeping its other profile settings (an output budget, say). With activate
// the top-level settings switch to it.
func SaveProvider(raw, name string, s ProviderSettings, activate bool) (string, error) {
	if !profileName.MatchString(name) || name == UnsavedProviderName {
		return "", fmt.Errorf("provider name %q must be letters, digits, - or _ (not %q)", name, UnsavedProviderName)
	}
	return editDocument(raw, func(doc map[string]json.RawMessage, profiles map[string]map[string]json.RawMessage) error {
		route := routeFields(s)
		p := profiles[name]
		if p == nil {
			p = map[string]json.RawMessage{}
		}
		for _, k := range []string{"protocol", "base_url", "endpoint", "model", "context_window", "api_key_env"} {
			delete(p, k)
			if v, ok := route[k]; ok {
				p[k] = v
			}
		}
		profiles[name] = p
		if activate {
			setTop(doc, route)
			delete(doc, "profile")
		}
		return nil
	})
}

// ActivateProvider switches the top-level settings to a saved provider.
func ActivateProvider(raw, name string) (string, error) {
	return editDocument(raw, func(doc map[string]json.RawMessage, profiles map[string]map[string]json.RawMessage) error {
		p, ok := profiles[name]
		if !ok {
			return fmt.Errorf("no provider named %q", name)
		}
		if _, ok := p["base_url"]; !ok {
			if _, ok := p["endpoint"]; !ok {
				if _, ok := p["protocol"]; !ok {
					return fmt.Errorf("%q is a model profile, not a provider", name)
				}
			}
		}
		setTop(doc, p)
		delete(doc, "profile")
		return nil
	})
}

// RemoveProvider deletes a saved provider. When it is the active one the
// top-level settings move to next (another provider's name), or are
// cleared when next is "". A provider that an agent, model role or fallback
// list names is refused, since removing it would break those settings.
func RemoveProvider(raw, name string, active bool, next string) (string, error) {
	if name != UnsavedProviderName {
		if c, err := DecodeConfig(raw); err == nil {
			for agent, a := range c.Agents {
				if a.Profile == name {
					return "", fmt.Errorf("agent %q uses %q; change the agent's profile first", agent, name)
				}
			}
			for role, profile := range c.ModelRoles {
				if profile == name {
					return "", fmt.Errorf("model role %q uses %q; change it first", role, name)
				}
			}
			for _, profile := range c.FallbackProfiles {
				if profile == name {
					return "", fmt.Errorf("fallback_profiles lists %q; remove it there first", name)
				}
			}
		}
	}
	return editDocument(raw, func(doc map[string]json.RawMessage, profiles map[string]map[string]json.RawMessage) error {
		if name != UnsavedProviderName {
			if _, ok := profiles[name]; !ok {
				return fmt.Errorf("no provider named %q", name)
			}
			delete(profiles, name)
		}
		if active || name == UnsavedProviderName {
			setTop(doc, profiles[next])
		}
		if b, ok := doc["profile"]; ok {
			var selected string
			if json.Unmarshal(b, &selected) == nil && selected == name {
				delete(doc, "profile")
			}
		}
		return nil
	})
}
