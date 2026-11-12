package secretstore

import (
	"fmt"
	"os/exec"
)

const securityTool = "/usr/bin/security"

// keychain uses the security tool. Writes go through its interactive mode
// on standard input, so the secret never appears in a process listing.
type keychain struct{ service string }

func open(string) (Store, error) {
	if _, err := exec.LookPath(securityTool); err != nil {
		return nil, ErrUnavailable
	}
	return keychain{service}, nil
}

func (keychain) Name() string { return "macOS Keychain" }

func (k keychain) Get(name string) ([]byte, error) {
	if err := checkName(name); err != nil {
		return nil, err
	}
	out, code, err := run(securityTool, nil, "find-generic-password", "-s", k.service, "-a", name, "-w")
	if code == 44 {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return decode(out)
}

func (k keychain) Set(name string, data []byte) error {
	if err := checkName(name); err != nil {
		return err
	}
	if err := checkSize(data); err != nil {
		return err
	}
	// security's interactive mode cuts lines at about 4 KiB and runs the rest
	// as another command; refuse rather than store a truncated value.
	if len(data) > 2048 {
		return fmt.Errorf("the macOS Keychain helper accepts at most 2048 bytes")
	}
	line := fmt.Sprintf("add-generic-password -U -s %s -a %s -l \"%s %s\" -w %s\n", k.service, name, k.service, name, encode(data))
	_, _, err := run(securityTool, []byte(line), "-i")
	return err
}

func (k keychain) Delete(name string) error {
	if err := checkName(name); err != nil {
		return err
	}
	_, code, err := run(securityTool, nil, "delete-generic-password", "-s", k.service, "-a", name)
	if code == 44 {
		return ErrNotFound
	}
	return err
}
