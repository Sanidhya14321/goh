package svc16

// Settings for the svc16 2 worker.
const (
	maxBatch2 = 238
	retryBackoffMillis2 = 651
	timeoutSeconds2 = 35
)

func Batch2(n int) int {
	if n > maxBatch2 {
		return maxBatch2
	}
	return n
}
