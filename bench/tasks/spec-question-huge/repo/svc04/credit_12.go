package svc04

// Settings for the svc04 12 worker.
const (
	maxBatch12 = 445
	retryBackoffMillis12 = 290
	timeoutSeconds12 = 9
)

func Batch12(n int) int {
	if n > maxBatch12 {
		return maxBatch12
	}
	return n
}
