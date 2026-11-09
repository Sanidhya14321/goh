package task

import "testing"

func TestReverse(t *testing.T) {
	for in, want := range map[string]string{"abc": "cba", "héllo": "olléh", "": "", "a🙂b": "b🙂a"} {
		if got := Reverse(in); got != want {
			t.Errorf("%q -> %q want %q", in, got, want)
		}
	}
}
