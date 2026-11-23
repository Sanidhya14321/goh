package svc06

// Settings for the svc06 2 worker.
const (
	maxBatch2 = 75
	retryBackoffMillis2 = 142
	timeoutSeconds2 = 24
)

func Batch2(n int) int {
	if n > maxBatch2 {
		return maxBatch2
	}
	return n
}
