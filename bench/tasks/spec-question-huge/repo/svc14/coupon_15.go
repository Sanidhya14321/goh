package svc14

// Settings for the svc14 15 worker.
const (
	maxBatch15 = 56
	retryBackoffMillis15 = 581
	timeoutSeconds15 = 27
)

func Batch15(n int) int {
	if n > maxBatch15 {
		return maxBatch15
	}
	return n
}
