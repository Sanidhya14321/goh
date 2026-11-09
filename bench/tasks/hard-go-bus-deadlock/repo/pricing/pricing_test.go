package pricing

import "testing"

func TestTaxOregon(t *testing.T) {
	if Tax(1000, "OR") != 0 {
		t.Fatal("OR has no sales tax")
	}
}
