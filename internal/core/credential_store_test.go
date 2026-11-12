package core

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Sanidhya14321/goh/internal/secretstore"
)

type fakeStore map[string][]byte

func (f fakeStore) Name() string { return "Fake Store" }
func (f fakeStore) Get(name string) ([]byte, error) {
	if b, ok := f[name]; ok {
		return b, nil
	}
	return nil, secretstore.ErrNotFound
}
func (f fakeStore) Set(name string, data []byte) error {
	f[name] = append([]byte(nil), data...)
	return nil
}
func (f fakeStore) Delete(name string) error {
	if _, ok := f[name]; !ok {
		return secretstore.ErrNotFound
	}
	delete(f, name)
	return nil
}

// isolateConfig gives the test its own user config folder on every OS
// (Windows reads %AppData%, not HOME).
func isolateConfig(t *testing.T) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("APPDATA", home)
}

func useFakeStore(t *testing.T) fakeStore {
	t.Helper()
	isolateConfig(t)
	store := fakeStore{}
	old := openSecretStore
	openSecretStore = func(string) (secretstore.Store, error) { return store, nil }
	t.Cleanup(func() { openSecretStore = old })
	return store
}

func testAccount() chatGPTAccount {
	return chatGPTAccount{ClientID: "client", Subject: "subject", Model: "m",
		Tokens: chatGPTTokens{Access: "a", Refresh: "r", Type: "Bearer", Scope: chatGPTScope, Lifetime: 60}}
}

// Moving between stores carries every saved secret, leaves no plaintext
// copy behind, and reads keep working throughout.
func TestCredentialStoreMovesSecrets(t *testing.T) {
	store := useFakeStore(t)
	t.Setenv("GOH_TEST_KEY", "")
	cfg := Config{BaseURL: "https://gateway.example/v1", APIKeyEnv: "GOH_TEST_KEY"}
	if err := cfg.SaveProviderKey("key-one"); err != nil {
		t.Fatal(err)
	}
	if err := saveChatGPTAccount(testAccount()); err != nil {
		t.Fatal(err)
	}
	keyPath, _ := cfg.credentialPath()
	accountPath, _ := chatGPTPath(chatGPTAccountFile)

	moved, err := SetCredentialStore(StoreKeychain)
	if err != nil || moved != 2 {
		t.Fatal(moved, err)
	}
	for _, p := range []string{keyPath, accountPath} {
		if _, err := os.Stat(p); !os.IsNotExist(err) {
			t.Fatal("plaintext copy left:", filepath.Base(p))
		}
		if !inKeychain(p) {
			t.Fatal("not sealed:", filepath.Base(p))
		}
		if b, _ := os.ReadFile(p + sealedSuffix); strings.Contains(string(b), "key-one") || strings.Contains(string(b), "subject") {
			t.Fatal("sealed file holds plaintext")
		}
	}
	if len(store[dataKeyName]) != 32 || len(store) != 1 {
		t.Fatal("OS store should hold only the data key")
	}
	// Sealed files are bound to their names.
	sealedKey, _ := os.ReadFile(keyPath + sealedSuffix)
	sealedAccount, _ := os.ReadFile(accountPath + sealedSuffix)
	if err := writePrivate(keyPath+sealedSuffix, string(sealedAccount)); err != nil {
		t.Fatal(err)
	}
	if _, err := cfg.ProviderKey(); err == nil {
		t.Fatal("swapped sealed file accepted")
	}
	if err := writePrivate(keyPath+sealedSuffix, string(sealedKey)); err != nil {
		t.Fatal(err)
	}
	if key, err := cfg.ProviderKey(); err != nil || key != "key-one" {
		t.Fatal(key, err)
	}
	if a, err := readChatGPTAccount(); err != nil || a.Subject != "subject" {
		t.Fatal(a, err)
	}
	if got := cfg.CredentialSource(); got != "saved key (Fake Store)" {
		t.Fatal(got)
	}
	if ChatGPTKeptIn() != "Fake Store" {
		t.Fatal("status does not name the store")
	}

	// New saves follow the choice.
	if err := cfg.SaveProviderKey("key-two"); err != nil {
		t.Fatal(err)
	}
	if key, err := keychainSecret(keyPath); err != nil || string(key) != "key-two" {
		t.Fatal("new key not sealed", err)
	}
	if _, err := os.Stat(keyPath); !os.IsNotExist(err) {
		t.Fatal("new key also written to a file")
	}

	if moved, err := SetCredentialStore(StoreFile); err != nil || moved != 2 {
		t.Fatal(moved, err)
	}
	if len(store) != 0 {
		t.Fatal("data key left in the OS store")
	}
	if inKeychain(keyPath) || inKeychain(accountPath) {
		t.Fatal("markers left")
	}
	if key, err := cfg.ProviderKey(); err != nil || key != "key-two" {
		t.Fatal(key, err)
	}

	// Removing a key removes it from whichever store holds it.
	if _, err := SetCredentialStore(StoreKeychain); err != nil {
		t.Fatal(err)
	}
	if err := cfg.SaveProviderKey(""); err != nil {
		t.Fatal(err)
	}
	if key, _ := cfg.ProviderKey(); key != "" || inKeychain(keyPath) {
		t.Fatal("key not removed")
	}
}

