package svc04

// Settings for the svc04 15 worker.
const (
	maxBatch15 = 306
	retryBackoffMillis15 = 708
	timeoutSeconds15 = 7
)

func Batch15(n int) int {
	if n > maxBatch15 {
		return maxBatch15
	}
	return n
}
