package tools

import (
	"path/filepath"
	"strings"
)

// minPackageTimeout is the shortest limit a system package manager gets:
// stopping one midway (for example dpkg while it unpacks) leaves the machine's
// package database broken for every later command, which is worse than waiting.
const minPackageTimeout = 300

var packageManagers = map[string]bool{
	"apt": true, "apt-get": true, "aptitude": true, "dpkg": true,
	"yum": true, "dnf": true, "microdnf": true, "rpm": true, "zypper": true,
	"apk": true, "pacman": true, "emerge": true,
}

// runsPackageManager reports whether a command line invokes a system package
// manager anywhere in it (after sudo, env, a pipe or a list operator).
func runsPackageManager(command string) bool {
	words := strings.FieldsFunc(command, func(r rune) bool {
		switch r {
		case ' ', '\t', '\n', ';', '&', '|', '(', ')', '`', '"', '\'':
			return true
		}
		return false
	})
	for _, w := range words {
		if packageManagers[filepath.Base(w)] {
			return true
		}
	}
	return false
}
