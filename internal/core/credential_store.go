package core

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Sanidhya14321/goh/internal/secretstore"
)

// Saved secrets (the ChatGPT account and provider keys) live either in a
// private plaintext file under the credential folder (the default) or
// sealed by a key that only the OS credential store holds, chosen with
// `goh auth store`. A sealed secret is NAME.sealed beside where its plain
// file would be: AES-256-GCM under the store's data key, with NAME as
// associated data so sealed files cannot be swapped for one another. The OS
// store holds only that 32-byte key, so no secret meets a store's size
// limits, and copying the credential folder without the key reveals nothing.
// Each secret is read from wherever it actually is, so an interrupted move
// between stores loses nothing; a sealed copy wins over a plain one.

const (
	StoreFile     = "file"
	StoreKeychain = "keychain"
	sealedSuffix  = ".sealed"
	storeChoice   = "store"
	dataKeyName   = "credential-key"
)

// openSecretStore is replaced in tests.
var openSecretStore = secretstore.Open

func credentialDir() (string, error) {
	cfg, err := UserConfigPath()
	if err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(cfg), "credentials"), nil
}

// CredentialStore reports where new secrets are saved: StoreFile or
// StoreKeychain.
func CredentialStore() (string, error) {
	dir, err := credentialDir()
	if err != nil {
		return "", err
	}
	b, err := readConfig(filepath.Join(dir, storeChoice))
	if os.IsNotExist(err) {
		return StoreFile, nil
	}
	if err != nil {
		return "", err
	}
	switch s := strings.TrimSpace(string(b)); s {
	case StoreFile, StoreKeychain:
		return s, nil
	}
	return "", fmt.Errorf("unrecognised credential store setting in %s; run `goh auth store file` or `goh auth store keychain`", filepath.Join(dir, storeChoice))
}

// CredentialStoreName describes the OS store on this system.
func CredentialStoreName() string {
	dir, _ := credentialDir()
	s, err := openSecretStore(dir)
	if err != nil {
		return "OS credential store (unavailable on this system)"
	}
	return s.Name()
}

func inKeychain(path string) bool {
	info, err := os.Lstat(path + sealedSuffix)
	return err == nil && info.Mode().IsRegular()
}

func osStore(dir string) (secretstore.Store, error) {
	s, err := openSecretStore(dir)
	if err != nil {
		return nil, fmt.Errorf("%w; sign in again, or run `goh auth store file` on the system that saved it", err)
	}
	return s, nil
}

// dataKey reads the sealing key from the OS store. Only SetCredentialStore
// creates it, under the sign-in lock, so concurrent saves never race to
// create different keys.
func dataKey(dir string) ([]byte, secretstore.Store, error) {
	s, err := osStore(dir)
	if err != nil {
		return nil, nil, err
	}
	key, err := s.Get(dataKeyName)
	if errors.Is(err, secretstore.ErrNotFound) {
		return nil, s, fmt.Errorf("the %s no longer holds GoHarness's credential key; run `goh auth store keychain` and sign in or save the key again", s.Name())
	}
	if err == nil && len(key) != 32 {
		err = fmt.Errorf("the %s holds a malformed GoHarness credential key", s.Name())
	}
	return key, s, err
}

