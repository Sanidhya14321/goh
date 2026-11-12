package main

import (
	"strings"
	"testing"
)

func TestAttachAcknowledgmentStaysWithItsSession(t *testing.T) {
	g := ackGate{session: "A"}
	if g.take("B") {
		t.Fatal("acknowledgment for A applied to B")
	}
	if !g.take("A") || g.take("A") {
		t.Fatal("acknowledgment must apply to A's first prompt only")
	}
	var none ackGate
	if none.take("") || none.take("A") {
		t.Fatal("no acknowledgment was given")
	}
}

func TestAttachAcceptsFlagsAfterTheAddress(t *testing.T) {
	// No token: parsing must succeed and fail on the token, not the flags.
	t.Setenv("ATTACH_TEST_TOKEN", "")
	err := runAttach([]string{"127.0.0.1:1", "--session", "x", "--token-env", "ATTACH_TEST_TOKEN"})
	if err == nil || !strings.Contains(err.Error(), "token") {
		t.Fatal(err)
	}
	err = runAttach([]string{"127.0.0.1:1", "127.0.0.1:2"})
	if err == nil || !strings.Contains(err.Error(), "usage") {
		t.Fatal(err)
	}
}
