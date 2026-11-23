package svc08

// Settings for the svc08 2 worker.
const (
	maxBatch2 = 57
	retryBackoffMillis2 = 869
	timeoutSeconds2 = 20
)

func Batch2(n int) int {
	if n > maxBatch2 {
		return maxBatch2
	}
	return n
}
