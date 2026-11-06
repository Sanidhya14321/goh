package core

import (
	"crypto/sha256"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

func WriteUserConfig(path, raw string) error {
	if len(raw) > 1<<20 {
		return fmt.Errorf("configuration exceeds 1 MiB")
	}
	if _, err := DecodeConfig(raw); err != nil {
		return err
	}
	return writePrivate(path, raw)
}

func writePrivate(path, data string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	f, err := os.CreateTemp(dir, ".goh-settings-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	_, writeErr := f.WriteString(data)
	closeErr := f.Close()
	if writeErr != nil {
		return writeErr
	}
	if closeErr != nil {
		return closeErr
	}
	return os.Rename(f.Name(), path)
}

func (c Config) credentialPath() (string, error) {
	endpoint, err := c.ProviderEndpoint()
	if err != nil {
		return "", err
	}
	u, err := url.Parse(endpoint)
	if err != nil {
		return "", err
	}
	if u.Scheme == "http" && u.Hostname() != "localhost" && !net.ParseIP(u.Hostname()).IsLoopback() {
		return "", fmt.Errorf("saved keys require HTTPS or a loopback URL")
	}
	path, err := UserConfigPath()
	if err != nil {
		return "", err
	}
	id := sha256.Sum256([]byte(c.EffectiveProtocol() + "\n" + endpoint))
	return filepath.Join(filepath.Dir(path), "credentials", fmt.Sprintf("%x.key", id)), nil
}

// ProviderKey prefers the explicit environment override. The saved credential
// is scoped to one request URL and protocol and never enters the environment.
func (c Config) ProviderKey() (string, error) {
	if key := os.Getenv(c.CredentialEnvironment()); key != "" {
		return key, nil
	}
	path, err := c.credentialPath()
	if err != nil {
		return "", nil
	}
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() || info.Mode().Perm() != 0600 || info.Size() > 4096 {
		return "", fmt.Errorf("saved provider key must be a private regular file of at most 4096 bytes")
	}
	b, err := readConfig(path)
	return string(b), err
}

func (c Config) SaveProviderKey(key string) error {
	path, err := c.credentialPath()
	if err != nil {
		return err
	}
	if key == "" {
		err = os.Remove(path)
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if len(key) > 4096 || strings.ContainsAny(key, "\r\n\x00") {
		return fmt.Errorf("invalid provider key")
	}
	return writePrivate(path, key)
}

// CheckProjectCredentialBoundary keeps saved keys outside the project tools'
// readable root. The project may be anywhere else, including under the home dir.
func CheckProjectCredentialBoundary(project string) error {
	path, err := UserConfigPath()
	if err != nil {
		return nil
	}
	dir := filepath.Join(filepath.Dir(path), "credentials")
	within := func(base, target string) bool {
		rel, err := filepath.Rel(base, target)
		return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
	}
	if resolved, err := filepath.EvalSymlinks(project); err == nil {
		project = resolved
	}
	if within(project, dir) || within(dir, project) {
		return fmt.Errorf("choose a project directory outside the GoHarness credential storage scope")
	}
	if resolved, err := filepath.EvalSymlinks(dir); err == nil && (within(project, resolved) || within(resolved, project)) {
		return fmt.Errorf("choose a project directory outside the GoHarness credential storage scope")
	}
	return nil
}
