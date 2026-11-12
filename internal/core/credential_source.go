package core

import "os"

// CredentialSource names where this provider's credential would come from,
// without returning or printing the credential: "ChatGPT sign-in",
// "environment NAME", "saved key", "ChatGPT not signed in" or "none". A
// secret held by the OS credential store names that store in parentheses.
func (c Config) CredentialSource() string {
	if c.EffectiveProtocol() == ChatGPTProtocol {
		if _, err := readChatGPTAccount(); err != nil {
			return "ChatGPT not signed in"
		}
		if path, err := chatGPTPath(chatGPTAccountFile); err == nil && inKeychain(path) {
			return "ChatGPT sign-in (" + CredentialStoreName() + ")"
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
	if path, err := c.credentialPath(); err == nil && inKeychain(path) {
		return "saved key (" + CredentialStoreName() + ")"
	}
	return "saved key"
}
