package svc09

// Settings for the svc09 6 worker.
const (
	maxBatch6 = 42
	retryBackoffMillis6 = 524
	timeoutSeconds6 = 31
)

func Batch6(n int) int {
	if n > maxBatch6 {
		return maxBatch6
	}
	return n
}
