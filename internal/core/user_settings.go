package core

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"github.com/Sanidhya14321/goh/internal/platform"
	"net"
	"net/url"
	"os"
	"path/filepath"
	goruntime "runtime"
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
	if writeErr == nil {
		writeErr = f.Sync()
	}
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
	if c.EffectiveProtocol() == ChatGPTProtocol {
		return "", nil
	}
	env := c.CredentialEnvironment()
	if key := os.Getenv(env); key != "" {
		endpoint, err := c.ProviderEndpoint()
		if err != nil {
			return "", err
		}
		u, _ := url.Parse(endpoint)
		host := strings.ToLower(u.Hostname())
		// Windows environment names ignore case, so "openai_api_key" reads
		// the official key there and must be held to the same rule.
		name := env
		if goruntime.GOOS == "windows" {
			name = strings.ToUpper(env)
		}
		trusted := u.Scheme == "https" && (u.Port() == "" || u.Port() == "443") && ((name == "OPENAI_API_KEY" && host == "api.openai.com") ||
			(name == "ANTHROPIC_API_KEY" && host == "api.anthropic.com"))
		if trusted || (c.APIKeyEnv != "" && name != "OPENAI_API_KEY" && name != "ANTHROPIC_API_KEY") {
			if u.Scheme != "https" && host != "localhost" && !net.ParseIP(host).IsLoopback() {
				return "", fmt.Errorf("environment keys require HTTPS or a loopback URL")
			}
			return key, nil
		}
	}
	path, err := c.credentialPath()
	if err != nil {
		return "", nil
	}
	if inKeychain(path) {
		b, err := keychainSecret(path)
		return string(b), err
	}
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() || !platform.PrivateFile(info, 0600) || info.Size() > 4096 {
		return "", fmt.Errorf("saved provider key must be a private regular file of at most 4096 bytes")
	}
	b, err := readConfig(path)
	return string(b), err
}

func (c Config) SaveProviderKey(key string) error {
	if c.EffectiveProtocol() == ChatGPTProtocol {
		return fmt.Errorf("use goh auth login to authorize ChatGPT")
	}
	path, err := c.credentialPath()
	if err != nil {
		return err
	}
	if key == "" {
		return removeSecret(path)
	}
	if len(key) > 4096 || strings.ContainsAny(key, "\r\n\x00") {
		return fmt.Errorf("invalid provider key")
	}
	return saveSecret(path, []byte(key))
}

// ErrProjectHoldsKeys means the chosen project folder contains goh's saved
// keys (the home folder does, on every platform), so tools could read them.
var ErrProjectHoldsKeys = errors.New("goh can't work in this folder because goh's saved keys are stored inside it (your home folder holds them); open a project folder instead: cd into it and run goh, or pass --project PATH")

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
	// Compare both the paths as given and with symlinks resolved: either
	// may not exist yet, and temp and home folders are often symlinked.
	projects, dirs := []string{project}, []string{dir}
	if resolved, err := filepath.EvalSymlinks(project); err == nil {
		projects = append(projects, resolved)
	}
	if resolved, err := filepath.EvalSymlinks(dir); err == nil {
		dirs = append(dirs, resolved)
	}
	for _, p := range projects {
		for _, d := range dirs {
			if within(p, d) || within(d, p) {
				return ErrProjectHoldsKeys
			}
		}
	}
	return nil
}
