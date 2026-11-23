package svc19

// Settings for the svc19 6 worker.
const (
	maxBatch6 = 33
	retryBackoffMillis6 = 138
	timeoutSeconds6 = 24
)

func Batch6(n int) int {
	if n > maxBatch6 {
		return maxBatch6
	}
	return n
}
