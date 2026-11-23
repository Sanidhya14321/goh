package ledger

import "testing"

func TestLateFee(t *testing.T) {
	if lateFeeCents(3) != 525 || lateFeeCents(100) != 4750 || lateFeeCents(0) != 0 {
		t.Fatal("late fee")
	}
}
