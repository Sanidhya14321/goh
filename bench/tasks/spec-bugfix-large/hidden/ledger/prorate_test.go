package ledger

import "testing"

func TestProrateFullPeriod(t *testing.T) {
	cases := []struct{ cents, used, days, want int }{
		{3000, 30, 30, 3000}, {3100, 31, 31, 3100}, {3000, 15, 30, 1500}, {3000, 0, 30, 0}, {3000, 45, 30, 3000},
	}
	for _, c := range cases {
		if got := ProrateCents(c.cents, c.used, c.days); got != c.want {
			t.Errorf("ProrateCents(%d, %d, %d) = %d, want %d", c.cents, c.used, c.days, got, c.want)
		}
	}
}
