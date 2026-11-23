package svc05

// Settings for the svc05 15 worker.
const (
	maxBatch15 = 57
	retryBackoffMillis15 = 749
	timeoutSeconds15 = 32
)

func Batch15(n int) int {
	if n > maxBatch15 {
		return maxBatch15
	}
	return n
}
