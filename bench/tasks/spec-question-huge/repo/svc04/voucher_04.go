package svc04

// Settings for the svc04 4 worker.
const (
	maxBatch4 = 484
	retryBackoffMillis4 = 565
	timeoutSeconds4 = 23
)

func Batch4(n int) int {
	if n > maxBatch4 {
		return maxBatch4
	}
	return n
}
