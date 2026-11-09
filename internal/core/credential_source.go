package core

import "os"

// CredentialSource names where this provider's credential would come from,
// without returning or printing the credential: "ChatGPT sign-in",
// "environment NAME", "saved key", "ChatGPT not signed in" or "none".
func (c Config) CredentialSource() string {
	if c.EffectiveProtocol() == ChatGPTProtocol {
		if _, err := readChatGPTAccount(); err != nil {
			return "ChatGPT not signed in"
		}
		return "ChatGPT sign-in"
	}
	env := c.CredentialEnvironment()
	key, err := c.ProviderKey()
	if err != nil {
		return "unavailable: " + err.Error()
	}
	if key == "" {
		return "none"
	}
	if env != "" && key == os.Getenv(env) {
		return "environment " + env
	}
	return "saved key"
}
