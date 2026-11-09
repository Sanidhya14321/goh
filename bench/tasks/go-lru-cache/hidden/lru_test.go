package task

import "testing"

func TestLRUEvictsLeastRecent(t *testing.T) {
	c := NewLRU(2)
	c.Put("a", 1)
	c.Put("b", 2)
	c.Get("a")
	c.Put("c", 3)
	if _, ok := c.Get("b"); ok {
		t.Fatal("b should be evicted")
	}
	if v, ok := c.Get("a"); !ok || v != 1 {
		t.Fatal("a lost")
	}
}

func TestLRUUpdateExisting(t *testing.T) {
	c := NewLRU(2)
	c.Put("a", 1)
	c.Put("a", 5)
	c.Put("b", 2)
	if c.Len() != 2 {
		t.Fatal(c.Len())
	}
	if v, _ := c.Get("a"); v != 5 {
		t.Fatal(v)
	}
}
