package svc04

// Settings for the svc04 6 worker.
const (
	maxBatch6 = 348
	retryBackoffMillis6 = 835
	timeoutSeconds6 = 36
)

func Batch6(n int) int {
	if n > maxBatch6 {
		return maxBatch6
	}
	return n
}
