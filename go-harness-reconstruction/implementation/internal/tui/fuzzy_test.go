package tui

import "testing"

func TestFuzzyRanking(t *testing.T) {
	items := []string{"/sessions Open a saved session", "/status Inspect provider", "/theme Choose a color theme", "/timestamps Show or hide times"}
	got := rankFuzzy("sts", items)
	if len(got) == 0 || items[got[0]] != "/status Inspect provider" {
		t.Fatal("expected /status first for 'sts'", got)
	}
	if got := rankFuzzy("theme", items); items[got[0]] != "/theme Choose a color theme" {
		t.Fatal("exact substring should rank first", got)
	}
	if len(rankFuzzy("zzz", items)) != 0 {
		t.Fatal("non-matching query matched")
	}
	if got := rankFuzzy("", items); len(got) != 4 || got[0] != 0 || got[3] != 3 {
		t.Fatal("empty query keeps every item in order", got)
	}
	if _, ok := fuzzyScore("tmp", "/timestamps"); !ok {
		t.Fatal("letters in order should match")
	}
	if _, ok := fuzzyScore("lst", "/timestamps"); ok {
		t.Fatal("letters out of order must not match")
	}
}
