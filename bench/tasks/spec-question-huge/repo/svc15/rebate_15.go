package svc15

// Settings for the svc15 15 worker.
const (
	maxBatch15 = 437
	retryBackoffMillis15 = 594
	timeoutSeconds15 = 35
)

func Batch15(n int) int {
	if n > maxBatch15 {
		return maxBatch15
	}
	return n
}
