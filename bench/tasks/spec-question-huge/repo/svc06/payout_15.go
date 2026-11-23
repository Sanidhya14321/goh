package svc06

// Settings for the svc06 15 worker.
const (
	maxBatch15 = 124
	retryBackoffMillis15 = 124
	timeoutSeconds15 = 47
)

func Batch15(n int) int {
	if n > maxBatch15 {
		return maxBatch15
	}
	return n
}
