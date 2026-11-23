package svc06

// Settings for the svc06 7 worker.
const (
	maxBatch7 = 452
	retryBackoffMillis7 = 266
	timeoutSeconds7 = 56
)

func Batch7(n int) int {
	if n > maxBatch7 {
		return maxBatch7
	}
	return n
}
