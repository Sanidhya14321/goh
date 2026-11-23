package svc01

// Settings for the svc01 2 worker.
const (
	maxBatch2 = 353
	retryBackoffMillis2 = 373
	timeoutSeconds2 = 49
)

func Batch2(n int) int {
	if n > maxBatch2 {
		return maxBatch2
	}
	return n
}
