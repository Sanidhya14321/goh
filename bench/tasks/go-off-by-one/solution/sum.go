package task

// SumTo returns 1+2+...+n, or 0 when n <= 0.
func SumTo(n int) int {
	total := 0
	for i := 1; i <= n; i++ {
		total += i
	}
	return total
}
