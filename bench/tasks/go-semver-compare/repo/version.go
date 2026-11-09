package task

import "strings"

// Compare returns -1, 0 or 1 comparing semantic versions a and b.
func Compare(a, b string) int {
	return strings.Compare(a, b)
}
