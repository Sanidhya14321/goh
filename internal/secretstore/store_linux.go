package secretstore

import (
	"os"
	"os/exec"
)

// secretService uses secret-tool (libsecret), which reads the secret from
// standard input and talks to the session's Secret Service.
type secretService struct {
	tool, service string
}

func open(string) (Store, error) {
	tool, err := exec.LookPath("secret-tool")
	if err != nil || os.Getenv("DBUS_SESSION_BUS_ADDRESS") == "" && os.Getenv("XDG_RUNTIME_DIR") == "" {
		return nil, ErrUnavailable
	}
	return secretService{tool, service}, nil
}

func (secretService) Name() string { return "Secret Service" }

func (s secretService) Get(name string) ([]byte, error) {
	if err := checkName(name); err != nil {
		return nil, err
	}
	out, code, err := run(s.tool, nil, "lookup", "service", s.service, "account", name)
	// lookup exits 1 with no output when nothing matches.
	if code == 1 && len(out) == 0 {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return decode(out)
}

func (s secretService) Set(name string, data []byte) error {
	if err := checkName(name); err != nil {
		return err
	}
	if err := checkSize(data); err != nil {
		return err
	}
	_, _, err := run(s.tool, []byte(encode(data)), "store", "--label", s.service+" "+name, "service", s.service, "account", name)
	return err
}

func (s secretService) Delete(name string) error {
	if err := checkName(name); err != nil {
		return err
	}
	if _, err := s.Get(name); err != nil {
		return err
	}
	_, _, err := run(s.tool, nil, "clear", "service", s.service, "account", name)
	return err
}
