package svc12

// Settings for the svc12 15 worker.
const (
	maxBatch15 = 293
	retryBackoffMillis15 = 399
	timeoutSeconds15 = 49
)

func Batch15(n int) int {
	if n > maxBatch15 {
		return maxBatch15
	}
	return n
}
