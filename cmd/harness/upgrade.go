package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	goruntime "runtime"
	"strings"
	"time"
)

const releaseRepo = "Sanidhya14321/goh"

// Upgrade sources; tests point them at a local server. GOH_BASE_URL (as for
// the installers) names a directory holding the binaries and SHA256SUMS.
var (
	releaseAPI      = "https://api.github.com/repos/" + releaseRepo + "/releases/latest"
	releaseDownload = "https://github.com/" + releaseRepo + "/releases/download/"
)

const (
	maxBinary = 256 << 20
	maxSums   = 1 << 20
)

// runUpgrade implements `goh upgrade`: it replaces this executable with a
// published release after checking the release's SHA256SUMS entry and that
// the new binary runs here. The existing executable is left untouched on
// any failure.
func runUpgrade(args []string, w io.Writer) error {
	f := flag.NewFlagSet("upgrade", flag.ContinueOnError)
	check := f.Bool("check", false, "only report whether a newer release exists")
	tag := f.String("version", os.Getenv("GOH_VERSION"), "release tag to install (default: latest)")
	force := f.Bool("force", false, "reinstall even when the version is unchanged")
	if err := f.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if f.NArg() != 0 {
		return errors.New("usage: goh upgrade [--check] [--version TAG] [--force]")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	base := os.Getenv("GOH_BASE_URL")
	if base == "" {
		if *tag == "" || *tag == "latest" {
			latest, err := latestTag(ctx)
			if err != nil {
				return err
			}
			*tag = latest
		}
		if sameVersion(*tag, version) && !*force {
			fmt.Fprintf(w, "goh %s is up to date.\n", version)
			return nil
		}
		if *check {
			fmt.Fprintf(w, "goh %s is available (this is %s). Run goh upgrade to install it.\n", strings.TrimPrefix(*tag, "v"), version)
			return nil
		}
		base = releaseDownload + url.PathEscape(*tag)
	} else if *check {
		fmt.Fprintf(w, "goh %s; releases are read from %s.\n", version, base)
		return nil
	}

	exe, err := os.Executable()
	if err == nil {
		exe, err = filepath.EvalSymlinks(exe)
	}
	if err != nil {
		return err
	}
	newVersion, err := upgradeExecutable(ctx, exe, strings.TrimRight(base, "/"), *force)
	if err != nil {
		return err
	}
	if newVersion == "" {
		fmt.Fprintf(w, "goh %s is up to date.\n", version)
		return nil
	}
	fmt.Fprintf(w, "Upgraded %s from %s to %s.\n", exe, version, newVersion)
	return nil
}

func sameVersion(tag, current string) bool {
	return strings.TrimPrefix(tag, "v") == strings.TrimPrefix(current, "v")
}

// allowedSource keeps downloads on HTTPS, or plain HTTP to this machine.
func allowedSource(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return err
	}
	host := u.Hostname()
	if u.Scheme == "https" || u.Scheme == "http" && (host == "localhost" || net.ParseIP(host).IsLoopback()) {
		return nil
	}
	return fmt.Errorf("refusing to download from %s: HTTPS (or a loopback address) is required", raw)
}

func fetch(ctx context.Context, raw string, limit int64) ([]byte, error) {
	if err := allowedSource(raw); err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, raw, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "goh-upgrade")
	client := &http.Client{CheckRedirect: func(r *http.Request, via []*http.Request) error {
		if len(via) >= 10 {
			return errors.New("too many redirects")
		}
		return allowedSource(r.URL.String())
	}}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download failed: %s returned %s", raw, resp.Status)
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(b)) > limit {
		return nil, fmt.Errorf("%s is larger than expected", raw)
	}
	return b, nil
}

func latestTag(ctx context.Context) (string, error) {
	b, err := fetch(ctx, releaseAPI, 1<<20)
	if err != nil {
		return "", fmt.Errorf("could not find the latest release: %w", err)
	}
	var r struct {
		Tag string `json:"tag_name"`
	}
	if json.Unmarshal(b, &r) != nil || r.Tag == "" || strings.ContainsAny(r.Tag, "/\\ \t\r\n") {
		return "", errors.New("could not find the latest release: unexpected reply")
	}
	return r.Tag, nil
}

