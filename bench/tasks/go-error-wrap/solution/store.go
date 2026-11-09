package task

import (
	"errors"
	"fmt"
)

// ErrNotFound reports a missing record.
var ErrNotFound = errors.New("not found")

var users = map[string]string{"u1": "Ada"}

// LoadUser returns a user's name.
func LoadUser(id string) (string, error) {
	if id == "" {
		return "", errors.New("empty id")
	}
	name, ok := users[id]
	if !ok {
		return "", fmt.Errorf("user %s: %w", id, ErrNotFound)
	}
	return name, nil
}
