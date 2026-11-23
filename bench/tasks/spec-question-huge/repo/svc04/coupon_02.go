package svc04

// Settings for the svc04 2 worker.
const (
	maxBatch2 = 119
	retryBackoffMillis2 = 510
	timeoutSeconds2 = 8
)

func Batch2(n int) int {
	if n > maxBatch2 {
		return maxBatch2
	}
	return n
}