func sealer(key []byte) (cipher.AEAD, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

// keychainSecret opens a sealed secret.
func keychainSecret(path string) ([]byte, error) {
	sealed, err := readConfig(path + sealedSuffix)
	if err != nil {
		return nil, err
	}
	key, _, err := dataKey(filepath.Dir(path))
	if err != nil {
		return nil, err
	}
	aead, err := sealer(key)
	if err != nil {
		return nil, err
	}
	n := aead.NonceSize()
	if len(sealed) < n {
		return nil, fmt.Errorf("%s is damaged; sign in or save the key again", filepath.Base(path)+sealedSuffix)
	}
	plain, err := aead.Open(nil, sealed[:n], sealed[n:], []byte(filepath.Base(path)))
	if err != nil {
		return nil, fmt.Errorf("%s does not match the key in the OS credential store; sign in or save the key again", filepath.Base(path)+sealedSuffix)
	}
	return plain, nil
}

// saveSecret writes a secret to the chosen store and then removes any copy
// in the other one.
func saveSecret(path string, data []byte) error {
	choice, err := CredentialStore()
	if err != nil {
		return err
	}
	if choice == StoreKeychain {
		key, _, err := dataKey(filepath.Dir(path))
		if err != nil {
			return err
		}
		aead, err := sealer(key)
		if err != nil {
			return err
		}
		nonce := make([]byte, aead.NonceSize())
		if _, err := rand.Read(nonce); err != nil {
			return err
		}
		sealed := aead.Seal(nonce, nonce, data, []byte(filepath.Base(path)))
		if err := writePrivate(path+sealedSuffix, string(sealed)); err != nil {
			return err
		}
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return err
		}
		return nil
	}
	if err := writePrivate(path, string(data)); err != nil {
		return err
	}
	if err := os.Remove(path + sealedSuffix); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// removeSecret deletes a secret in both forms.
func removeSecret(path string) error {
	for _, p := range []string{path, path + sealedSuffix} {
		if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

// isSecretName reports the credential folder's entries that are secrets.
func isSecretName(name string) bool {
	return name == chatGPTAccountFile || strings.HasSuffix(name, ".key") && len(name) == 64+len(".key")
}

// savedSecrets lists the secrets in dir, in either form.
func savedSecrets(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	for _, e := range entries {
		name := strings.TrimSuffix(e.Name(), sealedSuffix)
		if e.Type().IsRegular() && isSecretName(name) {
			seen[name] = true
		}
	}
	names := make([]string, 0, len(seen))
	for n := range seen {
		names = append(names, n)
	}
	sort.Strings(names)
	return names, nil
}

// SetCredentialStore chooses where secrets are saved and moves the ones
// already saved. It returns how many were moved. Choosing the keychain
// creates the data key in the OS store if it is not there yet; going back to
// files deletes it once nothing is sealed with it.
func SetCredentialStore(choice string) (int, error) {
	if choice != StoreFile && choice != StoreKeychain {
		return 0, fmt.Errorf("credential store must be %q or %q", StoreFile, StoreKeychain)
	}
	dir, err := credentialDir()
	if err != nil {
		return 0, err
	}
	unlock, err := chatGPTLock()
	if err != nil {
		return 0, err
	}
	defer unlock()
	if choice == StoreKeychain {
		s, err := openSecretStore(dir)
		if err != nil {
			return 0, err
		}
		if _, err := s.Get(dataKeyName); errors.Is(err, secretstore.ErrNotFound) {
			key := make([]byte, 32)
			if _, err := rand.Read(key); err != nil {
				return 0, err
			}
			if err := s.Set(dataKeyName, key); err != nil {
				return 0, err
			}
		} else if err != nil {
			return 0, err
		}
	}
	if err := writePrivate(filepath.Join(dir, storeChoice), choice+"\n"); err != nil {
		return 0, err
	}
	names, err := savedSecrets(dir)
	if err != nil {
		return 0, err
	}
	moved := 0
	for _, name := range names {
		path := filepath.Join(dir, name)
		_, plainErr := os.Lstat(path)
		sealed, plain := inKeychain(path), plainErr == nil
		if choice == StoreKeychain && sealed && !plain || choice == StoreFile && !sealed {
			continue
		}
		var data []byte
		if sealed {
			data, err = keychainSecret(path)
		} else {
			data, err = readConfig(path)
		}
		if err == nil {
			err = saveSecret(path, data)
		}
		if err != nil {
			return moved, fmt.Errorf("moving %s: %w", name, err)
		}
		moved++
	}
	if choice == StoreFile {
		if s, err := openSecretStore(dir); err == nil {
			if err := s.Delete(dataKeyName); err != nil && !errors.Is(err, secretstore.ErrNotFound) {
				return moved, err
			}
		}
	}
	return moved, nil
}

// ChatGPTKeptIn names the OS store whose key seals the ChatGPT account, or
// "" when it is a plain private file (or absent).
func ChatGPTKeptIn() string {
	if path, err := chatGPTPath(chatGPTAccountFile); err == nil && inKeychain(path) {
		return CredentialStoreName()
	}
	return ""
}

// ForgetCredentialKey deletes the data key from the OS store, for uninstall.
// It reports whether a key was there to delete.
func ForgetCredentialKey() (bool, error) {
	dir, err := credentialDir()
	if err != nil {
		return false, err
	}
	s, err := openSecretStore(dir)
	if err != nil {
		return false, nil
	}
	err = s.Delete(dataKeyName)
	if errors.Is(err, secretstore.ErrNotFound) {
		return false, nil
	}
	return err == nil, err
}

// UsesCredentialKey reports whether the keychain store is chosen, so a key
// may exist in the OS store.
func UsesCredentialKey() bool {
	s, err := CredentialStore()
	return err == nil && s == StoreKeychain
}
