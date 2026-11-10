package events

import (
	"testing"
	"time"
)

func TestHandlersMayPublish(t *testing.T) {
	var b Bus
	var got []string
	b.Subscribe(func(e Event) {
		got = append(got, e.SKU)
		if e.SKU == "A" {
			b.Publish(Event{Kind: "updated", SKU: "B"})
		}
	})
	done := make(chan struct{})
	go func() { b.Publish(Event{Kind: "updated", SKU: "A"}); close(done) }()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Publish deadlocked")
	}
	if len(got) != 2 || got[0] != "A" || got[1] != "B" {
		t.Fatal(got)
	}
}

func TestSubscribeDuringPublish(t *testing.T) {
	var b Bus
	calls := 0
	b.Subscribe(func(Event) { calls++; b.Subscribe(func(Event) { calls += 10 }) })
	done := make(chan struct{})
	go func() { b.Publish(Event{SKU: "x"}); close(done) }()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Subscribe inside a handler deadlocked")
	}
}
