package task

import "errors"

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
		return "", errors.New("lookup failed")
	}
	return name, nil
}
