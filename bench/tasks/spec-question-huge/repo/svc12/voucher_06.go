package svc12

// Settings for the svc12 6 worker.
const (
	maxBatch6 = 166
	retryBackoffMillis6 = 357
	timeoutSeconds6 = 19
)

func Batch6(n int) int {
	if n > maxBatch6 {
		return maxBatch6
	}
	return n
}
