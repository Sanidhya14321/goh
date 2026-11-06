package core

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
)

// ModelProfile contains only credential environment-variable names, never keys.
// Pointers distinguish an absent numeric override from an explicitly invalid zero.
type ModelProfile struct {
	Protocol        *string `json:"protocol,omitempty"`
	Endpoint        *string `json:"endpoint,omitempty"`
	BaseURL         *string `json:"base_url,omitempty"`
	Model           *string `json:"model,omitempty"`
	APIKeyEnv       *string `json:"api_key_env,omitempty"`
	ContextWindow   *int    `json:"context_window,omitempty"`
	OutputBudget    *int    `json:"output_budget,omitempty"`
	OutputParameter *string `json:"output_parameter,omitempty"`
}
type AgentProfile struct {
	Description  string `json:"description"`
	Instructions string `json:"instructions"`
	Mode         string `json:"mode"`
	Profile      string `json:"profile,omitempty"`
}

var profileName = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]{0,63}$`)

// Resolve selects explicit user-configured profiles. Precedence is defaults,
// base configuration, selected agent/model profile, supported environment, then
// caller-applied CLI overrides. Profile/agent instructions never grant authority.
func (c Config) Resolve(profile, agent string) (Config, error) {
	if err := c.ValidateModelRoles(); err != nil {
		return c, err
	}
	if profile == "" {
		profile = c.Profile
	}
	if agent == "" {
		agent = c.Agent
	}
	if agent == "" {
		agent = "build"
	}
	for name := range c.Profiles {
		if !profileName.MatchString(name) {
			return c, fmt.Errorf("invalid model profile name %q", name)
		}
	}
	for name, p := range c.Agents {
		if !profileName.MatchString(name) || name == "build" || name == "plan" {
			return c, fmt.Errorf("custom agent %q must have a unique name distinct from build/plan", name)
		}
		if p.Mode != "build" && p.Mode != "plan" {
			return c, fmt.Errorf("agent %q mode must be build or plan", name)
		}
		if len(p.Description) > 8192 || len(p.Instructions) > 64<<10 {
			return c, fmt.Errorf("agent %q instructions or description exceed limits", name)
		}
		if p.Profile != "" {
			if _, ok := c.Profiles[p.Profile]; !ok {
				return c, fmt.Errorf("agent %q references unknown profile %q", name, p.Profile)
			}
		}
	}
	c.Agent = agent
	c.AgentMode = agent
	c.AgentInstructions = ""
	c.AgentDescription = ""
	switch agent {
	case "build":
		c.AgentDescription = "Investigate and change code with explicit permission for effects."
	case "plan":
		c.AgentDescription = "Investigate and plan using read-only project tools."
	default:
		p, ok := c.Agents[agent]
		if !ok {
			return c, fmt.Errorf("unknown agent %q", agent)
		}
		c.AgentMode = p.Mode
		c.AgentInstructions = p.Instructions
		c.AgentDescription = p.Description
		if profile == "" {
			profile = p.Profile
		}
	}
	c.Profile = profile
	if profile != "" {
		p, ok := c.Profiles[profile]
		if !ok {
			return c, fmt.Errorf("unknown model profile %q", profile)
		}
		c.overlayProfile(p)
	}
	if err := c.applyEnvironment(); err != nil {
		return c, err
	}
	// Explicit selectors win over HARNESS_PROFILE/HARNESS_AGENT because their
	// selection occurred before applying provider overlays above.
	c.Profile = profile
	c.Agent = agent
	return c, nil
}
func (c *Config) applyEnvironment() error {
	endpoint, hasEndpoint := os.LookupEnv("HARNESS_ENDPOINT")
	baseURL, hasBaseURL := os.LookupEnv("HARNESS_BASE_URL")
	if hasEndpoint || hasBaseURL {
		c.Endpoint, c.BaseURL = endpoint, baseURL
	}
	for key, p := range map[string]*string{"HARNESS_PROTOCOL": &c.Protocol, "HARNESS_MODEL": &c.Model, "HARNESS_API_KEY_ENV": &c.APIKeyEnv, "HARNESS_SESSION_DIR": &c.SessionDir} {
		if v, ok := os.LookupEnv(key); ok {
			*p = v
		}
	}
	for key, p := range map[string]*int{"HARNESS_CONTEXT_WINDOW": &c.ContextWindow, "HARNESS_OUTPUT_BUDGET": &c.OutputBudget} {
		if v, ok := os.LookupEnv(key); ok {
			n, err := strconv.Atoi(v)
			if err != nil {
				return fmt.Errorf("%s: %w", key, err)
			}
			*p = n
		}
	}
	return nil
}
