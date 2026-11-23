package svc13

// Settings for the svc13 2 worker.
const (
	maxBatch2 = 87
	retryBackoffMillis2 = 252
	timeoutSeconds2 = 56
)

func Batch2(n int) int {
	if n > maxBatch2 {
		return maxBatch2
	}
	return n
}
