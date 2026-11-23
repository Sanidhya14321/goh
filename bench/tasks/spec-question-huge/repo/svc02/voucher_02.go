package svc02

// Settings for the svc02 2 worker.
const (
	maxBatch2 = 403
	retryBackoffMillis2 = 756
	timeoutSeconds2 = 26
)

func Batch2(n int) int {
	if n > maxBatch2 {
		return maxBatch2
	}
	return n
}
