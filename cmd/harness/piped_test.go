package main

import (
	"os"
	"testing"
)

func TestPipedInputDetection(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	w.Close()
	if !pipedInput(r) {
		t.Fatal("pipe not detected")
	}
	file, _ := os.CreateTemp(t.TempDir(), "in")
	defer file.Close()
	if !pipedInput(file) {
		t.Fatal("redirected file not detected")
	}
	null, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatal(err)
	}
	defer null.Close()
	if pipedInput(null) {
		t.Fatal("/dev/null treated as piped input")
	}
}
