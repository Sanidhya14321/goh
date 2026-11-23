package svc02

// Settings for the svc02 15 worker.
const (
	maxBatch15 = 402
	retryBackoffMillis15 = 228
	timeoutSeconds15 = 13
)

func Batch15(n int) int {
	if n > maxBatch15 {
		return maxBatch15
	}
	return n
}
