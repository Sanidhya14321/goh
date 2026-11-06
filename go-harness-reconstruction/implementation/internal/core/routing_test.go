package core

import "testing"

func TestRoleProfilesAndProtocolCredentialDefaults(t *testing.T) {
	c := Defaults()
	c.Endpoint = "http://localhost/chat"
	c.Model = "parent"
	c.ContextWindow = 131072
	c.APIKeyEnv = "PARENT_SECRET"
	c.Profiles = map[string]ModelProfile{"native": {Protocol: str("anthropic_messages"), Endpoint: str("http://localhost/messages"), Model: str("child"), ContextWindow: integer(32768)}}
	c.ModelRoles = map[string]string{"task": "native"}
	selected, err := c.WithModelRole("task")
	if err != nil {
		t.Fatal(err)
	}
	if selected.Model != "child" || selected.CredentialEnvironment() != "ANTHROPIC_API_KEY" || selected.Profile != "native" || c.CredentialEnvironment() != "PARENT_SECRET" || selected.ContextWindow != 32768 {
		t.Fatal(selected)
	}
	c.Profiles["native"] = ModelProfile{Protocol: str("anthropic_messages"), APIKeyEnv: str("CUSTOM_NATIVE_SECRET")}
	selected, err = c.WithModelRole("task")
	if err != nil || selected.CredentialEnvironment() != "CUSTOM_NATIVE_SECRET" {
		t.Fatal(selected, err)
	}
	inherited, err := c.WithModelRole("compaction")
	if err != nil || inherited.Model != c.Model {
		t.Fatal(inherited, err)
	}
	if Defaults().CredentialEnvironment() != "OPENAI_API_KEY" {
		t.Fatal("chat default changed")
	}
	for _, roles := range []map[string]string{{"default": "native"}, {"task": "unknown"}, {"explore": ""}} {
		c.ModelRoles = roles
		if c.ValidateModelRoles() == nil {
			t.Fatal("accepted role", roles)
		}
	}
	c.Protocol = "unsupported"
	if c.ValidateProtocol() == nil {
		t.Fatal("accepted protocol")
	}
}
func TestRoleEnvironmentPrecedence(t *testing.T) {
	c := Defaults()
	c.Model = "parent"
	c.Profiles = map[string]ModelProfile{"fast": {Model: str("fast")}}
	c.ModelRoles = map[string]string{"compaction": "fast"}
	t.Setenv("HARNESS_MODEL", "global-model")
	t.Setenv("HARNESS_PROTOCOL", "anthropic_messages")
	selected, err := c.WithModelRole("compaction")
	if err != nil || selected.Model != "global-model" || selected.EffectiveProtocol() != "anthropic_messages" {
		t.Fatal(selected, err)
	}
}
