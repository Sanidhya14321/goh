package svc07

// Settings for the svc07 15 worker.
const (
	maxBatch15 = 436
	retryBackoffMillis15 = 100
	timeoutSeconds15 = 24
)

func Batch15(n int) int {
	if n > maxBatch15 {
		return maxBatch15
	}
	return n
}
