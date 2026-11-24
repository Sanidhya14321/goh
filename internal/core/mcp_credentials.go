package core

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/Sanidhya14321/goh/internal/platform"
)

// MCP sign-ins (OAuth tokens for remote MCP servers) are secrets like the
// ChatGPT sign-in: kept in the credential folder, sealed with the OS
// credential store's key when that store is chosen, and moved with the rest
// by `goh auth store`. Each is keyed by the server's URL, so a token is only
// ever found for the exact server it was issued for.

const mcpCredentialPrefix = "mcp-oauth-"

func mcpCredentialPath(serverURL string) (string, error) {
	sum := sha256.Sum256([]byte(serverURL))
	return chatGPTPath(mcpCredentialPrefix + hex.EncodeToString(sum[:12]) + ".json")
}

func isMCPCredentialName(name string) bool {
	return strings.HasPrefix(name, mcpCredentialPrefix) && strings.HasSuffix(name, ".json") && len(name) == len(mcpCredentialPrefix)+24+len(".json")
}

// ReadMCPCredential returns the saved sign-in for serverURL, or an error
// wrapping os.ErrNotExist when there is none.
func ReadMCPCredential(serverURL string) ([]byte, error) {
	path, err := mcpCredentialPath(serverURL)
	if err != nil {
		return nil, err
	}
	if inKeychain(path) {
		return keychainSecret(path)
	}
	f, err := os.OpenFile(path, os.O_RDONLY|platform.NoFollow|platform.NonBlock, 0)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() || !platform.PrivateFile(info, 0600) || info.Size() > 1<<20 {
		return nil, fmt.Errorf("%s must be a private regular file (mode 0600); sign in again with `goh mcp login`", path)
	}
	return io.ReadAll(io.LimitReader(f, 1<<20))
}

// SaveMCPCredential saves the sign-in for serverURL in the chosen store.
func SaveMCPCredential(serverURL string, data []byte) error {
	path, err := mcpCredentialPath(serverURL)
	if err != nil {
		return err
	}
	return saveSecret(path, data)
}

// RemoveMCPCredential forgets the sign-in for serverURL; it reports whether
// one was saved.
func RemoveMCPCredential(serverURL string) (bool, error) {
	path, err := mcpCredentialPath(serverURL)
	if err != nil {
		return false, err
	}
	_, plainErr := os.Lstat(path)
	saved := plainErr == nil || inKeychain(path)
	if err := removeSecret(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return saved, err
	}
	return saved, nil
}
