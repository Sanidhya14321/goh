package core

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func str(v string) *string { return &v }
func integer(v int) *int   { return &v }
func TestProfileAgentSelectionAndEnvironmentPrecedence(t *testing.T) {
	cfg := Defaults()
	cfg.Endpoint = "http://localhost/base"
	cfg.Model = "base"
	cfg.ContextWindow = 32000
	cfg.Profiles = map[string]ModelProfile{"fast": {Endpoint: str("http://localhost/fast"), Model: str("fast-model"), ContextWindow: integer(64000), OutputBudget: integer(1000), APIKeyEnv: str("FAST_CREDENTIAL")}, "deep": {Model: str("deep-model")}}
	cfg.Agents = map[string]AgentProfile{"reviewer": {Description: "Review code", Instructions: "Explain file findings", Mode: "plan", Profile: "fast"}}
	selected, err := cfg.Resolve("", "reviewer")
	if err != nil {
		t.Fatal(err)
	}
	if selected.AgentMode != "plan" || selected.Profile != "fast" || selected.Model != "fast-model" || selected.AgentInstructions != "Explain file findings" || selected.APIKeyEnv != "FAST_CREDENTIAL" {
		t.Fatalf("selection %+v", selected)
	}
	selected, err = cfg.Resolve("deep", "reviewer")
	if err != nil {
		t.Fatal(err)
	}
	if selected.Model != "deep-model" || selected.Profile != "deep" {
		t.Fatal("explicit profile did not override agent preference")
	}
	t.Setenv("HARNESS_MODEL", "environment")
	t.Setenv("HARNESS_CONTEXT_WINDOW", "128000")
	t.Setenv("HARNESS_OUTPUT_BUDGET", "2000")
	selected, err = cfg.Resolve("fast", "reviewer")
	if err != nil {
		t.Fatal(err)
	}
	if selected.Model != "environment" || selected.ContextWindow != 128000 || selected.OutputBudget != 2000 {
		t.Fatal("profile overwrote environment")
	}
	selected.Model = "cli-model"
	if err := selected.Validate(); err != nil {
		t.Fatal(err)
	}
	// The resolved instructions/mode are ephemeral configuration, not stored provider secrets.
	data, err := json.Marshal(selected)
	if err != nil {
		t.Fatal(err)
	}
	var wire map[string]any
	json.Unmarshal(data, &wire)
	if _, ok := wire["AgentMode"]; ok {
		t.Fatal("runtime policy serialized")
	}
}
func TestProfileExplicitInvalidOverridesRemainInvalid(t *testing.T) {
	cfg := Defaults()
	cfg.Endpoint = "http://localhost/base"
	cfg.Model = "base"
	cfg.ContextWindow = 32000
	cfg.Profiles = map[string]ModelProfile{"zero": {ContextWindow: integer(0)}, "empty": {Model: str("")}}
	for _, name := range []string{"zero", "empty"} {
		selected, err := cfg.Resolve(name, "build")
		if err != nil {
			t.Fatal(err)
		}
		if selected.Validate() == nil {
			t.Fatalf("profile %s silently ignored invalid override", name)
		}
	}
}
func TestProfileRejectsUnknownAndAuthorityFields(t *testing.T) {
	cfg := Defaults()
	for _, selectors := range [][2]string{{"missing", "build"}, {"", "missing"}} {
		if _, err := cfg.Resolve(selectors[0], selectors[1]); err == nil {
			t.Fatal("unknown selection accepted")
		}
	}
	for _, definition := range []AgentProfile{{Mode: "subagent"}, {Mode: "build", Profile: "missing"}} {
		cfg.Agents = map[string]AgentProfile{"reviewer": definition}
		if _, err := cfg.Resolve("", "reviewer"); err == nil {
			t.Fatal("invalid custom agent accepted")
		}
	}
	cfg.Agents = map[string]AgentProfile{"plan": {Mode: "build"}}
	if _, err := cfg.Resolve("", "plan"); err == nil {
		t.Fatal("built-in plan policy overridden")
	}
	p := filepath.Join(t.TempDir(), "config.json")
	for _, input := range []string{`{"agents":{"reviewer":{"mode":"plan","instructions":"review","allow":["shell"]}}}`, `{"profiles":{"fast":{"api_key":"secret"}}}`, `{} {}`} {
		if err := os.WriteFile(p, []byte(input), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := LoadConfig(p); err == nil {
			t.Fatalf("unsupported config admitted: %s", input)
		}
	}
}
func TestConfigProfileSelectorsAndSkillPaths(t *testing.T) {
	p := filepath.Join(t.TempDir(), "config.json")
	input := `{"profile":"slow","agent":"build","profiles":{"fast":{"endpoint":"http://localhost/profile","model":"fixture","context_window":32000}},"skill_paths":["/explicit/skills"]}`
	if err := os.WriteFile(p, []byte(input), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HARNESS_PROFILE", "fast")
	t.Setenv("HARNESS_AGENT", "plan")
	cfg, err := LoadConfig(p)
	if err != nil {
		t.Fatal(err)
	}
	cfg, err = cfg.Resolve("", "")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Profile != "fast" || cfg.Agent != "plan" || cfg.AgentMode != "plan" || len(cfg.SkillPaths) != 1 {
		t.Fatalf("selectors %+v", cfg)
	}
	cfg, err = cfg.Resolve("fast", "build")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Agent != "build" {
		t.Fatal("explicit agent overridden by environment")
	}
}
