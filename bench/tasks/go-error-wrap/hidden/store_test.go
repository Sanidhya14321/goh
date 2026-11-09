package task

import (
	"errors"
	"strings"
	"testing"
)

func TestLoadUser(t *testing.T) {
	if n, err := LoadUser("u1"); err != nil || n != "Ada" {
		t.Fatal(n, err)
	}
	_, err := LoadUser("u9")
	if !errors.Is(err, ErrNotFound) || !strings.Contains(err.Error(), "u9") {
		t.Fatalf("%v", err)
	}
	if _, err := LoadUser(""); err == nil || errors.Is(err, ErrNotFound) {
		t.Fatalf("empty id: %v", err)
	}
}
