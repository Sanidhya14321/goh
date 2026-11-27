package core

import "testing"

func TestWebsearchSetting(t *testing.T) {
	ok := func(c Config) bool { return c.Validate() == nil }
	base := Defaults()
	base.Protocol, base.Model, base.ContextWindow = ChatGPTProtocol, "m", 200000
	if !ok(base) {
		t.Fatal(base.Validate())
	}
	c := base
	c.Websearch = "provider"
	if !ok(c) || !c.WebsearchEnabled() {
		t.Fatal("provider rejected")
	}
	c.Websearch = "google"
	if ok(c) {
		t.Fatal("unknown websearch value accepted")
	}
	c.Websearch, c.WebsearchURL = "provider", "https://search.example.com"
	if ok(c) {
		t.Fatal("both websearch and websearch_url accepted")
	}
	if base.WebsearchEnabled() {
		t.Fatal("enabled by default")
	}
}
