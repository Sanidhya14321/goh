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
		case "compaction", "task", "explore":
		default:
			return fmt.Errorf("unsupported model role %q (use compaction, task or explore)", role)
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
