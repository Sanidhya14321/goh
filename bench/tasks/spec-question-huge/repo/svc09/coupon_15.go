package svc09

// Settings for the svc09 15 worker.
const (
	maxBatch15 = 24
	retryBackoffMillis15 = 498
	timeoutSeconds15 = 26
)

func Batch15(n int) int {
	if n > maxBatch15 {
		return maxBatch15
	}
	return n
}
