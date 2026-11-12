// Package secretstore keeps small secrets in the operating system's
// credential store instead of a private file: the macOS Keychain, the
// freedesktop Secret Service (through secret-tool) on Linux, and DPAPI
// encryption bound to the user's Windows account. A secret is passed to
// helper programs on standard input, never on their command line.
package secretstore

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

// Service names GoHarness's entries in the OS store.
const Service = "GoHarness"

// service is the name new stores use; tests change it.
var service = Service

// MaxSecret bounds one stored value.
const MaxSecret = 1 << 16

var (
	// ErrNotFound reports that the store holds no entry under that name.
	ErrNotFound = errors.New("the OS credential store has no such entry")
	// ErrUnavailable reports that this system offers no usable store.
	ErrUnavailable = errors.New("no OS credential store is available")
)

// Store reads and writes named secrets.
type Store interface {
	Get(name string) ([]byte, error)
	Set(name string, data []byte) error
	Delete(name string) error
	// Name describes the store for people, e.g. "macOS Keychain".
	Name() string
}

// validName keeps names to characters that need no quoting anywhere.
var validName = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$`)

func checkName(name string) error {
	if !validName.MatchString(name) {
		return fmt.Errorf("invalid credential name %q", name)
	}
	return nil
}

// Open returns this system's store. dir is GoHarness's private credential
// folder; stores without their own database (Windows) keep encrypted
// entries there.
func Open(dir string) (Store, error) { return open(dir) }

const helperTimeout = 30 * time.Second

// run executes a helper program with input on standard input.
func run(name string, input []byte, args ...string) (stdout []byte, code int, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), helperTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdin = bytes.NewReader(input)
	var out, errOut bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errOut
	err = cmd.Run()
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		msg := strings.TrimSpace(errOut.String())
		if len(msg) > 200 {
			msg = msg[:200]
		}
		return out.Bytes(), exit.ExitCode(), fmt.Errorf("%s failed (exit %d): %s", name, exit.ExitCode(), msg)
	}
	return out.Bytes(), 0, err
}

// encode and decode carry arbitrary bytes through line-oriented helpers.
func encode(data []byte) string { return base64.StdEncoding.EncodeToString(data) }

func decode(text []byte) ([]byte, error) {
	b, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(text)))
	if err != nil {
		return nil, errors.New("the OS credential store returned an entry GoHarness did not write")
	}
	return b, nil
}

func checkSize(data []byte) error {
	if len(data) == 0 || len(data) > MaxSecret {
		return fmt.Errorf("credential must be 1 to %d bytes", MaxSecret)
	}
	return nil
}
