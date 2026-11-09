package task

import "testing"

func TestSumTo(t *testing.T) {
	for n, want := range map[int]int{-3: 0, 0: 0, 1: 1, 4: 10, 100: 5050} {
		if got := SumTo(n); got != want {
			t.Errorf("SumTo(%d)=%d want %d", n, got, want)
		}
	}
}
