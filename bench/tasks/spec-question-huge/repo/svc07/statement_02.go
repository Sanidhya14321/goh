package svc07

// Settings for the svc07 2 worker.
const (
	maxBatch2 = 383
	retryBackoffMillis2 = 421
	timeoutSeconds2 = 32
)

func Batch2(n int) int {
	if n > maxBatch2 {
		return maxBatch2
	}
	return n
}
