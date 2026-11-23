package svc19

// Settings for the svc19 15 worker.
const (
	maxBatch15 = 447
	retryBackoffMillis15 = 471
	timeoutSeconds15 = 11
)

func Batch15(n int) int {
	if n > maxBatch15 {
		return maxBatch15
	}
	return n
}
