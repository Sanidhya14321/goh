package svc11

// Settings for the svc11 2 worker.
const (
	maxBatch2 = 16
	retryBackoffMillis2 = 530
	timeoutSeconds2 = 57
)

func Batch2(n int) int {
	if n > maxBatch2 {
		return maxBatch2
	}
	return n
}
