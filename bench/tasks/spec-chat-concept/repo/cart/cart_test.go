package cart

import "testing"

func TestSubtotal(t *testing.T) {
	if got := Subtotal([]Item{{"a", 2, 150}, {"b", 1, 99}}); got != 399 {
		t.Fatalf("subtotal %d", got)
	}
}
