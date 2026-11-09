package core

import (
	"encoding/json"
	"fmt"
	"github.com/Sanidhya14321/goh/internal/platform"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

type MCPConfig struct {
	Command string            `json:"command,omitempty"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
	// URL selects a remote MCP server (Streamable HTTP) instead of Command.
	URL string `json:"url,omitempty"`
	// HeadersEnv maps an HTTP header to the environment variable holding its
	// value (for example {"Authorization": "MCP_TOKEN"}), so tokens never sit
	// in the config file.
	HeadersEnv map[string]string `json:"headers_env,omitempty"`
	// Headers are literal values supplied by an ACP client for this
	// connection only; they are never read from or written to config files.
	Headers map[string]string `json:"-"`
}
type CommandConfig struct {
	Description string `json:"description"`
	Template    string `json:"template"`
	// Agent (user configuration only) hands the expanded prompt to this
	// subagent through the task tool, like an @mention. Command files in a
	// repository cannot set it.
	Agent string `json:"agent,omitempty"`
}

type LSPConfig struct {
	LanguageID     string            `json:"language_id,omitempty"`
	Command        string            `json:"command"`
	Args           []string          `json:"args,omitempty"`
	Extensions     []string          `json:"extensions"`
	Env            map[string]string `json:"env,omitempty"`
	Initialization map[string]any    `json:"initialization,omitempty"`
}

// ModelPrice is a user-supplied price per million tokens.
type ModelPrice struct {
	Input  float64 `json:"input"`
	Output float64 `json:"output"`
}

type FormatterConfig struct {
	Command    string            `json:"command,omitempty"`
	Args       []string          `json:"args,omitempty"`
	Extensions []string          `json:"extensions,omitempty"`
	Env        map[string]string `json:"env,omitempty"`
	Disabled   bool              `json:"disabled,omitempty"`
	Timeout    int               `json:"timeout_seconds,omitempty"`
}

type Config struct {
	Memory              bool              `json:"memory,omitempty"`
	AllowGeneratedEdits bool              `json:"allow_generated_edits,omitempty"`
	Protocol            string            `json:"protocol,omitempty"`
	ModelRoles          map[string]string `json:"model_roles,omitempty"`
	FallbackProfiles    []string          `json:"fallback_profiles,omitempty"`
	// ShellRules auto-allow or deny shell commands by pattern; see shellrules.go.
	ShellRules []ShellRule `json:"shell_rules,omitempty"`
	// FileRules allow, deny or always ask edits by project path pattern.
	FileRules []ShellRule `json:"file_rules,omitempty"`
	// ReasoningEffort asks OpenAI reasoning models for minimal, low, medium or
	// high effort; empty leaves the provider default.
	ReasoningEffort string `json:"reasoning_effort,omitempty"`
	// CompactionThreshold is the context fill (percent, 50-95) at which
	// automatic compaction starts; 0 means the default of 75. Compaction
	// always runs when the next request would not otherwise fit.
	CompactionThreshold int `json:"compaction_threshold,omitempty"`
	// ThinkingBudget enables Anthropic extended thinking with this many
	// tokens (at least 1024 and below output_budget); 0 disables it.
	ThinkingBudget int `json:"thinking_budget,omitempty"`
	// WebsearchURL is a SearXNG-compatible base URL enabling the websearch tool.
	WebsearchURL string `json:"websearch_url,omitempty"`
	// CompactionMasking (default true) masks old tool output before summarizing.
	CompactionMasking *bool `json:"compaction_masking,omitempty"`
	// Keybinds remaps chat actions; see DefaultKeybinds.
	Keybinds map[string]string `json:"keybinds,omitempty"`
	// AutoTitle asks the model (the "title" role, else the main model) for a
	// short session title after the first completed run. Off by default
	// because it is an extra request.
	AutoTitle bool `json:"auto_title,omitempty"`
	// Theme selects a built-in color theme (dark, light, high-contrast).
	Theme      string                     `json:"theme,omitempty"`
	Formatters map[string]FormatterConfig `json:"formatters,omitempty"`
	Snapshots  bool                       `json:"snapshots,omitempty"`
	// SnapshotsStrict is set when checkpoints were requested explicitly: a
	// failed capture then blocks the edit. Default checkpoints are best effort.
	SnapshotsStrict bool                 `json:"-"`
	LSP             map[string]LSPConfig `json:"lsp,omitempty"`
	// Prices maps a model name to its price in USD per million tokens, used
	// only for cost estimates in goh stats. GoHarness ships no price table.
	Prices map[string]ModelPrice `json:"prices,omitempty"`
	// LSPAuto adds installed well-known language servers (see knownLSP).
	LSPAuto bool `json:"lsp_auto,omitempty"`
	// DisablePromptCache stops GoHarness marking Anthropic prompt-cache
	// breakpoints (for gateways that reject cache_control).
	DisablePromptCache bool `json:"disable_prompt_cache,omitempty"`
	// EnvPassthrough names extra environment variables (NAME, or PREFIX_*)
	// that shell commands, jobs, formatters, MCP and LSP servers receive in
	// addition to a fixed safe set; secrets are withheld unless listed. A lone
	// "*" passes the whole environment.
	EnvPassthrough []string `json:"env_passthrough,omitempty"`
	// FormatterAuto adds installed well-known formatters (see knownFormatters).
	FormatterAuto   bool                     `json:"formatter_auto,omitempty"`
	Commands        map[string]CommandConfig `json:"commands,omitempty"`
	Endpoint        string                   `json:"endpoint"`
	BaseURL         string                   `json:"base_url,omitempty"`
	Model           string                   `json:"model"`
	APIKeyEnv       string                   `json:"api_key_env"`
	ContextWindow   int                      `json:"context_window"`
	OutputBudget    int                      `json:"output_budget"`
	OutputParameter string                   `json:"output_parameter"`
	MaxTurns        int                      `json:"max_turns"`
	SessionDir      string                   `json:"session_dir"`
	MCP             map[string]MCPConfig     `json:"mcp,omitempty"`
	Profile         string                   `json:"profile,omitempty"`
	Agent           string                   `json:"agent,omitempty"`
	Profiles        map[string]ModelProfile  `json:"profiles,omitempty"`
	Agents          map[string]AgentProfile  `json:"agents,omitempty"`
	// SkippedAgentFiles explains project agent files that were not loaded.
	SkippedAgentFiles []SkippedAgentFile `json:"-"`
	SkillPaths        []string           `json:"skill_paths,omitempty"`
	AgentMode         string             `json:"-"`
	AgentInstructions string             `json:"-"`
	AgentDescription  string             `json:"-"`
}

func Defaults() Config {
	home, _ := os.UserHomeDir()
	return Config{OutputBudget: 4096, OutputParameter: "max_tokens", MaxTurns: 64, SessionDir: filepath.Join(home, ".local", "state", "independent-harness", "sessions")}
}
func UserConfigPath() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "independent-harness", "config.json"), nil
}

func LoadConfig(path string) (Config, error) {
	c := Defaults()
	if path == "" {
		candidate, e := UserConfigPath()
		if e == nil {
			if _, e = os.Stat(candidate); e == nil {
				path = candidate
			}
		}
	}
	if path != "" {
		b, e := readConfig(path)
		if e != nil {
			return c, e
		}
		c, e = DecodeConfig(string(b))
		if e != nil {
			return c, e
		}
	} else {
		c.Snapshots = true // best-effort checkpoints by default, as with a file
	}
	if v, ok := os.LookupEnv("HARNESS_PROFILE"); ok {
		c.Profile = v
	}
	if v, ok := os.LookupEnv("HARNESS_AGENT"); ok {
		c.Agent = v
	}
	if err := c.applyEnvironment(); err != nil {
		return c, err
	}
	for _, name := range c.EnvPassthrough {
		if name == "" || strings.ContainsAny(name, "=\x00") {
			return c, fmt.Errorf("env_passthrough: invalid variable name %q", name)
		}
	}
	platform.SetEnvPassthrough(c.EnvPassthrough)
	return c, nil
}

// DecodeConfig reads file settings without applying environment overrides.
func DecodeConfig(raw string) (Config, error) {
	c := Defaults()
	raw, err := StripJSONC(raw)
	if err != nil {
		return c, err
	}
	d := json.NewDecoder(strings.NewReader(raw))
	d.DisallowUnknownFields()
	if err := d.Decode(&c); err != nil {
		return c, err
	}
	if err := d.Decode(new(any)); err != io.EOF {
		return c, fmt.Errorf("configuration contains trailing JSON content")
	}
	// File checkpoints default to on (best effort), as OpenCode snapshots
	// do; an explicit "snapshots" setting is honored, and true is strict.
	var keys map[string]json.RawMessage
	if json.Unmarshal([]byte(raw), &keys) == nil {
		if _, set := keys["snapshots"]; set {
			c.SnapshotsStrict = c.Snapshots
		} else {
			c.Snapshots = true
		}
	}
	return c, nil
}

func ReadConfigText(path string) (string, error) {
	b, err := readConfig(path)
	return string(b), err
}
func (c Config) Validate() error {
	if err := c.ValidateModelRoles(); err != nil {
		return err
	}
	if err := c.ValidateFallbackProfiles(); err != nil {
		return err
	}
	if err := c.ValidateShellRules(); err != nil {
		return err
	}
	if err := (Config{ShellRules: c.FileRules}).ValidateShellRules(); err != nil {
		return fmt.Errorf("file_rules: %w", err)
	}
	if err := c.ValidateReasoningEffort(); err != nil {
		return err
	}
	if b := c.ThinkingBudget; b != 0 {
		if c.EffectiveProtocol() != "anthropic_messages" {
			return fmt.Errorf("thinking_budget applies only to the anthropic_messages protocol (use reasoning_effort for OpenAI)")
		}
		if b < 1024 || b >= c.OutputBudget {
			return fmt.Errorf("thinking_budget must be at least 1024 and below output_budget (%d)", c.OutputBudget)
		}
	}
	if c.WebsearchURL != "" {
		u, err := url.Parse(c.WebsearchURL)
		local := err == nil && (u.Hostname() == "localhost" || u.Hostname() == "127.0.0.1" || u.Hostname() == "::1")
		if err != nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || !(u.Scheme == "https" || u.Scheme == "http" && local) {
			return fmt.Errorf("websearch_url must be an https base URL (http only for localhost) without credentials, query or fragment")
		}
	}
	if t := c.CompactionThreshold; t != 0 && (t < 50 || t > 95) {
		return fmt.Errorf("compaction_threshold must be between 50 and 95 (percent)")
	}
	if _, err := ResolveKeymap(c.Keybinds); err != nil {
		return err
	}
	if err := c.ValidateTheme(); err != nil {
		return err
	}
	if err := c.ValidateProtocol(); err != nil {
		return err
	}
	if c.AgentMode != "" && c.AgentMode != "build" && c.AgentMode != "plan" {
		return fmt.Errorf("agent_mode must be build or plan")
	}
	if _, err := c.ProviderEndpoint(); err != nil {
		return err
	}
	if c.Model == "" || c.ContextWindow <= 0 {
		return fmt.Errorf("model and positive context_window are required")
	}
	if c.OutputBudget < 1 || c.ContextWindow < c.OutputBudget+1024 {
		return fmt.Errorf("context_window must reserve output_budget plus at least 1024 tokens")
	}
	if c.MaxTurns < 1 || c.MaxTurns > 256 {
		return fmt.Errorf("max_turns must be 1..256")
	}
	if c.OutputParameter != "max_tokens" && c.OutputParameter != "max_completion_tokens" && c.OutputParameter != "" {
		return fmt.Errorf("unsupported output_parameter")
	}
	return nil
}

// ProviderEndpoint resolves the request URL after configuration overlays.
func (c Config) ProviderEndpoint() (string, error) {
	if c.EffectiveProtocol() == ChatGPTProtocol {
		if (c.Endpoint != "" && c.Endpoint != ChatGPTEndpoint) ||
			(c.BaseURL != "" && strings.TrimRight(c.BaseURL, "/") != chatGPTResource) ||
			(c.Endpoint != "" && c.BaseURL != "") {
			return "", fmt.Errorf("ChatGPT authorization requires the official OpenAI Responses URL")
		}
		return ChatGPTEndpoint, nil
	}

	if c.Endpoint != "" && c.BaseURL != "" {
		return "", fmt.Errorf("specify either endpoint or base_url")
	}
	raw, field := c.Endpoint, "endpoint"
	if c.BaseURL != "" {
		raw, field = c.BaseURL, "base_url"
	}
	u, err := url.Parse(raw)
	if err != nil || u.Hostname() == "" || (u.Scheme != "http" && u.Scheme != "https") || u.User != nil || strings.Contains(raw, "#") {
		return "", fmt.Errorf("%s must be an http(s) URL without embedded credentials or a fragment", field)
	}
	if c.BaseURL == "" {
		return raw, nil
	}
	if u.RawQuery != "" || u.ForceQuery {
		return "", fmt.Errorf("base_url must not contain a query string; use endpoint for an exact URL")
	}
	route := "/chat/completions"
	if c.EffectiveProtocol() == "openai_responses" {
		route = "/responses"
	}
	if c.EffectiveProtocol() == "anthropic_messages" {
		route = "/messages"
	}
	return strings.TrimRight(raw, "/") + route, nil
}

func readConfig(path string) ([]byte, error) {
	f, e := os.OpenFile(path, os.O_RDONLY|platform.NonBlock, 0)
	if e != nil {
		return nil, e
	}
	defer f.Close()
	info, e := f.Stat()
	if e != nil {
		return nil, e
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("configuration must be a regular file")
	}
	if info.Size() > 1<<20 {
		return nil, fmt.Errorf("configuration exceeds1MiB")
	}
	b, e := io.ReadAll(io.LimitReader(f, (1<<20)+1))
	if e != nil {
		return nil, e
	}
	if len(b) > 1<<20 {
		return nil, fmt.Errorf("configuration exceeds1MiB")
	}
	return b, nil
}

// ValidateReasoningEffort accepts OpenAI effort levels. Anthropic extended
// thinking needs thinking blocks carried across turns, which GoHarness does not
// implement yet, so the setting is refused there rather than silently ignored.
func (c Config) ValidateReasoningEffort() error {
	switch c.ReasoningEffort {
	case "", "minimal", "low", "medium", "high":
	default:
		return fmt.Errorf("reasoning_effort must be minimal, low, medium or high")
	}
	if c.ReasoningEffort != "" && c.EffectiveProtocol() == "anthropic_messages" {
		return fmt.Errorf("reasoning_effort is for OpenAI protocols; use thinking_budget for Anthropic extended thinking")
	}
	return nil
}

// CompactAt returns the automatic compaction trigger as a percentage.
func (c Config) CompactAt() int {
	if c.CompactionThreshold == 0 {
		return 75
	}
	return c.CompactionThreshold
}
