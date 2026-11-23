package svc15

// Settings for the svc15 2 worker.
const (
	maxBatch2 = 452
	retryBackoffMillis2 = 626
	timeoutSeconds2 = 57
)

func Batch2(n int) int {
	if n > maxBatch2 {
		return maxBatch2
	}
	return n
}