func releaseArtifact() string {
	name := "goh-" + goruntime.GOOS + "-" + goruntime.GOARCH
	if goruntime.GOOS == "windows" {
		name += ".exe"
	}
	return name
}

// expectedDigest finds the single SHA256SUMS line for name.
func expectedDigest(sums []byte, name string) (string, error) {
	var found []string
	sc := bufio.NewScanner(bytes.NewReader(sums))
	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		if len(fields) == 2 && (fields[1] == name || fields[1] == "*"+name) {
			found = append(found, strings.ToLower(fields[0]))
		}
	}
	switch {
	case len(found) == 0:
		return "", fmt.Errorf("no checksum for %s in SHA256SUMS", name)
	case len(found) > 1:
		return "", fmt.Errorf("ambiguous checksum entries for %s", name)
	case len(found[0]) != 64:
		return "", fmt.Errorf("malformed checksum for %s", name)
	}
	return found[0], nil
}

// probeVersion runs a downloaded binary's `version` command; tests replace it.
var probeVersion = func(ctx context.Context, path string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, path, "version").CombinedOutput()
	text := strings.TrimSpace(string(out))
	if err != nil {
		return "", fmt.Errorf("the downloaded goh does not run on this system (%v): %s", err, text)
	}
	return text, nil
}

// upgradeExecutable downloads, verifies and installs the release at base
// over exe. It returns "" when the release is this version (and !force).
func upgradeExecutable(ctx context.Context, exe, base string, force bool) (string, error) {
	name := releaseArtifact()
	sums, err := fetch(ctx, base+"/SHA256SUMS", maxSums)
	if err != nil {
		return "", err
	}
	want, err := expectedDigest(sums, name)
	if err != nil {
		return "", err
	}
	bin, err := fetch(ctx, base+"/"+name, maxBinary)
	if err != nil {
		return "", err
	}
	got := sha256.Sum256(bin)
	if hex.EncodeToString(got[:]) != want {
		return "", fmt.Errorf("checksum mismatch for %s (expected %s, got %x); nothing was changed", name, want, got)
	}

	dir := filepath.Dir(exe)
	tmp, err := os.CreateTemp(dir, ".goh-upgrade-*")
	if err != nil {
		return "", fmt.Errorf("cannot write next to %s (%v); reinstall with the installer instead", exe, err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	_, err = tmp.Write(bin)
	if err == nil {
		err = tmp.Sync()
	}
	if cerr := tmp.Close(); err == nil {
		err = cerr
	}
	if err == nil {
		err = os.Chmod(tmpName, 0755)
	}
	if err != nil {
		return "", err
	}
	if goruntime.GOOS == "windows" {
		// Windows runs only files named .exe.
		renamed := tmpName + ".exe"
		if err := os.Rename(tmpName, renamed); err != nil {
			return "", err
		}
		tmpName = renamed
		defer os.Remove(tmpName)
	}
	newVersion, err := probeVersion(ctx, tmpName)
	if err != nil {
		return "", err
	}
	if sameVersion(newVersion, version) && !force {
		return "", nil
	}
	if err := replaceExecutable(exe, tmpName); err != nil {
		return "", err
	}
	return newVersion, nil
}

// replaceExecutable moves next over exe. Windows cannot replace a running
// program but can rename it, so the old one is kept as exe.old until the
// next upgrade.
func replaceExecutable(exe, next string) error {
	if goruntime.GOOS != "windows" {
		return os.Rename(next, exe)
	}
	old := exe + ".old"
	_ = os.Remove(old)
	if err := os.Rename(exe, old); err != nil {
		return err
	}
	if err := os.Rename(next, exe); err != nil {
		if rerr := os.Rename(old, exe); rerr != nil {
			return fmt.Errorf("%v; the previous goh is at %s", err, old)
		}
		return err
	}
	return nil
}