// A sealed secret whose key has gone from the OS store, or that does not
// open with it, is a damaged credential: sign-in replaces it and sign-out
// clears it, as with a damaged file.
func TestUnopenableSealedSecretIsDamaged(t *testing.T) {
	for name, damage := range map[string]func(fakeStore, string){
		"key gone": func(s fakeStore, _ string) { delete(s, dataKeyName) },
		"garbage":  func(_ fakeStore, p string) { _ = writePrivate(p+sealedSuffix, "not sealed") },
	} {
		t.Run(name, func(t *testing.T) {
			store := useFakeStore(t)
			if _, err := SetCredentialStore(StoreKeychain); err != nil {
				t.Fatal(err)
			}
			if err := saveChatGPTAccount(testAccount()); err != nil {
				t.Fatal(err)
			}
			path, _ := chatGPTPath(chatGPTAccountFile)
			damage(store, path)
			if _, err := readChatGPTAccount(); !errors.Is(err, ErrChatGPTDamaged) {
				t.Fatal(err)
			}
			if err := LogoutChatGPT(context.Background()); err != nil {
				t.Fatal(err)
			}
			if inKeychain(path) {
				t.Fatal("sealed file not cleared")
			}
			if _, err := readChatGPTAccount(); !errors.Is(err, ErrChatGPTLogin) {
				t.Fatal(err)
			}
		})
	}
}

// When no OS store exists the choice is refused and nothing moves.
func TestKeychainUnavailable(t *testing.T) {
	isolateConfig(t)
	old := openSecretStore
	openSecretStore = func(string) (secretstore.Store, error) { return nil, secretstore.ErrUnavailable }
	t.Cleanup(func() { openSecretStore = old })
	if err := saveChatGPTAccount(testAccount()); err != nil {
		t.Fatal(err)
	}
	if _, err := SetCredentialStore(StoreKeychain); !errors.Is(err, secretstore.ErrUnavailable) {
		t.Fatal(err)
	}
	if s, err := CredentialStore(); err != nil || s != StoreFile {
		t.Fatal(s, err)
	}
	if _, err := readChatGPTAccount(); err != nil {
		t.Fatal(err)
	}
	if _, err := SetCredentialStore("vault"); err == nil {
		t.Fatal("unknown store accepted")
	}
}

// A move interrupted after the OS copy was written but before the file was
// removed reads the OS copy, and finishing the move cleans up the file.
func TestInterruptedMovePrefersTheOSCopy(t *testing.T) {
	store := useFakeStore(t)
	if err := saveChatGPTAccount(testAccount()); err != nil {
		t.Fatal(err)
	}
	path, _ := chatGPTPath(chatGPTAccountFile)
	newer := testAccount()
	newer.Subject = "newer"
	if _, err := SetCredentialStore(StoreKeychain); err != nil {
		t.Fatal(err)
	}
	// Simulate the half-finished state: the file reappears beside the marker.
	if err := saveChatGPTAccountFile(path, testAccount()); err != nil {
		t.Fatal(err)
	}
	if err := saveChatGPTAccount(newer); err != nil {
		t.Fatal(err)
	}
	if err := saveChatGPTAccountFile(path, testAccount()); err != nil {
		t.Fatal(err)
	}
	if a, err := readChatGPTAccount(); err != nil || a.Subject != "newer" {
		t.Fatal(a.Subject, err)
	}
	if moved, err := SetCredentialStore(StoreFile); err != nil || moved != 1 || len(store) != 0 {
		t.Fatal(moved, err, len(store))
	}
	if a, err := readChatGPTAccount(); err != nil || a.Subject != "newer" {
		t.Fatal(a.Subject, err)
	}
}

func saveChatGPTAccountFile(path string, a chatGPTAccount) error {
	b, err := json.Marshal(a)
	if err != nil {
		return err
	}
	return writePrivate(path, string(b))
}

func TestForgetCredentialKey(t *testing.T) {
	store := useFakeStore(t)
	if UsesCredentialKey() {
		t.Fatal("file store reported as keychain")
	}
	if _, err := SetCredentialStore(StoreKeychain); err != nil {
		t.Fatal(err)
	}
	if !UsesCredentialKey() {
		t.Fatal("keychain store not reported")
	}
	if removed, err := ForgetCredentialKey(); !removed || err != nil || len(store) != 0 {
		t.Fatal(removed, err, len(store))
	}
	if removed, err := ForgetCredentialKey(); removed || err != nil {
		t.Fatal(removed, err)
	}
}
