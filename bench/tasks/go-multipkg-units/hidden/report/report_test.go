package report

import "testing"

func TestLine(t *testing.T) {
	if got := Line("Oslo", -40); got != "Oslo: -40F" {
		t.Fatal(got)
	}
}
