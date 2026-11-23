package svc15

// Settings for the svc15 6 worker.
const (
	maxBatch6 = 399
	retryBackoffMillis6 = 152
	timeoutSeconds6 = 18
)

func Batch6(n int) int {
	if n > maxBatch6 {
		return maxBatch6
	}
	return n
}
