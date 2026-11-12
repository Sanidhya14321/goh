package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// releaseServer serves a release directory: one binary and SHA256SUMS.
func releaseServer(t *testing.T, body []byte, sums string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/latest":
			fmt.Fprint(w, `{"tag_name":"v9.9.9"}`)
		case "/download/v9.9.9/SHA256SUMS", "/SHA256SUMS":
			fmt.Fprint(w, sums)
		case "/download/v9.9.9/" + releaseArtifact(), "/" + releaseArtifact():
			w.Write(body)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func fakeProbe(t *testing.T, reported string) {
	t.Helper()
	old := probeVersion
	probeVersion = func(_ context.Context, path string) (string, error) {
		b, err := os.ReadFile(path)
		if err != nil || !bytes.HasPrefix(b, []byte("NEW BINARY")) {
			return "", fmt.Errorf("probe ran the wrong file")
		}
		return reported, nil
	}
	t.Cleanup(func() { probeVersion = old })
}

func installed(t *testing.T) string {
	t.Helper()
	exe := filepath.Join(t.TempDir(), "goh")
	if err := os.WriteFile(exe, []byte("OLD BINARY"), 0755); err != nil {
		t.Fatal(err)
	}
	return exe
}

func TestUpgradeReplacesTheExecutable(t *testing.T) {
	body := []byte("NEW BINARY")
	sum := sha256.Sum256(body)
	srv := releaseServer(t, body, fmt.Sprintf("%x  %s\n%x  other\n", sum, releaseArtifact(), sum))
	fakeProbe(t, "9.9.9")
	exe := installed(t)
	got, err := upgradeExecutable(context.Background(), exe, srv.URL, false)
	if err != nil || got != "9.9.9" {
		t.Fatal(got, err)
	}
	if b, _ := os.ReadFile(exe); string(b) != "NEW BINARY" {
		t.Fatalf("executable is %q", b)
	}
	if entries, _ := os.ReadDir(filepath.Dir(exe)); len(entries) > 2 {
		t.Fatal("temporary files left:", len(entries))
	}
}

// Nothing changes on a checksum mismatch, a missing or duplicated entry,
// or a binary that does not run.
func TestUpgradeRefusals(t *testing.T) {
	body := []byte("NEW BINARY")
	sum := sha256.Sum256(body)
	good := fmt.Sprintf("%x  %s\n", sum, releaseArtifact())
	for name, tc := range map[string]struct {
		body []byte
		sums string
		want string
	}{
		"tampered":  {append(body, '!'), good, "checksum mismatch"},
		"missing":   {body, fmt.Sprintf("%x  goh-plan9-mips\n", sum), "no checksum"},
		"ambiguous": {body, good + good, "ambiguous"},
		"broken":    {[]byte("NOT RUNNABLE"), fmt.Sprintf("%x  %s\n", sha256.Sum256([]byte("NOT RUNNABLE")), releaseArtifact()), "wrong file"},
	} {
		t.Run(name, func(t *testing.T) {
			srv := releaseServer(t, tc.body, tc.sums)
			fakeProbe(t, "9.9.9")
			exe := installed(t)
			if _, err := upgradeExecutable(context.Background(), exe, srv.URL, false); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatal(err)
			}
			if b, _ := os.ReadFile(exe); string(b) != "OLD BINARY" {
				t.Fatal("executable changed")
			}
			if entries, _ := os.ReadDir(filepath.Dir(exe)); len(entries) != 1 {
				t.Fatal("temporary files left")
			}
		})
	}
}

func TestUpgradeCommand(t *testing.T) {
	body := []byte("NEW BINARY")
	sum := sha256.Sum256(body)
	srv := releaseServer(t, body, fmt.Sprintf("%x  %s\n", sum, releaseArtifact()))
	oldAPI, oldDownload, oldVersion := releaseAPI, releaseDownload, version
	releaseAPI, releaseDownload = srv.URL+"/latest", srv.URL+"/download/"
	t.Cleanup(func() { releaseAPI, releaseDownload, version = oldAPI, oldDownload, oldVersion })
	t.Setenv("GOH_BASE_URL", "")
	t.Setenv("GOH_VERSION", "")

	version = "9.9.9"
	var out bytes.Buffer
	if err := runUpgrade(nil, &out); err != nil || !strings.Contains(out.String(), "up to date") {
		t.Fatal(out.String(), err)
	}
	version = "1.0.0"
	out.Reset()
	if err := runUpgrade([]string{"--check"}, &out); err != nil || !strings.Contains(out.String(), "9.9.9 is available") {
		t.Fatal(out.String(), err)
	}
	if err := allowedSource("http://example.com/goh"); err == nil {
		t.Fatal("plain HTTP to a remote host accepted")
	}
}

func TestReplaceExecutableKeepsOldOnWindowsOnly(t *testing.T) {
	exe := installed(t)
	next := filepath.Join(filepath.Dir(exe), "next")
	if err := os.WriteFile(next, []byte("NEW BINARY"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := replaceExecutable(exe, next); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(exe); string(b) != "NEW BINARY" {
		t.Fatal("not replaced")
	}
	_, err := os.Stat(exe + ".old")
	if (err == nil) != (os.PathSeparator == '\\') {
		t.Fatal("unexpected .old state:", err)
	}
}
