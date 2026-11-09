package core

import "fmt"

func (c Config) EffectiveProtocol() string {
	if c.Protocol == "" {
		return "chat_completions"
	}
	return c.Protocol
}
func (c Config) CredentialEnvironment() string {
	if c.APIKeyEnv != "" {
		return c.APIKeyEnv
	}
	if c.EffectiveProtocol() == "anthropic_messages" {
		return "ANTHROPIC_API_KEY"
	}
	return "OPENAI_API_KEY"
}
func (c Config) ValidateProtocol() error {
	switch c.EffectiveProtocol() {
	case "chat_completions", "anthropic_messages", "openai_responses", ChatGPTProtocol:
		return nil
	default:
		return fmt.Errorf("protocol must be chat_completions, anthropic_messages, openai_responses or openai_chatgpt")
	}
}
func (c Config) ValidateModelRoles() error {
	for role, profile := range c.ModelRoles {
		switch role {
		case "compaction", "task", "explore", "title":
		default:
			return fmt.Errorf("unsupported model role %q (use compaction, task, explore or title)", role)
		}
		if !profileName.MatchString(profile) {
			return fmt.Errorf("model role %s needs a named profile", role)
		}
		if _, ok := c.Profiles[profile]; !ok {
			return fmt.Errorf("model role %s references unknown profile %q", role, profile)
		}
	}
	return nil
}

// SetProviderRoute drops inherited gateway credentials when the request binding
// changes. Callers can then apply a credential explicitly chosen for that route.
func (c *Config) SetProviderRoute(protocol, endpoint, baseURL string) {
	oldEndpoint, _ := c.ProviderEndpoint()
	oldProtocol := c.EffectiveProtocol()
	c.Protocol, c.Endpoint, c.BaseURL = protocol, endpoint, baseURL
	newEndpoint, _ := c.ProviderEndpoint()
	if oldProtocol != c.EffectiveProtocol() || oldEndpoint != newEndpoint {
		c.APIKeyEnv = ""
	}
}

func (c *Config) overlayProfile(p ModelProfile) {
	if p.Protocol != nil || p.Endpoint != nil || p.BaseURL != nil {
		protocol, endpoint, baseURL := c.Protocol, c.Endpoint, c.BaseURL
		if p.Protocol != nil {
			protocol = *p.Protocol
		}
		if p.Endpoint != nil || p.BaseURL != nil {
			endpoint, baseURL = "", ""
			if p.Endpoint != nil {
				endpoint = *p.Endpoint
			}
			if p.BaseURL != nil {
				baseURL = *p.BaseURL
			}
		}
		c.SetProviderRoute(protocol, endpoint, baseURL)
	}
	if p.Model != nil {
		c.Model = *p.Model
	}
	if p.APIKeyEnv != nil {
		c.APIKeyEnv = *p.APIKeyEnv
	}
	if p.ContextWindow != nil {
		c.ContextWindow = *p.ContextWindow
	}
	if p.OutputBudget != nil {
		c.OutputBudget = *p.OutputBudget
	}
	if p.OutputParameter != nil {
		c.OutputParameter = *p.OutputParameter
	}
}

// WithModelRole overlays model settings only. Permissions, agent mode, tools,
// session location and history ownership do not change. Unset roles inherit.
func (c Config) WithModelRole(role string) (Config, error) {
	if err := c.ValidateModelRoles(); err != nil {
		return c, err
	}
	name, ok := c.ModelRoles[role]
	if !ok {
		return c, nil
	}
	c.Profile = name
	c.overlayProfile(c.Profiles[name])
	if err := c.applyEnvironment(); err != nil {
		return c, err
	}
	return c, c.ValidateProtocol()
}

// ValidateFallbackProfiles checks the explicit, ordered failover list. Only
// profiles the user named here may receive a conversation after a failure.
func (c Config) ValidateFallbackProfiles() error {
	if len(c.FallbackProfiles) > 8 {
		return fmt.Errorf("fallback_profiles allows at most 8 profiles")
	}
	seen := map[string]bool{}
	for _, name := range c.FallbackProfiles {
		if !profileName.MatchString(name) {
			return fmt.Errorf("fallback profile %q is not a valid profile name", name)
		}
		if _, ok := c.Profiles[name]; !ok {
			return fmt.Errorf("fallback_profiles references unknown profile %q", name)
		}
		if seen[name] {
			return fmt.Errorf("fallback profile %q is listed twice", name)
		}
		seen[name] = true
	}
	return nil
}

// WithFallbackProfile overlays one explicitly listed fallback profile. Like
// model roles, it changes model routing only, never permissions or history.
func (c Config) WithFallbackProfile(name string) (Config, error) {
	if err := c.ValidateFallbackProfiles(); err != nil {
		return c, err
	}
	listed := false
	for _, n := range c.FallbackProfiles {
		listed = listed || n == name
	}
	if !listed {
		return c, fmt.Errorf("profile %q is not listed in fallback_profiles", name)
	}
	c.Profile = name
	c.overlayProfile(c.Profiles[name])
	if err := c.applyEnvironment(); err != nil {
		return c, err
	}
	return c, c.Validate()
}
