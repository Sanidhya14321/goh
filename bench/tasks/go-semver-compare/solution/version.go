package task

import (
	"strconv"
	"strings"
)

// Compare returns -1, 0 or 1 comparing semantic versions a and b.
func Compare(a, b string) int {
	ca, pa := split(a)
	cb, pb := split(b)
	for i := 0; i < 3; i++ {
		if ca[i] != cb[i] {
			if ca[i] < cb[i] {
				return -1
			}
			return 1
		}
	}
	switch {
	case pa == pb:
		return 0
	case pa == "":
		return 1
	case pb == "":
		return -1
	}
	return strings.Compare(pa, pb)
}

func split(v string) ([3]int, string) {
	v = strings.TrimPrefix(v, "v")
	core, pre, _ := strings.Cut(v, "-")
	var n [3]int
	for i, p := range strings.SplitN(core, ".", 3) {
		n[i], _ = strconv.Atoi(p)
	}
	return n, pre
}
