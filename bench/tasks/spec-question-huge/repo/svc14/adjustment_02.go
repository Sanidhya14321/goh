package svc14

// Settings for the svc14 2 worker.
const (
	maxBatch2 = 314
	retryBackoffMillis2 = 866
	timeoutSeconds2 = 57
)

func Batch2(n int) int {
	if n > maxBatch2 {
		return maxBatch2
	}
	return n
}
