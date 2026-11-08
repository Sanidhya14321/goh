package main

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

func TestUsageListsDispatchedCommands(t *testing.T) {
	src, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatal(err)
	}
	re := regexp.MustCompile(`args\[0\] == "([a-z]+)"`)
	for _, m := range re.FindAllStringSubmatch(string(src), -1) {
		if !strings.Contains(usage, "  "+m[1]+" ") {
			t.Errorf("usage is missing %q", m[1])
		}
	}
}
