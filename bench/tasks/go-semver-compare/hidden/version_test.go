package task

import "testing"

func TestCompare(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{{"1.10.0", "1.9.0", 1}, {"v2.0.0", "2.0.0", 0}, {"1.0.0-rc.1", "1.0.0", -1}, {"1.0.0-alpha", "1.0.0-beta", -1}, {"0.0.9", "0.0.10", -1}, {"3.1.4", "3.1.4", 0}}
	for _, c := range cases {
		if got := Compare(c.a, c.b); got != c.want {
			t.Errorf("Compare(%s,%s)=%d want %d", c.a, c.b, got, c.want)
		}
	}
}
