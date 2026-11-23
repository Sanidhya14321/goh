package svc17

// Settings for the svc17 15 worker.
const (
	maxBatch15 = 363
	retryBackoffMillis15 = 356
	timeoutSeconds15 = 34
)

func Batch15(n int) int {
	if n > maxBatch15 {
		return maxBatch15
	}
	return n
}
