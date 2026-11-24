package core

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// MCP sign-ins round-trip in both stores, move with `goh auth store`, are
// keyed by the exact server URL, and are removed by logout.
func TestMCPCredentialsStoreAndMove(t *testing.T) {
	useFakeStore(t)
	const url = "https://mcp.example/mcp"
	if _, err := ReadMCPCredential(url); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("missing sign-in should be ErrNotExist:", err)
	}
	if err := SaveMCPCredential(url, []byte(`{"access_token":"secret-a"}`)); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadMCPCredential(url + "?x=1"); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("another URL found this sign-in")
	}
	path, _ := mcpCredentialPath(url)
	if !isSecretName(filepath.Base(path)) {
		t.Fatal("MCP sign-in is not treated as a secret:", path)
	}
	if moved, err := SetCredentialStore(StoreKeychain); err != nil || moved != 1 {
		t.Fatal(moved, err)
	}
	if b, _ := os.ReadFile(path + sealedSuffix); strings.Contains(string(b), "secret-a") {
		t.Fatal("sealed sign-in holds plaintext")
	}
	if b, err := ReadMCPCredential(url); err != nil || string(b) != `{"access_token":"secret-a"}` {
		t.Fatal("sealed read:", string(b), err)
	}
	if removed, err := RemoveMCPCredential(url); !removed || err != nil {
		t.Fatal("remove", removed, err)
	}
	if _, err := ReadMCPCredential(url); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("sign-in still readable after removal:", err)
	}
	if removed, _ := RemoveMCPCredential(url); removed {
		t.Fatal("removing twice reported a saved sign-in")
	}
}
