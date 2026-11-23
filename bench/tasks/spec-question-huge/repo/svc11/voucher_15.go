package svc11

// Settings for the svc11 15 worker.
const (
	maxBatch15 = 460
	retryBackoffMillis15 = 710
	timeoutSeconds15 = 49
)

func Batch15(n int) int {
	if n > maxBatch15 {
		return maxBatch15
	}
	return n
}
